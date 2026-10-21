import test from 'node:test';
import assert from 'node:assert/strict';
import * as THREE from 'three';
import { StudioRig } from '../src/rig.js';
import { buildEngagement, bladeContact, bladePoints, segmentContact } from '../src/combat.js';
const actions = ['STRIKE', 'PARRY', 'FEINT'];
const setup = () => [true, false].map((facing, i) => {
  const rig = new StudioRig(i ? 0x990000 : 0x0044ff, facing);
  rig.group.position.x = i ? 1.1 : -1.1;
  return rig;
});
for (const parry of ['PARRY', 'PARRY_QUARTE', 'PARRY_PRIME']) {
  for (const a of actions) for (const b of actions) {
    test(`${a}/${b} ${parry}: physical contact, bounded reach, grounded feet`, () => {
      const rigs = setup();
      const engagement = buildEngagement(rigs, [a, b], parry);
      rigs.forEach((rig, i) => rig.applyInterpolation(engagement.contact[i], engagement.contact[i], 1));
      if (engagement.kind === 'blade') {
        const hit = bladeContact(...rigs);
        assert.ok(hit.distance < 1e-5, `Blades miss by ${hit.distance}`);
        assert.ok(hit.point.distanceTo(engagement.point) < 1e-5);
        if (a !== b) {
          assert.ok(hit.fractions[engagement.winner] < 0.34, 'Parry uses forte');
          assert.ok(hit.fractions[1-engagement.winner] > 0.65, 'Catch attacker foible');
        }
      } else if (engagement.kind === 'body') {
        const tip = rigs[engagement.winner].bladeTip.getWorldPosition(new THREE.Vector3());
        assert.ok(tip.distanceTo(engagement.point) < 1e-5, 'Point must touch surface, not torso center');
      } else assert.equal(engagement.kind, 'read');
      rigs.forEach(rig => {
        rig.group.updateMatrixWorld(true);
        const shoulder = rig.bones.shoulderLead.getWorldPosition(new THREE.Vector3());
        const elbow = rig.bones.elbowLead.getWorldPosition(new THREE.Vector3());
        const wrist = rig.bones.wristLead.getWorldPosition(new THREE.Vector3());
        assert.ok(Math.abs(shoulder.distanceTo(elbow) - 0.39) < 1e-6);
        assert.ok(Math.abs(elbow.distanceTo(wrist) - 0.38) < 1e-6);
        ['Lead','Rear'].forEach(side => assert.ok(Math.abs(rig.bones[`ankle${side}`].getWorldPosition(new THREE.Vector3()).y - 0.075) < 1e-5));
        const points = bladePoints(rig);
        assert.equal(points.length, 9);
        const forearm = wrist.clone().sub(elbow).normalize();
        const blade = points[8].clone().sub(points[0]).normalize();
        assert.ok(THREE.MathUtils.radToDeg(forearm.angleTo(blade)) < 75, 'No hyperflexed weapon wrists');
      });
    });
  }
}
test('local guard anatomy is identical on both right-handed rigs', () => {
  const rigs = setup();
  rigs.forEach(rig => rig.pose('ON_GUARD'));
  assert.ok(rigs[0].bones.spine.quaternion.angleTo(rigs[1].bones.spine.quaternion) < 1e-8);
});

test('finite segment contact handles crossing, parallel, endpoint and degenerate geometry', () => {
  const v = (...n) => new THREE.Vector3(...n);
  const crossed = segmentContact(v(-1,0,0),v(1,0,0),v(0,-1,0),v(0,1,0));
  assert.equal(crossed.distance, 0);
  assert.deepEqual(crossed.fractions, [0.5,0.5]);
  assert.equal(segmentContact(v(0,0,0),v(1,0,0),v(0,1,0),v(1,1,0)).distance,1);
  assert.equal(segmentContact(v(0,0,0),v(0,0,0),v(0,1,0),v(0,1,0)).distance,1);
  assert.equal(segmentContact(v(0,0,0),v(1,0,0),v(1,0,0),v(1,1,0)).distance,0);
  assert.equal(segmentContact(v(0,0,0),v(0,0,0),v(1,-1,0),v(1,1,0)).distance,1);
});
test('combat rejects invalid actions, parry variants and unreachable wrists', () => {
  const rigs = setup();
  assert.throws(() => buildEngagement(rigs,['STRIKE','BOGUS']), /Invalid matchup/);
  assert.throws(() => buildEngagement(rigs,['STRIKE','PARRY'],'BOGUS'), /Invalid parry/);
  assert.throws(() => rigs[0].setBlade(new THREE.Vector3(100,100,100),new THREE.Vector3(1,0,0)), /Unreachable/);
});
