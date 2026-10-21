import * as THREE from 'three';
import { FIGHTER_COLORS } from './palette.js';

export const CLASH_SPARK_COLOR = 0xffdf9e;
export const sparkColor = winner => winner === null ? CLASH_SPARK_COLOR : FIGHTER_COLORS[winner];

export const SPARK_COUNT = 48;
export const SPARK_LIFETIME = 0.46;

// Reusable world-space buffers; no geometry, texture or material allocations
// during playback. Additive HDR light stays vivid under scene tone mapping.
export class ContactSparks {
  constructor(scene) {
    this.group = new THREE.Group();
    this.group.name = 'blade-contact-effects';
    scene.add(this.group);
    this.positions = new Float32Array(SPARK_COUNT * 3);
    this.velocities = new Float32Array(SPARK_COUNT * 3);
    this.streakPositions = new Float32Array(SPARK_COUNT * 6);
    this.colors = new Float32Array(SPARK_COUNT * 6);
    this.strengths = new Float32Array(SPARK_COUNT);
    this.geometry = new THREE.BufferGeometry();
    this.geometry.setAttribute('position', new THREE.BufferAttribute(this.positions, 3).setUsage(THREE.DynamicDrawUsage));
    this.geometry.setAttribute('strength', new THREE.BufferAttribute(this.strengths, 1).setUsage(THREE.DynamicDrawUsage));
    this.tint = new THREE.Color(CLASH_SPARK_COLOR);
    this.material = new THREE.ShaderMaterial({
      uniforms: { tint: { value: this.tint } },
      transparent: true, blending: THREE.AdditiveBlending, depthWrite: false, toneMapped: false,
      vertexShader: `attribute float strength; varying float glow;
        void main() { glow = strength; vec4 p = modelViewMatrix * vec4(position, 1.0);
          gl_Position = projectionMatrix * p;
          gl_PointSize = clamp(85.0 * projectionMatrix[1][1] * strength / (isOrthographic ? 7.0 : -p.z), 1.0, 5.0); }`,
      fragmentShader: `uniform vec3 tint; varying float glow;
        void main() { float r = length(gl_PointCoord - 0.5) * 2.0;
          float a = pow(max(0.0, 1.0 - r), 1.7) * glow;
          gl_FragColor = vec4(tint * (1.0 + pow(max(0.0, 1.0-r), 3.0) * 0.5), a); }`,
    });
    this.points = new THREE.Points(this.geometry, this.material);
    this.points.frustumCulled = false;
    this.group.add(this.points);
    this.streakGeometry = new THREE.BufferGeometry();
    this.streakGeometry.setAttribute('position', new THREE.BufferAttribute(this.streakPositions, 3).setUsage(THREE.DynamicDrawUsage));
    this.streakGeometry.setAttribute('color', new THREE.BufferAttribute(this.colors, 3).setUsage(THREE.DynamicDrawUsage));
    this.streaks = new THREE.LineSegments(this.streakGeometry, new THREE.LineBasicMaterial({
      vertexColors: true, transparent: true, blending: THREE.AdditiveBlending, depthWrite: false, toneMapped: false,
    }));
    this.streaks.frustumCulled = false;
    this.group.add(this.streaks);
    this.flare = new THREE.Mesh(new THREE.PlaneGeometry(1, 1), new THREE.ShaderMaterial({
      uniforms: { power: { value: 0 }, tint: { value: this.tint } }, transparent: true, blending: THREE.AdditiveBlending,
      depthWrite: false, depthTest: false, toneMapped: false,
      vertexShader: `varying vec2 uvFlare; void main() { uvFlare = uv; gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0); }`,
      fragmentShader: `uniform vec3 tint; uniform float power; varying vec2 uvFlare;
        void main() { vec2 p = (uvFlare - 0.5) * 2.0; float r = length(p);
          float halo = pow(max(0.0, 1.0-r), 3.0);
          float core = exp(-r*r*100.0);
          float rays = (exp(-abs(p.x)*65.0) + exp(-abs(p.y)*65.0)) * pow(max(0.0, 1.0-r), 1.5);
          gl_FragColor = vec4(tint * (halo*0.35 + core*1.4 + rays*0.18), power); }`,
    }));
    this.group.add(this.flare);
    this.light = new THREE.PointLight(CLASH_SPARK_COLOR, 0, 1.4);
    this.group.add(this.light);
    this.life = 0;
    this.group.visible = false;
  }

