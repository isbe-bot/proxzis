import test from 'node:test';
import assert from 'node:assert/strict';
import * as THREE from 'three';
import { StudioCamera } from '../src/camera.js';
import { StudioRig } from '../src/rig.js';
import { RoundResult } from '../src/result.js';
import { buildEngagement, applyCombatFrame } from '../src/combat.js';

const aspects = [0.5, 1, 720 / 660, 1.2, 2.4];
const smooth = t => t * t * (3 - 2 * t);
const out = t => 1 - (1 - t) ** 3;
const inward = t => t ** 3;

// Every mesh is enclosed, including shoe soles, helmet shell, sword guard,
// flexible blade segments and the glowing tip. Bone origins miss those edges.
function meshExtrema(rigs) {
  const meshes = [];
  for (const rig of rigs) rig.group.traverse(mesh => {
    if (!mesh.isMesh) return;
    mesh.geometry.computeBoundingBox();
    const { min, max } = mesh.geometry.boundingBox;
    const corners = [];
    for (const x of [min.x, max.x]) for (const y of [min.y, max.y]) for (const z of [min.z, max.z]) {
      corners.push(new THREE.Vector3(x, y, z));
    }
    meshes.push({ mesh, corners });
  });
  return meshes;
}

function timeline(rigs, stances, parry) {
  const capture = types => rigs.map((rig, i) => rig.capture(types[i] === 'PARRY' ? parry : types[i]));
  const guard = capture(['ON_GUARD', 'ON_GUARD']);
  const salute = capture(['SALUTE', 'SALUTE']);
  const windup = capture(stances.map(s => s === 'STRIKE' ? 'WINDUP' : 'ON_GUARD'));
  const { action, contact, winner, kind } = buildEngagement(rigs, stances, parry);
  const counter = winner !== null && stances[winner] === 'FEINT';
  const bait = capture(stances.map(s => s === 'FEINT' ? 'FEINT_EXTEND' : s === 'STRIKE' ? 'WINDUP' : 'PARRY'));
  const react = capture(stances.map((s, i) => winner === null ? 'DEFLECT_REACT' : i === winner ? (s === 'PARRY' ? 'PARRY' : 'ON_GUARD') : stances[winner] === 'PARRY' ? 'DEFLECT_REACT' : 'HIT_REACT'));
  if (kind === 'body') {
    rigs[winner].applyInterpolation(contact[winner], guard[winner], 0.12);
    react[winner] = rigs[winner].snapshot();
    react[winner].feet = contact[winner].feet.map((x, j) => THREE.MathUtils.lerp(x, guard[winner].feet[j], 0.12));
  }
  const paths = [
    { from: guard, to: salute, phase: 'salute', ease: smooth },
    { from: salute, to: guard, phase: 'set', ease: out },
    { from: guard, to: windup, phase: 'windup', ease: out },
  ];
  if (stances.includes('FEINT')) {
    paths.push({ from: windup, to: bait, phase: 'bait', ease: out });
    paths.push({ from: bait, to: counter || winner === null ? action : contact, phase: 'slip', ease: counter || winner === null ? smooth : inward, disengage: true, drive: !counter && winner !== null });
    if (counter) paths.push({ from: action, to: contact, phase: 'riposte', ease: inward, disengage: true, drive: true });
  } else paths.push({ from: windup, to: contact, phase: 'commit', ease: stances.every(s => s === 'PARRY') ? smooth : inward, drive: stances.includes('STRIKE') });
  paths.push({ from: contact, to: winner === null && stances[0] !== 'STRIKE' ? guard : react, phase: 'rebound', ease: out });
  return { paths, winner, kind };
}

