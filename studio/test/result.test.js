import test from 'node:test';
import assert from 'node:assert/strict';
import * as THREE from 'three';
import { StudioRig } from '../src/rig.js';
import { StudioCamera } from '../src/camera.js';
import { RoundResult, resultLabel } from '../src/result.js';

function setup() {
  const scene = new THREE.Scene();
  const rigs = [new StudioRig(0x0044ff, true), new StudioRig(0x990000, false)];
  rigs.forEach((rig, i) => { rig.group.position.x = i ? 1.1 : -1.1; scene.add(rig.group); });
  return { scene, rigs, result: new RoundResult(scene, rigs) };
}

test('outcomes smoothly settle side by side facing camera, no winner lights or beams', () => {
  const { scene, rigs, result } = setup();
  const camera = new THREE.PerspectiveCamera();
  camera.position.set(-0.45, 5.5, 6.4);
  for (const winner of [0, 1, null, 1, 0]) {
    result.clear();
    rigs.forEach(rig => rig.pose('ON_GUARD'));
    const start = rigs.map(rig => rig.group.quaternion.clone());
    result.show(winner, camera);
    rigs.forEach((rig, i) => assert.ok(rig.group.quaternion.angleTo(start[i]) < 1e-6, 'No orientation teleport'));
    for (let frame = 0; frame < 120; frame++) {
      result.update(1 / 60);
      scene.updateMatrixWorld(true);
      rigs.forEach(rig => ['Lead', 'Rear'].forEach(side => {
        const ankle = rig.bones[`ankle${side}`];
        const up = new THREE.Vector3(0, 1, 0).applyQuaternion(ankle.getWorldQuaternion(new THREE.Quaternion()));
        assert.ok(up.y > 0.9999);
        assert.ok(Math.abs(ankle.getWorldPosition(new THREE.Vector3()).y - 0.075) < 0.001);
      }));
    }
    assert.ok(result.settled);
    assert.ok(rigs[0].group.quaternion.angleTo(rigs[1].group.quaternion) < 1e-6);
    assert.ok(Math.abs(rigs[0].group.position.distanceTo(rigs[1].group.position) - 1.6) < 1e-6);
    const direction = camera.position.clone().setY(0).normalize();
    rigs.forEach((rig, i) => {
      assert.ok(new THREE.Vector3(1, 0, 0).applyQuaternion(rig.group.quaternion).dot(direction) > 0.99999);
      assert.equal(rig.currentPose, winner === null ? 'ATTENTION' : winner === i ? 'VICTORY' : 'DEFEAT');
      assert.equal(rig.orbState, winner === null ? 'neutral' : winner === i ? 'win' : 'lose');
      const tip = rig.bladeTip.getWorldPosition(new THREE.Vector3());
      const base = rig.bladeBase.getWorldPosition(new THREE.Vector3());
      assert.ok(winner !== null && i !== winner ? tip.y < base.y - 0.75 : tip.y > base.y + 0.8);
      if (winner !== null && i === winner) assert.ok(base.y > 2, 'Victor raises sword hand high');
      if (winner !== null && i !== winner) {
        assert.ok(rig.bones.head.rotation.z < -0.3, 'Defeated head bows');
        assert.ok(base.y < 1, 'Defeated sword hangs low');
        assert.equal(rig.orbMaterial.emissive.getHex(), 0);
      }
    });
  }
  assert.equal(scene.children.length, 2, 'Only rigs: no result lights, targets or beam geometry');
});

test('reset restores duel transforms and neutral orbs; validates winner', () => {
  const { rigs, result } = setup();
  result.show(1);
  result.update(1);
  result.clear();
  assert.equal(result.active, false);
  assert.equal(result.winner, null);
  rigs.forEach((rig, i) => {
    assert.equal(rig.group.position.x, i ? 1.1 : -1.1);
    assert.ok(rig.group.quaternion.angleTo(new THREE.Quaternion().setFromAxisAngle(new THREE.Vector3(0, 1, 0), i ? Math.PI : 0)) < 1e-6);
    assert.equal(rig.orbState, 'neutral');
  });
  assert.match(resultLabel(0), /COBALT WINS/);
  assert.match(resultLabel(1), /BLOOD RED WINS/);
  assert.match(resultLabel(null), /NO POINT/);
  assert.throws(() => result.show(2), /Invalid round winner/);
});

