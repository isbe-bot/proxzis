import test from 'node:test';
import assert from 'node:assert/strict';
import * as THREE from 'three';
import { StudioRig } from '../src/rig.js';
const poses = ['ON_GUARD', 'SALUTE', 'WINDUP', 'ROCK', 'FOLLOW_THROUGH', 'PAPER', 'PAPER_QUARTE', 'PAPER_PRIME', 'SCISSORS', 'SCISSORS_EXTEND', 'HIT_REACT', 'DEFLECT_REACT', 'VICTORY', 'DEFEAT', 'ATTENTION'];
function checkFeet(rig) {
  rig.group.updateMatrixWorld(true);
  for (const side of ['Lead', 'Rear']) {
    const ankle = rig.bones[`ankle${side}`];
    const p = ankle.getWorldPosition(new THREE.Vector3());
    assert.ok(Math.abs(p.y - 0.075) < 0.001, `${side} foot height ${p.y}`);
    const up = new THREE.Vector3(0, 1, 0).applyQuaternion(ankle.getWorldQuaternion(new THREE.Quaternion()));
    assert.ok(up.y > 0.999, 'Sole must be level');
  }
}
for (const facing of [true, false]) {
  const rig = new StudioRig(0x0044ff, facing);
  test(`all poses finite, feet planted, mirrored=${!facing}`, () => {
    assert.equal(Object.keys(rig.bones).length, 20);
    for (const pose of poses) {
      rig.pose(pose);
      checkFeet(rig);
      for (const bone of Object.values(rig.bones)) assert.ok([...bone.position.toArray(), ...bone.quaternion.toArray()].every(Number.isFinite));
    }
  });
  test(`every pose pair interpolates with foot contact, mirrored=${!facing}`, () => {
    for (const from of poses) for (const to of poses) {
      const a = rig.capture(from), b = rig.capture(to);
      for (let i = 0; i <= 10; i++) {
        rig.applyInterpolation(a, b, i / 10);
        checkFeet(rig);
      }
    }
  });
  test(`pose reset independent of previous action, mirrored=${!facing}`, () => {
    rig.pose('ON_GUARD');
    const expected = rig.snapshot();
    for (const pose of poses) {
      rig.pose(pose); rig.pose('ON_GUARD');
      for (const [name, bone] of Object.entries(rig.bones)) {
        assert.ok(bone.quaternion.angleTo(expected[name].quat) < 1e-6);
        assert.ok(bone.position.distanceTo(expected[name].pos) < 1e-6);
      }
    }
    assert.throws(() => rig.pose('INVALID'), /Unknown pose/);
  });
}