test('God/broadcast enclose actual mesh extrema through all 27 choreographies without excessive recording void', t => {
  const rigs = [new StudioRig(0x0044ff, true), new StudioRig(0x990000, false)];
  const scene = new THREE.Scene();
  rigs.forEach((rig, i) => { rig.group.position.x = i ? 1.1 : -1.1; scene.add(rig.group); });
  const meshes = meshExtrema(rigs);
  const point = new THREE.Vector3(), matrix = new THREE.Matrix4();
  const views = aspects.flatMap(aspect => ['god', 'broadcast'].map(preset => {
    const camera = new StudioCamera();
    camera.resize(aspect); camera.select(preset);
    return { camera: camera.camera, preset, aspect, min: [Infinity, Infinity], max: [-Infinity, -Infinity], occupancy: {}, frames: 0, worst: '' };
  }));
  function measure(view, phase, label) {
    const bounds = [Infinity, Infinity, -Infinity, -Infinity];
    for (const { mesh, corners } of meshes) {
      matrix.multiplyMatrices(view.camera.matrixWorldInverse, mesh.matrixWorld);
      matrix.premultiply(view.camera.projectionMatrix);
      for (const corner of corners) {
        point.copy(corner).applyMatrix4(matrix);
        assert.ok(point.z > -1 && point.z < 1, 'Mesh must lie between near/far planes');
        bounds[0] = Math.min(bounds[0], point.x); bounds[1] = Math.min(bounds[1], point.y);
        bounds[2] = Math.max(bounds[2], point.x); bounds[3] = Math.max(bounds[3], point.y);
      }
    }
    for (let axis = 0; axis < 2; axis++) {
      if (bounds[axis] < view.min[axis] || bounds[axis + 2] > view.max[axis]) view.worst = label;
      view.min[axis] = Math.min(view.min[axis], bounds[axis]);
      view.max[axis] = Math.max(view.max[axis], bounds[axis + 2]);
    }
    const occupancy = view.occupancy[phase] ??= { min: [Infinity, Infinity], max: [0, 0] };
    for (let axis = 0; axis < 2; axis++) {
      const size = (bounds[axis + 2] - bounds[axis]) / 2;
      occupancy.min[axis] = Math.min(occupancy.min[axis], size);
      occupancy.max[axis] = Math.max(occupancy.max[axis], size);
    }
    view.frames++;
  }
  for (const parry of ['PARRY', 'PARRY_QUARTE', 'PARRY_PRIME']) {
    for (const a of ['STRIKE', 'PARRY', 'FEINT']) for (const b of ['STRIKE', 'PARRY', 'FEINT']) {
      const stances = [a, b], label = `${parry}:${a}/${b}`;
      const { paths, winner, kind } = timeline(rigs, stances, parry);
      for (const path of paths) for (let frame = 0; frame <= 60; frame++) {
        const alpha = frame / 60;
        rigs.forEach((rig, i) => {
          applyCombatFrame(rig, path.from[i], path.to[i], alpha, path.ease(alpha), { disengage: path.disengage && stances[i] === 'FEINT', drive: path.drive });
          rig.animateBlade(alpha * 0.18, path.phase === 'rebound' ? (kind === 'blade' ? 0.065 : 0.025) * Math.exp(-alpha * 0.18 * 12) : 0);
          rig.group.updateMatrixWorld(true);
        });
        for (const view of views) measure(view, path.phase === 'salute' || path.phase === 'set' ? 'salute' : 'combat', `${label}:${path.phase}:${frame}`);
      }
      // Outcomes depend on camera heading. Sample each view's actual smooth
      // turn, raised victory sword, defeated sword, draw and return to guard.
      const rebound = rigs.map(rig => rig.snapshot());
      for (const view of views) {
        rigs.forEach((rig, i) => { rig.animateBlade(0, 0); rig.applyInterpolation(rebound[i], rebound[i], 0); });
        const result = new RoundResult(scene, rigs);
        result.show(winner, view.camera);
        for (const phase of ['outcome', 'recovery']) {
          if (phase === 'recovery') result.returnToGuard();
          for (let frame = 0; frame <= 60; frame++) {
            if (frame) result.update(result.duration / 60);
            scene.updateMatrixWorld(true);
            measure(view, phase, `${label}:${phase}:${winner}:${frame}`);
          }
        }
        result.clear();
      }
    }
  }
  for (const view of views) {
    const percent = values => values.map(x => +(x * 100).toFixed(1));
    t.diagnostic(JSON.stringify({ preset: view.preset, aspect: +view.aspect.toFixed(4), ndc: [...view.min, ...view.max].map(x => +x.toFixed(4)), occupancyPercent: Object.fromEntries(Object.entries(view.occupancy).map(([phase, span]) => [phase, { min: percent(span.min), max: percent(span.max) }])), frames: view.frames }));
  }
  for (const view of views) {
    assert.ok([...view.min, ...view.max].every(x => Math.abs(x) <= 0.96), `${view.preset} aspect=${view.aspect} clips mesh extrema: ${view.min}/${view.max}, ${view.worst}`);
    // Portrait is width-limited, landscape is height-limited. At least one
    // axis should be used substantially, rather than fitting an oversized box.
    assert.ok(Object.values(view.occupancy).some(span => Math.max(...span.max) > 0.8), `${view.preset}:${view.aspect} unnecessarily distant framing`);
    if (view.aspect === 720 / 660) {
      // Fixed God framing must also reserve the much taller victory sword;
      // its elevated angle foreshortens guard. These floors still require a
      // substantial improvement over the old 32% God / 41% broadcast height.
      const minHeight = view.preset === 'god' ? 0.40 : 0.51;
      for (const [phase, span] of Object.entries(view.occupancy)) {
        assert.ok(span.min[0] > 0.52 && span.min[1] > minHeight, `${view.preset}:${phase} excessive void: ${span.min}`);
      }
      assert.ok(view.occupancy.outcome.max[1] > (view.preset === 'god' ? 0.82 : 0.93), 'Raised victory swords should use the available frame height');
    }
  }
});
