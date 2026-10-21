import test from 'node:test';
import assert from 'node:assert/strict';
import * as THREE from 'three';
import { StudioRig } from '../src/rig.js';
import { RoundResult } from '../src/result.js';
import { buildEngagement, bladePoints, applyCombatFrame } from '../src/combat.js';

function bodyMeshes(rig) {
  const meshes = [];
  rig.group.traverse(object => {
    if (!object.isMesh) return;
    let parent = object;
    while (parent && parent !== rig.bones.weapon) parent = parent.parent;
    if (!parent) meshes.push(object);
  });
  return meshes;
}
function intersections(rig, meshes) {
  const points = bladePoints(rig), hits = [];
  for (let i = 0; i < 8; i++) {
    const direction = points[i+1].clone().sub(points[i]);
    const ray = new THREE.Raycaster(points[i], direction.clone().normalize(), 0, direction.length() - 1e-6);
    hits.push(...ray.intersectObjects(meshes, false));
  }
  return hits;
}
const setup = () => [true, false].map((facing, i) => {
  const rig = new StudioRig(i ? 0x990000 : 0x0044ff, facing);
  rig.group.position.x = i ? 1.1 : -1.1;
  return rig;
});
const smooth = t => t*t*(3-2*t);
const out = t => 1-(1-t)**3;

