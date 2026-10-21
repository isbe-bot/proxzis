import * as THREE from 'three';
import { FIGHTER_NAMES } from './palette.js';
import { TIMING } from './capture.js';

export const resultLabel = winner => winner === null ? 'DEADLOCK · NO POINT' : `${FIGHTER_NAMES[winner]} WINS · +1`;

// Outcomes are physical silhouettes, not lighting effects. No result lights or
// beam meshes exist in the scene, including in canvas/WebM recordings.
export class RoundResult {
  constructor(scene, rigs) {
    this.rigs = rigs;
    this.homes = rigs.map(rig => ({ position: rig.group.position.clone(), quaternion: rig.group.quaternion.clone() }));
    this.clear();
  }

  clear() {
    this.active = false;
    this.phase = 'ready';
    this.winner = null;
    this.elapsed = 0;
    this.settled = false;
    this.rigs.forEach((rig, i) => {
      rig.group.position.copy(this.homes[i].position);
      rig.group.quaternion.copy(this.homes[i].quaternion);
      rig.setOrbState('neutral');
    });
  }

  show(winner, camera) {
    if (winner !== null && winner !== 0 && winner !== 1) throw new Error('Invalid round winner');
    this.active = true;
    this.phase = 'outcome';
    this.duration = 0.85;
    this.winner = winner;
    this.elapsed = 0;
    this.settled = false;
    // A shared camera-facing heading keeps both shoulders in the same plane.
    const direction = camera?.position.clone() || new THREE.Vector3(0, 1, 10);
    direction.y = 0;
    if (direction.lengthSq() < 1e-6) direction.set(0, 0, 1);
    direction.normalize();
    const facing = new THREE.Quaternion().setFromAxisAngle(new THREE.Vector3(0, 1, 0), Math.atan2(-direction.z, direction.x));
    const right = new THREE.Vector3(direction.z, 0, -direction.x);
    this.transitions = this.rigs.map((rig, i) => {
      const from = rig.snapshot();
      rig.group.updateMatrixWorld(true);
      const inverse = rig.group.matrixWorld.clone().invert();
      from.feet = ['Lead', 'Rear'].map(side => rig.bones[`ankle${side}`].getWorldPosition(new THREE.Vector3()).applyMatrix4(inverse).x);
      const position = rig.group.position.clone(), quaternion = rig.group.quaternion.clone();
      const to = rig.capture(winner === null ? 'ATTENTION' : winner === i ? 'VICTORY' : 'DEFEAT');
      rig.applyInterpolation(from, from, 0);
      rig.setOrbState(winner === null ? 'neutral' : winner === i ? 'win' : 'lose');
      return { from, to, position, quaternion, target: right.clone().multiplyScalar(i ? 0.8 : -0.8), facing };
    });
    return new Promise(resolve => { this.onSettled = resolve; });
  }

  returnToGuard() {
    this.phase = 'recovery';
    this.elapsed = 0;
    this.duration = TIMING.recoverySeconds;
    this.settled = false;
    this.transitions = this.transitions.map((transition, i) => {
      const rig = this.rigs[i];
      // Preserve planted feet and exclude render-time blade recoil. Capture
      // mutates the rig, so restore the outcome before starting interpolation.
      const from = transition.to;
      const quaternion = rig.group.quaternion.clone();
      // Weapon aiming resolves in world space; author the destination under
      // its original duel heading so the final blade roll matches guard too.
      rig.group.quaternion.copy(this.homes[i].quaternion);
      const to = rig.capture('ON_GUARD');
      rig.group.quaternion.copy(quaternion);
      rig.applyInterpolation(from, from, 0);
      rig.setOrbState('neutral');
      return { from, to, position: rig.group.position.clone(), quaternion: rig.group.quaternion.clone(),
        target: this.homes[i].position, facing: this.homes[i].quaternion };
    });
    return new Promise(resolve => { this.onSettled = resolve; });
  }

  update(dt = 0) {
    if (!this.active || this.settled) return;
    this.elapsed += dt;
    const t = Math.min(this.elapsed / this.duration, 1);
    const alpha = t * t * (3 - 2 * t);
    this.rigs.forEach((rig, i) => {
      const transition = this.transitions[i];
      rig.applyInterpolation(transition.from, transition.to, alpha);
      rig.group.position.lerpVectors(transition.position, transition.target, alpha);
      rig.group.quaternion.slerpQuaternions(transition.quaternion, transition.facing, alpha);
    });
    this.settled = t === 1;
    if (this.settled) {
      if (this.phase === 'recovery') { this.active = false; this.phase = 'ready'; }
      this.onSettled?.();
      this.onSettled = null;
    }
  }
}
