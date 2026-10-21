import test from 'node:test';
import assert from 'node:assert/strict';
import * as THREE from 'three';
import { StudioCamera, CAMERA_PRESETS } from '../src/camera.js';

test('all six camera presets switch without invalid transforms at any aspect', () => {
  const rig = new StudioCamera();
  assert.equal(Object.keys(CAMERA_PRESETS).length, 6);
  for (const aspect of [0.5, 1, 1.2, 2.4]) {
    rig.resize(aspect);
    for (const preset of Object.keys(CAMERA_PRESETS)) {
      rig.select(preset);
      rig.update(1.5, 0.05);
      assert.equal(rig.camera.isOrthographicCamera === true, preset === 'broadcast');
      assert.ok([...rig.camera.position.toArray(), ...rig.camera.projectionMatrix.elements].every(Number.isFinite));
      const target = new THREE.Vector3(...CAMERA_PRESETS[preset].target).project(rig.camera);
      assert.ok(Math.abs(target.x) < 1e-6 && Math.abs(target.y) < 1e-6);
    }
    // Whole-body/blade fit is covered by the real-mesh choreography sweep in
    // camera-framing.test.js, not an oversized box of impossible corner poses.
  }
  assert.throws(() => rig.select('invalid'), /Unknown camera preset/);
});

test('broadcast stays front-left and stable; close clash is materially tighter', () => {
  const rig = new StudioCamera();
  rig.resize(1.4);
  rig.select('broadcast');
  const position = rig.camera.position.clone();
  rig.update(9, 0.5);
  assert.ok(position.equals(rig.camera.position));
  assert.ok(position.x < 0);
  assert.equal(position.y, CAMERA_PRESETS.broadcast.target[1]);
  rig.select('dynamic');
  const wide = rig.camera.position.distanceTo(rig.aim);
  rig.select('close');
  assert.ok(rig.camera.position.distanceTo(rig.aim) < wide * 0.65);
  rig.select('low');
  assert.ok(rig.camera.position.y < rig.aim.y);
});


test('stadium looks down from the stands and fits fighters and salutes', () => {
  const rig = new StudioCamera();
  for (const aspect of [0.5, 1, 1.4, 2.4]) {
    rig.resize(aspect);
    rig.select('stadium');
    assert.ok(rig.camera.isPerspectiveCamera);
    assert.ok(rig.camera.position.y - rig.aim.y > 6);
    for (const x of [-2.55, 2.55]) for (const y of [0, 2.9]) {
      const point = new THREE.Vector3(x, y, 0).project(rig.camera);
      assert.ok(Math.abs(point.x) < 1 && Math.abs(point.y) < 1);
    }
  }
});

test('all presets remain on the front-left side through dynamic motion and impact shake', () => {
  const rig = new StudioCamera();
  for (const aspect of [0.5, 1, 1.4, 2.4]) {
    rig.resize(aspect);
    for (const preset of Object.keys(CAMERA_PRESETS)) {
      rig.select(preset);
      for (let t = 0; t < 30; t += 0.1) {
        rig.update(t, 0.08);
        assert.ok(rig.camera.position.x < 0, preset);
        assert.ok(rig.camera.position.z > 0, preset);
      }
    }
  }
});

test('interactive orbit persists across frames/resize and presets reset both projections', () => {
  const root = new EventTarget();
  const canvas = Object.assign(new EventTarget(), { style: {}, getRootNode: () => root });
  const rig = new StudioCamera(canvas);
  rig.resize(1.4);
  for (const preset of Object.keys(CAMERA_PRESETS)) {
    rig.select(preset);
    const original = rig.camera.position.clone();
    const target = rig.controls.target.clone();
    rig.camera.position.sub(target).applyAxisAngle(new THREE.Vector3(0, 1, 0), 0.7).add(target);
    rig.camera.zoom = 1.8;
    rig.controls.update();
    const orbited = rig.camera.position.clone();
    rig.update(5, 0.5);
    rig.resize(1.6);
    rig.update(8, 0.5);
    assert.ok(rig.camera.position.distanceTo(orbited) < 1e-8, 'orbit must not snap back each frame');
    assert.equal(rig.camera.zoom, 1.8);
    rig.resize(1.4);
    const oldControls = rig.controls;
    rig.select(preset);
    assert.notEqual(rig.controls, oldControls);
    assert.equal(rig.controls.object, rig.camera);
    assert.ok(rig.camera.position.distanceTo(original) < 1e-8);
    assert.ok(rig.controls.target.distanceTo(new THREE.Vector3(...CAMERA_PRESETS[preset].target)) < 1e-8);
    assert.equal(rig.camera.zoom, 1);
    assert.equal(rig.controls.enablePan, false);
    assert.equal(canvas.style.touchAction, 'none');
  }
  rig.controls.dispose();
});


