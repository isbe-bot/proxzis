import * as THREE from 'three';

export const wins = (a, b) => (a === 'ROCK' && b === 'SCISSORS') || (a === 'SCISSORS' && b === 'PAPER') || (a === 'PAPER' && b === 'ROCK');
const position = object => object.getWorldPosition(new THREE.Vector3());
export function bladePoints(rig) {
  rig.group.updateMatrixWorld(true);
  return [...rig.bladeSegments.map(position), position(rig.bladeTip)];
}

// Closest points of two finite segments, including parallel/degenerate lines.
export function segmentContact(p, q, r, s) {
  const u = q.clone().sub(p), v = s.clone().sub(r), w = p.clone().sub(r);
  const a = u.dot(u), b = u.dot(v), c = v.dot(v), d = u.dot(w), e = v.dot(w);
  const clamp = THREE.MathUtils.clamp;
  let t = 0, h = 0;
  if (a < 1e-12) h = c < 1e-12 ? 0 : clamp(e / c, 0, 1);
  else if (c < 1e-12) t = clamp(-d / a, 0, 1);
  else {
    const denominator = a * c - b * b;
    t = denominator > 1e-12 ? clamp((b * e - c * d) / denominator, 0, 1) : 0;
    h = (b * t + e) / c;
    if (h < 0) { h = 0; t = clamp(-d / a, 0, 1); }
    else if (h > 1) { h = 1; t = clamp((b - d) / a, 0, 1); }
  }
  const first = p.clone().addScaledVector(u, t), second = r.clone().addScaledVector(v, h);
  return { point: first.clone().lerp(second, 0.5), first, second, distance: first.distanceTo(second), fractions: [t, h] };
}

// Inspect all eight actual flexing blade segments, not the tip midpoint or
// the straight chord of a curved blade. Shared by rendering and regression tests.
export function bladeContact(a, b) {
  const aa = bladePoints(a), bb = bladePoints(b);
  let nearest = { distance: Infinity };
  for (let i = 0; i < 8; i++) for (let j = 0; j < 8; j++) {
    const hit = segmentContact(aa[i], aa[i+1], bb[j], bb[j+1]);
    if (hit.distance < nearest.distance) nearest = { ...hit, fractions: [(i+hit.fractions[0])/8, (j+hit.fractions[1])/8] };
  }
  return nearest;
}

function snapshot(rig) {
  const snap = rig.snapshot();
  rig.group.updateMatrixWorld(true);
  const inverse = rig.group.matrixWorld.clone().invert();
  snap.feet = ['Lead', 'Rear'].map(side => position(rig.bones[`ankle${side}`]).applyMatrix4(inverse).x);
  return snap;
}