  burst(point, collisionVector, winner = null) {
    if (winner !== null && winner !== 0 && winner !== 1) throw new Error('Invalid spark winner');
    this.winner = winner;
    this.tint.setHex(sparkColor(winner));
    this.light.color.copy(this.tint);
    const axis = collisionVector.clone().normalize();
    if (axis.lengthSq() < 0.01) axis.set(1, 0, 0);
    const u = new THREE.Vector3().crossVectors(axis, Math.abs(axis.y) < 0.9 ? new THREE.Vector3(0, 1, 0) : new THREE.Vector3(1, 0, 0)).normalize();
    const v = new THREE.Vector3().crossVectors(axis, u);
    for (let i = 0; i < SPARK_COUNT; i++) {
      const k = i * 3, angle = i * 2.399963;
      const speed = 0.7 + (i % 17) * 0.065;
      // Opposing fans along the blade collision vector, plus a broad cone.
      const along = (i % 4 === 0 ? -1 : 1) * (0.65 + (i % 7) * 0.08);
      const spread = 0.2 + (i % 11) * 0.035;
      point.toArray(this.positions, k);
      for (let a = 0; a < 3; a++) this.velocities[k + a] = speed * (axis.getComponent(a) * along + spread * (Math.cos(angle) * u.getComponent(a) + Math.sin(angle) * v.getComponent(a))) + (a === 1 ? 0.35 : 0);
    }
    this.life = SPARK_LIFETIME;
    this.group.visible = true;
    this.flare.position.copy(point);
    this.light.position.copy(point);
    this.update(0);
    this.fresh = true;
  }

  update(dt, camera) {
    if (this.life <= 0) return;
    // The burst may be scheduled just before a slow render frame; never age
    // away its crisp core before it has appeared on screen once.
    if (this.fresh) { dt = 0; this.fresh = false; }
    this.life = Math.max(0, this.life - dt);
    const age = SPARK_LIFETIME - this.life;
    const flash = Math.exp(-age * 24);
    this.flare.material.uniforms.power.value = flash * 0.45;
    this.flare.scale.setScalar(0.28 + age * 0.45);
    if (camera) this.flare.quaternion.copy(camera.quaternion);
    this.light.intensity = flash * 3;
    for (let i = 0; i < SPARK_COUNT; i++) {
      const k = i * 3, line = i * 6;
      const fade = Math.max(0, 1 - age / (0.26 + (i % 13) * 0.016));
      this.strengths[i] = Math.pow(fade, 0.7) * 0.7;
      this.velocities[k + 1] -= 5.5 * dt;
      for (let a = 0; a < 3; a++) {
        this.positions[k + a] += this.velocities[k + a] * dt;
        this.streakPositions[line + a] = this.positions[k + a];
        this.streakPositions[line + 3 + a] = this.positions[k + a] - this.velocities[k + a] * 0.022;
        this.colors[line + a] = (a === 0 ? this.tint.r : a === 1 ? this.tint.g : this.tint.b) * fade * 1.2;
        this.colors[line + 3 + a] = (a === 0 ? this.tint.r : a === 1 ? this.tint.g : this.tint.b) * fade * 0.12;
      }
    }
    this.geometry.attributes.position.needsUpdate = true;
    this.geometry.attributes.strength.needsUpdate = true;
    this.streakGeometry.attributes.position.needsUpdate = true;
    this.streakGeometry.attributes.color.needsUpdate = true;
    this.group.visible = this.life > 0;
  }
}