for (const facing of [true, false]) {
  const rig = new StudioRig(0x0044ff, facing);
  const localPoint = object => object.getWorldPosition(new THREE.Vector3()).applyMatrix4(rig.group.matrixWorld.clone().invert());
  test(`classical fencing constraints, mirrored=${!facing}`, () => {
    rig.pose('ON_GUARD');
    rig.group.updateMatrixWorld(true);
    const leadFoot = new THREE.Vector3(1, 0, 0).applyQuaternion(rig.bones.ankleLead.getWorldQuaternion(new THREE.Quaternion()));
    const rearFoot = new THREE.Vector3(1, 0, 0).applyQuaternion(rig.bones.ankleRear.getWorldQuaternion(new THREE.Quaternion()));
    assert.ok(Math.abs(leadFoot.dot(rearFoot)) < 1e-6, 'Guard feet must be perpendicular');
    assert.ok(localPoint(rig.bones.wristRear).y < localPoint(rig.bones.shoulderRear).y - 0.25, 'Rear hand must rest below the shoulder');
    assert.ok(localPoint(rig.bones.elbowRear).y < localPoint(rig.bones.shoulderRear).y - 0.3, 'Rear elbow must hang naturally');
    assert.ok(Math.abs(rig.bones.kneeLead.rotation.z) > 0.8, 'Guard must visibly bend both knees');
    const guardTip = localPoint(rig.bladeTip);
    assert.ok(guardTip.y > 1.75 && guardTip.y < 2.15, 'High guard point must address head/face height');
    rig.pose('ROCK');
    rig.group.updateMatrixWorld(true);
    assert.ok(Math.abs(rig.bones.kneeRear.rotation.z) < 0.25, 'Rear leg must extend almost straight');
    assert.ok(Math.abs(Math.abs(rig.bones.kneeLead.rotation.z) - Math.PI / 2) < 0.15, 'Front knee must form a deep right-angle lunge');
    const shoulder = localPoint(rig.bones.shoulderLead), wrist = localPoint(rig.bones.wristLead);
    assert.ok(wrist.x - shoulder.x > 0.76, 'Weapon arm must be fully extended');
    assert.ok(Math.abs(wrist.y - shoulder.y) < 0.04, 'Weapon arm must extend point-in-line, not slash');
    assert.ok(localPoint(rig.bones.wristRear).x < localPoint(rig.bones.shoulderRear).x - 0.6, 'Rear arm must counterbalance behind');
    const blade = localPoint(rig.bladeTip).sub(localPoint(rig.bladeBase)).normalize();
    assert.ok(blade.x > 0.96 && Math.abs(blade.y) < 0.06, 'Lunge blade must address the opponent horizontally');
    for (const variant of ['PAPER', 'PAPER_QUARTE', 'PAPER_PRIME']) {
      rig.pose(variant);
      checkFeet(rig);
      const direction = localPoint(rig.bladeTip).sub(localPoint(rig.bladeBase));
      assert.ok(variant === 'PAPER_PRIME' ? direction.y < -0.5 : direction.y > 0.5, 'Paper guard must redirect into the selected line');
    }
    rig.group.traverse(object => assert.notEqual(object.geometry?.type, 'BoxGeometry', 'Rig must contain no chunky boxes'));
  });
}

for (const facing of [true, false]) {
  test(`rear arm stays below head in all poses, mirrored=${!facing}`, () => {
    const rig = new StudioRig(0x0044ff, facing);
    for (const pose of poses) {
      rig.pose(pose);
      rig.group.updateMatrixWorld(true);
      const hand = rig.bones.wristRear.getWorldPosition(new THREE.Vector3());
      const head = rig.bones.head.getWorldPosition(new THREE.Vector3());
      assert.ok(hand.y < head.y - 0.3, `${pose}: rear hand must not lock overhead`);
    }
    rig.pose('SALUTE');
    rig.group.updateMatrixWorld(true);
    const tip = rig.bladeTip.getWorldPosition(new THREE.Vector3());
    const base = rig.bladeBase.getWorldPosition(new THREE.Vector3());
    assert.ok(tip.y - base.y > 0.8, 'Salute must visibly raise the blade');
  });
}


test('slim right-handed rigs have flat athletic soles, independent chest energy and tiny tips', () => {
  for (const facing of [true, false]) {
    const rig = new StudioRig(0x0044ff, facing);
    assert.equal(rig.bones.weapon.parent, rig.bones.wristLead);
    assert.ok(rig.bones.clavicleLead.position.z > 0, 'Both bots use the anatomical right hand');
    assert.ok(rig.group.scale.x > 0, 'Opponent must rotate, never reflect handedness');
    assert.equal(rig.chestOrb.parent, rig.bones.chest);
    assert.equal(rig.chestOrb.scale.y, rig.chestOrb.scale.z);
    assert.ok(rig.bladeTip.children[0].scale.x <= 0.006);
    assert.ok(rig.tipMaterial.emissiveIntensity < 1);
    for (const state of ['win', 'lose', 'neutral']) {
      rig.setOrbState(state);
      assert.equal(rig.orbMaterial.emissiveIntensity === 0, state === 'lose');
      assert.equal(rig.orbMaterial.emissive.getHex() === 0, state === 'lose');
      assert.equal(rig.orbLight.intensity === 0, state === 'lose');
      assert.equal(rig.orbMaterial.toneMapped, state !== 'win');
      assert.equal(rig.tipMaterial.emissiveIntensity, 0.8, 'Chest state must not enlarge or overdrive the tip');
    }
    for (const pose of poses) {
      rig.pose(pose);
      rig.group.updateMatrixWorld(true);
      for (const side of ['Lead', 'Rear']) {
        const shoe = rig.group.getObjectByName(`fencing-shoe-sole-${side}`);
        const bounds = new THREE.Box3().setFromObject(shoe);
        assert.ok(Math.abs(bounds.min.y) < 1e-6, `${pose}: sole must contact ground exactly`);
        assert.ok(bounds.max.y < 0.04, 'Outsole must be thin');
      }
    }
  }
});