test('all 27 combat timelines keep blades outside both fighters, including outcome recovery', () => {
  const rigs = setup(), meshes = rigs.map(bodyMeshes), violations = [];
  let checks = 0;
  for (const guardLine of ['PAPER', 'PAPER_QUARTE', 'PAPER_PRIME']) {
    for (const a of ['ROCK', 'PAPER', 'SCISSORS']) for (const b of ['ROCK', 'PAPER', 'SCISSORS']) {
      const stances = [a, b];
      const capture = types => rigs.map((rig, i) => rig.capture(types[i] === 'PAPER' ? guardLine : types[i]));
      const guard = capture(['ON_GUARD', 'ON_GUARD']);
      const salute = capture(['SALUTE', 'SALUTE']);
      const windup = capture(stances.map(s => s === 'ROCK' ? 'WINDUP' : 'ON_GUARD'));
      const engagement = buildEngagement(rigs, stances, guardLine);
      const { action, contact, winner } = engagement;
      const counter = winner !== null && stances[winner] === 'SCISSORS';
      const bait = capture(stances.map(s => s === 'SCISSORS' ? 'SCISSORS_EXTEND' : s === 'ROCK' ? 'WINDUP' : 'PAPER'));
      const react = capture(stances.map((s, i) => winner === null ? 'DEFLECT_REACT' : i === winner ? (s === 'PAPER' ? 'PAPER' : 'ON_GUARD') : stances[winner] === 'PAPER' ? 'DEFLECT_REACT' : 'HIT_REACT'));
      if (engagement.kind === 'body') {
        rigs[winner].applyInterpolation(contact[winner], guard[winner], 0.12);
        react[winner] = rigs[winner].snapshot();
        react[winner].feet = contact[winner].feet.map((x, j) => THREE.MathUtils.lerp(x, guard[winner].feet[j], 0.12));
      }
      const paths = [
        { from: guard, to: salute, phase: 'salute', ease: smooth },
        { from: salute, to: guard, phase: 'set', ease: out },
        { from: guard, to: windup, phase: 'windup', ease: out },
      ];
      if (stances.includes('SCISSORS')) {
        paths.push({ from: windup, to: bait, phase: 'bait', ease: out });
        paths.push({ from: bait, to: counter || winner === null ? action : contact, phase: 'slip', ease: counter || winner === null ? smooth : t => t**3, disengage: true, drive: !counter && winner !== null });
        if (counter) paths.push({ from: action, to: contact, phase: 'riposte', ease: t => t**3, disengage: true, drive: true });
      } else paths.push({ from: windup, to: contact, phase: 'commit', ease: a === 'PAPER' && b === 'PAPER' ? smooth : t => t**3, drive: a === 'ROCK' || b === 'ROCK' });
      paths.push({ from: contact, to: winner === null && a !== 'ROCK' ? guard : react, phase: 'rebound', ease: out });
      for (const path of paths) for (let frame = 0; frame <= 240; frame++) {
        const t = frame / 240;
        rigs.forEach((rig, i) => applyCombatFrame(rig, path.from[i], path.to[i], t, path.ease(t), { disengage: path.disengage && stances[i] === 'SCISSORS', drive: path.drive }));
        rigs.forEach(rig => {
          rig.animateBlade(t*0.18, path.phase === 'rebound' ? (engagement.kind === 'blade' ? 0.065 : 0.025)*Math.exp(-t*0.18*12) : 0);
          rig.group.updateMatrixWorld(true);
        });
        rigs.forEach((rig, i) => {
          const own = intersections(rig, meshes[i]);
          if (own.length) violations.push(`${guardLine}:${a}/${b}:${path.phase}:${frame}:self:${i}:${own[0].object.parent.name}`);
          // No unintended opponent contact, including arms and recovery.
          // The flexible 9 mm point may press its intended jacket touch.
          {
            const torso = intersections(rig, meshes[1-i]);
            const tip = rig.bladeTip.getWorldPosition(new THREE.Vector3());
            const intendedTipTouch = engagement.kind === 'body' && i === winner && path.phase === 'rebound' && frame === 0 && torso.every(hit =>
              [rigs[1-i].chestShell, rigs[1-i].chestOrb].includes(hit.object) && hit.point.distanceTo(tip) < 0.009);
            if (torso.length && !intendedTipTouch) violations.push(`${guardLine}:${a}/${b}:${path.phase}:${frame}:unintended-body-hit:${i}:${torso[0].object.parent.name}`);
          }
          checks += 8;
          for (const side of ['Lead', 'Rear']) {
            const ankle = rig.bones[`ankle${side}`];
            const up = new THREE.Vector3(0, 1, 0).applyQuaternion(ankle.getWorldQuaternion(new THREE.Quaternion()));
            assert.ok(up.y > 0.999, 'No inverted/tilted ankles');
            assert.ok(ankle.getWorldPosition(new THREE.Vector3()).y >= 0.075 - 1e-6, 'No floor penetration');
          }
        });
      }
      rigs.forEach(rig => rig.animateBlade(0,0));
      const result = new RoundResult(new THREE.Scene(), rigs);
      result.show(winner);
      for (const phase of ['outcome', 'recovery']) {
        if (phase === 'recovery') result.returnToGuard();
        for (let frame = 0; frame < 61; frame++) {
          result.update(1/60);
          rigs.forEach((rig, i) => {
            const hits = intersections(rig, meshes[i]);
            if (hits.length) violations.push(`${guardLine}:${a}/${b}:${phase}:${frame}:self:${i}:${hits[0].object.parent.name}`);
            const rival = intersections(rig, meshes[1-i]);
            if (rival.length) violations.push(`${guardLine}:${a}/${b}:${phase}:${frame}:rival:${i}:${rival[0].object.parent.name}`);
          });
        }
      }
    }
  }
  assert.deepEqual(violations, []);
  assert.ok(checks > 150000);
});

test('all outcome transitions and recoveries keep blades outside own body meshes', () => {
  const rigs = setup(), meshes = rigs.map(bodyMeshes);
  const scene = new THREE.Scene(); rigs.forEach(rig => scene.add(rig.group));
  const result = new RoundResult(scene, rigs);
  for (const winner of [0, 1, null]) {
    result.clear(); rigs.forEach(rig => rig.pose('ON_GUARD'));
    result.show(winner);
    for (const phase of ['outcome', 'recovery']) {
      if (phase === 'recovery') result.returnToGuard();
      for (let frame = 0; frame < 61; frame++) {
        result.update(1/60);
        rigs.forEach((rig, i) => assert.equal(intersections(rig, meshes[i]).length, 0, `${winner}:${phase}:${frame}:${i}`));
      }
    }
  }
});