export function buildEngagement(rigs, stances, guardLine = 'PAPER') {
  const actions = ['ROCK', 'PAPER', 'SCISSORS'];
  if (stances.length !== 2 || stances.some(s => !actions.includes(s))) throw new Error('Invalid matchup');
  if (!['PAPER', 'PAPER_QUARTE', 'PAPER_PRIME'].includes(guardLine)) throw new Error('Invalid guard line');
  const tie = stances[0] === stances[1];
  const winner = tie ? null : wins(...stances) ? 0 : 1;
  const pose = (rig, type) => rig.pose(type === 'PAPER' ? guardLine : type);
  rigs.forEach((rig, i) => pose(rig, stances[i]));
  const action = rigs.map(snapshot);
  let kind = 'read', point = null;
  if (tie && stances[0] === 'SCISSORS') return { action, contact: action, winner, kind, point };
  if (tie) {
    kind = 'blade';
    const probe = stances[0] === 'PAPER';
    point = new THREE.Vector3(0, probe ? 1.94 : 1.55, 0);
    rigs.forEach((rig, i) => {
      if (!probe) {
        rig.pose('SCISSORS_EXTEND');
        rig.bones.pelvis.position.y = 0.94;
        rig.plant('Lead', 0.62); rig.plant('Rear', -0.40);
      }
      const sign = i ? -1 : 1;
      const direction = new THREE.Vector3(sign * (probe ? 0.85 : 0.72), probe ? 0.35 : 0.65, -sign * (probe ? 0.38 : 0.24)).normalize();
      const fraction = probe ? 0.90 : 0.60;
      rig.setBlade(point.clone().addScaledVector(direction, -0.88 * fraction), direction);
    });
  } else if (stances[winner] === 'PAPER') {
    kind = 'blade';
    const sign = winner ? 1 : -1; // Incoming attacker direction in world space.
    point = new THREE.Vector3(sign * 0.78, 1.60, guardLine === 'PAPER_QUARTE' ? -sign * 0.12 : 0);
    // A checked lunge stops 6 cm short; retain the straight rear drive leg.
    const attacker = rigs[1-winner];
    attacker.bones.pelvis.position.x -= 0.06;
    attacker.plant('Lead', 0.93); attacker.plant('Rear', -0.52);
    const incoming = new THREE.Vector3(sign, 0.015, -sign * 0.20).normalize();
    const low = guardLine === 'PAPER_PRIME';
    const outward = guardLine === 'PAPER_QUARTE' ? -0.35 : 0.35;
    const defending = new THREE.Vector3(-sign * 0.65, low ? -0.70 : 0.70, (low ? sign : -sign) * outward).normalize();
    rigs[1-winner].setBlade(point.clone().addScaledVector(incoming, -0.88 * 0.70), incoming);
    rigs[winner].setBlade(point.clone().addScaledVector(defending, -0.88 * 0.23), defending);
  } else {
    kind = 'body';
    // Scissors win only after a visible disengage and renewed extension.
    if (stances[winner] === 'SCISSORS') {
      rigs[winner].pose('ROCK');
      // The false guard commits the defending hand outside the renewed line.
      // Without this, the riposte passes through a stationary forearm first.
      rigs[1-winner].bones.shoulderLead.rotation.x = -0.65;
      rigs[1-winner].aimWeapon(0.45, guardLine === 'PAPER_PRIME' ? -0.8 : 0.72, 0.48);
    }
    const attacker = rigs[winner], defender = rigs[1-winner];
    defender.group.updateMatrixWorld(true);
    const shoulder = position(attacker.bones.shoulderLead);
    const target = defender.bones.chest.localToWorld(new THREE.Vector3(0.11, 0.055, 0));
    const direction = target.clone().sub(shoulder).normalize();
    const ray = new THREE.Raycaster(shoulder, direction, 0, 3);
    const surface = ray.intersectObjects([defender.chestShell, defender.chestOrb], false)[0];
    if (!surface) throw new Error('Thrust line misses target jacket');
    point = surface.point;
    attacker.setBlade(point.clone().addScaledVector(direction, -0.88), direction);
  }
  const contact = rigs.map(snapshot);
  return { action, contact, winner, kind, point };
}

// The same weight-transfer/disengage sampler is used by playback and tests.
export function applyCombatFrame(rig, from, to, t, alpha, { disengage = false, drive = false } = {}) {
  rig.applyInterpolation(from, to, alpha);
  const arc = Math.sin(Math.PI * t);
  const hipAlpha = drive ? Math.min(1, alpha + 0.08 * arc) : alpha;
  if (drive) rig.bones.pelvis.position.lerpVectors(from.pelvis.pos, to.pelvis.pos, hipAlpha);
  if (disengage) {
    // A withdrawn point needs wider wrist clearance than the driven riposte.
    rig.bones.weapon.rotateX(Math.sin(t * Math.PI * 2) * arc * (drive ? 0.34 : 0.50));
    rig.bones.weapon.rotateZ(-arc * 0.24);
  }
  if (drive && (disengage || to.pose === 'PAPER_PRIME')) {
    // Carry the point around the outside of the weapon arm before dropping
    // into prime / renewing the scissors thrust. A straight slerp can cross the forearm.
    rig.group.updateMatrixWorld(true);
    const direction = new THREE.Vector3(0, -1, 0).applyQuaternion(rig.bones.weapon.getWorldQuaternion(new THREE.Quaternion()));
    direction.applyQuaternion(rig.group.getWorldQuaternion(new THREE.Quaternion()).invert());
    direction.z += arc * (disengage ? 0.22 : 0.70);
    rig.aimWeapon(direction.x, direction.y, direction.z);
  }
  for (const [j, side] of ['Lead', 'Rear'].entries()) {
    const travel = Math.abs(to.feet[j] - from.feet[j]);
    const lift = travel > 0.10 ? arc * (j === 0 ? 0.045 : 0.025) : 0;
    rig.plant(side, THREE.MathUtils.lerp(from.feet[j], to.feet[j], hipAlpha), 0.075 + lift, rig.footYaws[j]);
  }
}
