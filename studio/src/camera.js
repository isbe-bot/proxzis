import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

export const CAMERA_PRESETS = Object.freeze({
  // Center the full choreography envelope (floor through victory's sword),
  // rather than centering the torso and reserving empty space below the feet.
  god: { position: [-0.35, 5.1, 5], target: [0, 1.65, 0], width: 5 },
  broadcast: { position: [-1.2, 1.8, 10], target: [0, 1.8, 0], halfHeight: 1.9, halfWidth: 1.95 },
  dynamic: { position: [-1.65, 2.65, 7.7], target: [0, 1.05, 0], width: 6 },
  stadium: { position: [-2.2, 7.2, 10.5], target: [0, 0.95, 0], width: 8.5 },
  low: { position: [-1.2, 0.48, 7.5], target: [0, 1.25, 0], width: 6 },
  close: { position: [-0.65, 1.95, 4.1], target: [0, 1.42, 0], width: 2.8 },
});

export class StudioCamera {
  constructor(domElement = null) {
    this.domElement = domElement;
    this.perspective = new THREE.PerspectiveCamera(36, 1, 0.1, 70);
    this.orthographic = new THREE.OrthographicCamera(-3, 3, 1.55, -1.55, 0.1, 70);
    this.aspect = 1;
    this.aim = new THREE.Vector3();
    this.select('god');
  }

  select(preset) {
    if (!Object.hasOwn(CAMERA_PRESETS, preset)) throw new Error(`Unknown camera preset: ${preset}`);
    this.preset = preset;
    this.camera = preset === 'broadcast' ? this.orthographic : this.perspective;
    this.resize(this.aspect);
    this.updatePreset(0);
    if (this.domElement) {
      // Discard pending drag/zoom inertia completely when selecting a preset.
      this.controls?.dispose();
      this.camera.zoom = 1;
      this.camera.updateProjectionMatrix();
      this.attachControls(this.aim);
    }
  }

  attachControls(target) {
    this.controls = new OrbitControls(this.camera, this.domElement);
    this.controls.target.copy(target);
    this.controls.enablePan = false;
    this.controls.enableDamping = true;
    this.controls.dampingFactor = 0.1;
    this.controls.minDistance = 1.5;
    this.controls.maxDistance = 35;
    this.controls.minZoom = 0.35;
    this.controls.maxZoom = 4;
    this.controls.update();
  }

  setLocked(value) {
    if (!value && this.locked && this.controls) {
      // Paused drag inertia must not resume after export. Rebind controls to
      // the exact recorded view, without selecting a preset or resetting zoom.
      const target = this.controls.target.clone();
      this.controls.dispose();
      this.attachControls(target);
    }
    this.locked = value;
    if (this.controls) this.controls.enabled = !value;
  }

  resize(aspect) {
    this.aspect = Number.isFinite(aspect) && aspect > 0 ? aspect : 1;
    this.perspective.aspect = this.aspect;
    this.perspective.updateProjectionMatrix();
    // The full mesh envelope includes victory's raised blade, not just the
    // shorter salute. Fit width on portrait screens and height on wide ones.
    const halfHeight = Math.max(CAMERA_PRESETS.broadcast.halfHeight, CAMERA_PRESETS.broadcast.halfWidth / this.aspect);
    Object.assign(this.orthographic, { left: -halfHeight * this.aspect, right: halfHeight * this.aspect, top: halfHeight, bottom: -halfHeight });
    this.orthographic.updateProjectionMatrix();
  }

  update(time, shake = 0) {
    if (this.locked) return;
    // Never overwrite user orbit/zoom with preset animation or impact shake.
    if (this.controls) {
      this.controls.update();
      this.camera.updateMatrixWorld();
      return;
    }
    this.updatePreset(time, shake);
  }

  updatePreset(time, shake = 0) {
    const preset = CAMERA_PRESETS[this.preset];
    this.camera.position.set(...preset.position);
    if (this.preset !== 'broadcast') {
      this.camera.position.z = Math.max(preset.position[2], preset.width / this.aspect);
      if (this.preset === 'stadium' || this.preset === 'god') {
        this.camera.position.y = preset.target[1] + (preset.position[1] - preset.target[1]) * this.camera.position.z / preset.position[2];
      }
      if (this.preset === 'dynamic') {
        this.camera.position.x += Math.sin(time * 0.24) * 0.12;
        this.camera.position.y += Math.sin(time * 0.4) * 0.035;
      }
      this.camera.position.x += Math.sin(time * 130) * shake;
      this.camera.position.y += Math.cos(time * 170) * shake;
    }
    this.aim.set(...preset.target);
    this.camera.lookAt(this.aim);
    this.camera.updateMatrixWorld();
  }
}