test('God and broadcast cameras fit complete outcome bodies and swords', () => {
  for (const aspect of [0.5, 1, 1.4, 2.4]) for (const preset of ['god', 'broadcast']) {
    const { rigs, result } = setup();
    const camera = new StudioCamera();
    camera.resize(aspect); camera.select(preset);
    for (const winner of [0, 1, null]) {
      result.clear();
      result.show(winner, camera.camera); result.update(1);
      for (const rig of rigs) {
        rig.group.updateMatrixWorld(true);
        for (const object of [rig.bladeTip, rig.bones.head, rig.bones.ankleLead, rig.bones.ankleRear]) {
          const point = object.getWorldPosition(new THREE.Vector3()).project(camera.camera);
          assert.ok(Math.abs(point.x) < 0.98 && Math.abs(point.y) < 0.98, `${preset} aspect=${aspect}: ${object.name} clipped ${point.toArray()}`);
        }
      }
    }
  }
});

test('outcome facing follows each camera preset and a manually orbited camera', () => {
  const camera = new StudioCamera();
  for (const preset of ['god', 'broadcast', 'dynamic', 'stadium', 'low', 'close']) {
    const { rigs, result } = setup();
    camera.select(preset);
    camera.camera.position.applyAxisAngle(new THREE.Vector3(0, 1, 0), 0.7);
    result.show(0, camera.camera); result.update(1);
    const forward = camera.camera.position.clone().setY(0).normalize();
    rigs.forEach(rig => assert.ok(new THREE.Vector3(1, 0, 0).applyQuaternion(rig.group.quaternion).dot(forward) > 0.99999));
    const separation = rigs[1].group.position.clone().sub(rigs[0].group.position);
    assert.ok(Math.abs(separation.dot(forward)) < 1e-6, 'Both stand in the same viewer-facing plane');
  }
});

test('all outcome poses recover smoothly to exact original on-guard transforms', async () => {
  for (const winner of [0, 1, null]) {
    const { rigs, result, scene } = setup();
    const guard = rigs.map(rig => rig.capture('ON_GUARD'));
    const shown = result.show(winner);
    result.update(0.85);
    await shown;
    const positions = rigs.map(rig => rig.group.position.clone());
    const recovered = result.returnToGuard();
    rigs.forEach((rig, i) => assert.ok(rig.group.position.distanceTo(positions[i]) < 1e-9, 'No recovery teleport'));
    for (let frame = 0; frame < 50; frame++) {
      result.update(1 / 60);
      scene.updateMatrixWorld(true);
      rigs.forEach(rig => ['Lead', 'Rear'].forEach(side => {
        const ankle = rig.bones[`ankle${side}`];
        assert.ok(Math.abs(ankle.getWorldPosition(new THREE.Vector3()).y - 0.075) < 0.001);
        assert.ok(new THREE.Vector3(0, 1, 0).applyQuaternion(ankle.getWorldQuaternion(new THREE.Quaternion())).y > 0.9999);
      }));
    }
    await recovered;
    assert.equal(result.active, false);
    assert.equal(result.phase, 'ready');
    rigs.forEach((rig, i) => {
      assert.equal(rig.currentPose, 'ON_GUARD');
      assert.equal(rig.orbState, 'neutral');
      assert.ok(rig.group.position.distanceTo(result.homes[i].position) < 1e-9);
      assert.ok(rig.group.quaternion.angleTo(result.homes[i].quaternion) < 1e-6);
      Object.entries(rig.bones).forEach(([name, bone]) => {
        assert.ok(bone.position.distanceTo(guard[i][name].pos) < 1e-6, name);
        assert.ok(bone.quaternion.angleTo(guard[i][name].quat) < 1e-6, name);
      });
    });
  }
});