test('bare silver helmet has no visor, strip, mask or fixtures', () => {
  for (const color of [0x0044ff, 0x990000]) {
    const rig = new StudioRig(color);
    assert.equal(rig.group.getObjectByName('woven-mask'), undefined);
    assert.deepEqual(rig.bones.head.children.map(object => object.name), ['clean-helmet-shell']);
    assert.equal(rig.group.getObjectByName('colored-eye-strip'), undefined);
  }
});

test('energy core sits inside a genuinely indented continuous sternum socket', () => {
  const rig = new StudioRig();
  assert.ok(rig.chestOrb.position.x < 0.1);
  assert.ok(rig.chestOrb.position.x + rig.chestOrb.scale.x < 0.125);
  assert.equal(rig.chestSocket.parent, rig.bones.chest);
  const positions = rig.chestShell.geometry.attributes.position;
  let center = Infinity, rim = 0;
  for (let i = 0; i < positions.count; i++) {
    const x = positions.getX(i) * 0.76;
    const radial = Math.hypot(positions.getY(i) - 0.055, positions.getZ(i) * 1.2);
    if (x > 0 && radial < 0.015) center = Math.min(center, x);
    if (x > 0 && radial > 0.08 && radial < 0.09) rim = Math.max(rim, x);
  }
  assert.ok(rim - center > 0.045, `Actual cavity depth: ${rim - center}`);
  assert.ok(center < rig.chestOrb.position.x, 'Core emerges from the cavity, not the uncut jacket');
});

test('rear toes turn outward, never inward; outcome feet are parallel and minimal', () => {
  for (const facing of [true, false]) {
    const rig = new StudioRig(0x0044ff, facing);
    for (const pose of poses) {
      rig.pose(pose);
      const inverse = rig.group.quaternion.clone().invert();
      const direction = new THREE.Vector3(1, 0, 0).applyQuaternion(rig.bones.ankleRear.getWorldQuaternion(new THREE.Quaternion())).applyQuaternion(inverse);
      if (['VICTORY', 'DEFEAT', 'ATTENTION'].includes(pose)) assert.ok(direction.x > 0.999);
      else assert.ok(direction.z < -0.999, `${pose}: rear toes must point away from lead leg`);
      const sole = rig.group.getObjectByName('fencing-shoe-sole-Rear');
      sole.geometry.computeBoundingBox();
      assert.ok(sole.geometry.boundingBox.max.x - sole.geometry.boundingBox.min.x < 0.18);
      assert.ok(sole.geometry.boundingBox.max.z - sole.geometry.boundingBox.min.z < 0.085);
    }
  }
});


test('guard gaze stays on duel axis with untwisted torso through breathing and action blends', () => {
  for (const facing of [true, false]) {
    const rig = new StudioRig(0x0044ff, facing);
    const forward = new THREE.Vector3(facing ? 1 : -1, 0, 0);
    for (let time = 0; time < 8; time += 0.1) {
      rig.pose('ON_GUARD', time);
      const gaze = new THREE.Vector3(1, 0, 0).applyQuaternion(rig.bones.head.getWorldQuaternion(new THREE.Quaternion()));
      assert.ok(gaze.distanceTo(forward) < 1e-9, 'Head must look level toward opponent');
      assert.equal(rig.bones.spine.rotation.y, 0);
      assert.equal(rig.bones.chest.rotation.y, 0);
      assert.equal(rig.bones.neck.rotation.y, 0);
      checkFeet(rig);
    }
    for (const pose of poses) {
      const from = rig.capture('ON_GUARD'), to = rig.capture(pose);
      for (let step = 0; step <= 20; step++) {
        rig.applyInterpolation(from, to, step / 20);
        assert.ok(Math.abs(rig.bones.spine.rotation.y) <= 0.121, 'No inherited guard twist');
        checkFeet(rig);
      }
    }
  }
});