test('viewport entry has no remaining ground badge module or lifecycle references', async () => {
  const { readFile } = await import('node:fs/promises');
  const main = await readFile(new URL('../src/main.js', import.meta.url), 'utf8');
  assert.doesNotMatch(main, /throwIndicators|ThrowIndicators|throws\.js/);
});


test('God View remains the default elevated front-left seat at every aspect', () => {
  const rig = new StudioCamera();
  assert.equal(rig.preset, 'god');
  for (const aspect of [0.5, 1, 1.4, 2.4]) {
    rig.resize(aspect);
    rig.select('god');
    const elevation = Math.atan2(rig.camera.position.y - rig.aim.y, rig.camera.position.z);
    assert.ok(elevation > Math.PI / 6 && elevation < Math.PI / 4);
    assert.ok(rig.camera.position.x < 0 && rig.camera.position.z > 0);
    // Complete mesh extrema, including the sword's tip radius during turns,
    // are asserted across all 27 timelines in camera-framing.test.js.
  }
});

test('recording lock freezes exact orbit/zoom without advancing controls or presets', () => {
  for (const preset of Object.keys(CAMERA_PRESETS)) {
    const rig = new StudioCamera();
    rig.select(preset);
    rig.camera.position.set(-3, 4, 8);
    rig.camera.zoom = 1.7;
    rig.camera.updateProjectionMatrix();
    rig.locked = true;
    rig.controls = { update() { throw new Error('Locked controls must not advance damping'); } };
    const position = rig.camera.position.clone();
    const quaternion = rig.camera.quaternion.clone();
    for (let frame = 0; frame < 120; frame++) rig.update(frame, 1);
    assert.ok(rig.camera.position.equals(position));
    assert.ok(rig.camera.quaternion.equals(quaternion));
    assert.equal(rig.camera.zoom, 1.7);
  }
});

test('unlock discards pending drag inertia while retaining exact camera view', () => {
  const root = new EventTarget();
  const canvas = Object.assign(new EventTarget(), { style: {}, getRootNode: () => root });
  const rig = new StudioCamera(canvas);
  for (const preset of Object.keys(CAMERA_PRESETS)) {
    rig.select(preset);
    rig.camera.position.set(-3, 4, 8);
    rig.camera.zoom = 1.7;
    rig.camera.updateProjectionMatrix();
    rig.controls.target.set(0.1, 1.2, 0.2);
    rig.controls.update();
    const oldControls = rig.controls;
    const position = rig.camera.position.clone(), quaternion = rig.camera.quaternion.clone();
    const target = rig.controls.target.clone();
    rig.setLocked(true);
    assert.equal(rig.controls.enabled, false);
    rig.update(100, 1);
    rig.setLocked(false);
    assert.notEqual(rig.controls, oldControls, 'Fresh controls discard old private drag state');
    assert.equal(rig.controls.enabled, true);
    for (let frame = 0; frame < 120; frame++) rig.update(frame, 1);
    assert.ok(rig.camera.position.distanceTo(position) < 1e-8);
    assert.ok(rig.camera.quaternion.angleTo(quaternion) < 1e-6);
    assert.ok(rig.controls.target.distanceTo(target) < 1e-8);
    assert.equal(rig.camera.zoom, 1.7);
  }
  rig.controls.dispose();
});
