import test from 'node:test';
import assert from 'node:assert/strict';
import * as THREE from 'three';
import { ContactSparks, SPARK_COUNT, SPARK_LIFETIME, CLASH_SPARK_COLOR } from '../src/sparks.js';
import { FIGHTER_COLORS } from '../src/palette.js';

test('contact bursts are directional, restrained, deterministic and reuse buffers', () => {
  const effect = new ContactSparks(new THREE.Scene());
  const point = new THREE.Vector3(0.2, 1.5, 0.1);
  effect.burst(point, new THREE.Vector3(1, 0, 0));
  assert.equal(SPARK_COUNT, 48);
  assert.equal(effect.light.intensity, 3);
  assert.equal(effect.flare.scale.x, 0.28);
  assert.equal(effect.flare.material.uniforms.power.value, 0.45);
  const velocities = [...effect.velocities];
  const positionBuffer = effect.positions;
  let forward = 0, backward = 0;
  for (let i = 0; i < SPARK_COUNT; i++) {
    effect.velocities[i * 3] > 0 ? forward++ : backward++;
  }
  assert.equal(forward, 36);
  assert.equal(backward, 12);
  assert.equal(effect.flare.material.depthTest, false, 'Core flare stays visible at torso contacts');
  effect.update(0.2, new THREE.PerspectiveCamera());
  assert.equal(effect.flare.material.uniforms.power.value, 0.45, 'First rendered frame preserves peak flash');
  effect.update(0.2);
  assert.ok(effect.group.visible);
  assert.ok(effect.strengths.some(value => value > 0.1));
  assert.ok(effect.streakPositions.every(Number.isFinite));
  effect.update(SPARK_LIFETIME);
  assert.equal(effect.group.visible, false);
  effect.burst(point, new THREE.Vector3(1, 0, 0));
  assert.equal(effect.positions, positionBuffer);
  assert.deepEqual([...effect.velocities], velocities);
  effect.burst(point, new THREE.Vector3());
  assert.ok(effect.velocities.every(Number.isFinite));
});

 test('winner tint reaches particles, streaks, flare and contact light; ties reset neutral', () => {
  const effect = new ContactSparks(new THREE.Scene());
  for (const winner of [0, 1, null, 0, null]) {
    effect.burst(new THREE.Vector3(), new THREE.Vector3(1, 0, 0), winner);
    const expected = winner === null ? CLASH_SPARK_COLOR : FIGHTER_COLORS[winner];
    assert.equal(effect.tint.getHex(), expected);
    assert.equal(effect.light.color.getHex(), expected);
    assert.equal(effect.material.uniforms.tint.value, effect.tint);
    assert.equal(effect.flare.material.uniforms.tint.value, effect.tint);
    const tint = effect.tint.toArray();
    tint.forEach((channel, i) => assert.ok(Math.abs(effect.colors[i] - channel * 1.2) < 1e-6));
  }
  assert.throws(() => effect.burst(new THREE.Vector3(), new THREE.Vector3(), 2), /Invalid spark winner/);
});
