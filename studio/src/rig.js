import * as THREE from 'three';
import { FIGHTER_COLORS } from './palette.js';

// +X is the fencing line, +Y up, +Z the weapon side. Mirror by rotation,
// never negative scale. The fitted jacket has a narrow sagittal profile.
const sphere = new THREE.SphereGeometry(1, 16, 10);
const down = new THREE.Vector3(0, -1, 0);
const up = new THREE.Vector3(0, 1, 0);
const axisZ = new THREE.Vector3(0, 0, 1);
const footTurn = new THREE.Quaternion();
const v = new THREE.Vector3();
const q = new THREE.Quaternion();
const q2 = new THREE.Quaternion();
const poses = ['ON_GUARD', 'SALUTE', 'WINDUP', 'STRIKE', 'FOLLOW_THROUGH', 'PARRY', 'PARRY_QUARTE', 'PARRY_PRIME', 'FEINT', 'FEINT_EXTEND', 'HIT_REACT', 'DEFLECT_REACT', 'VICTORY', 'DEFEAT', 'ATTENTION'];

export class StudioRig {
  constructor(color = FIGHTER_COLORS[0], isFacingRight = true) {
    this.color = color;
    this.isFacingRight = isFacingRight;
    this.group = new THREE.Group();
    this.group.rotation.y = isFacingRight ? 0 : Math.PI;
    this.bones = {};
    this.armor = new THREE.MeshStandardMaterial({ color: 0xc3ccd3, metalness: 0.28, roughness: 0.52 });
    this.edge = new THREE.MeshStandardMaterial({ color: 0x9eafbc, metalness: 0.86, roughness: 0.24 });
    this.dark = new THREE.MeshStandardMaterial({ color: 0x172430, metalness: 0.3, roughness: 0.62 });
    this.accent = new THREE.MeshStandardMaterial({ color, metalness: 0.5, roughness: 0.36 });
    this.glow = new THREE.MeshStandardMaterial({ color, emissive: color, emissiveIntensity: 1.8, toneMapped: false });
    this.bladeMat = new THREE.MeshStandardMaterial({ color: 0xe8efff, emissive: color, emissiveIntensity: 0.3, metalness: 0.9, roughness: 0.18 });
    // Chest energy is independent of the tiny blade-tip glow.
    this.orbMaterial = new THREE.MeshStandardMaterial({ color, emissive: color, emissiveIntensity: 0.6, roughness: 0.24, toneMapped: true });
    this.tipMaterial = this.glow.clone();
    this.tipMaterial.emissiveIntensity = 0.8;
    this.buildSkeleton();
    this.setOrbState('neutral');
    this.rest = this.snapshot();
    this.pose('ON_GUARD');
  }

  joint(name, parent, x = 0, y = 0, z = 0) {
    const pivot = new THREE.Group();
    pivot.name = name;
    pivot.position.set(x, y, z);
    parent.add(pivot);
    this.bones[name] = pivot;
    return pivot;
  }

  mesh(parent, geo, material, position, scale = [1, 1, 1]) {
    const mesh = new THREE.Mesh(geo, material);
    mesh.position.set(...position);
    mesh.scale.set(...scale);
    mesh.castShadow = material !== this.glow;
    mesh.receiveShadow = true;
    parent.add(mesh);
    return mesh;
  }

  oval(parent, material, position, scale) {
    return this.mesh(parent, sphere, material, position, scale);
  }

  // Smooth lathed garment profiles: no boxes, armor plates or broad actuators.
  contour(parent, profile, depth, width, y, material = this.armor) {
    const geo = new THREE.LatheGeometry([...profile].sort((a, b) => a[1] - b[1]).map(([radius, height]) => new THREE.Vector2(radius, height)), 16);
    return this.mesh(parent, geo, material, [0, y, 0], [depth, 1, width]);
  }

  limb(parent, length, upper, lower, material = this.armor) {
    return this.contour(parent, [[0, 0], [upper * 0.7, -0.015], [upper, -0.08], [upper * 0.96, -length * 0.32], [lower, -length * 0.82], [lower * 0.7, -length + 0.012], [0, -length]], 1, 0.88, 0, material);
  }

  ball(parent, radius, x = 0, y = 0, z = 0) {
    return this.oval(parent, this.dark, [x, y, z], [radius, radius, radius]);
  }

  buildSkeleton() {
    const p = this.joint('pelvis', this.group, 0, 1.02, 0);
    this.oval(p, this.dark, [0, -0.035, 0], [0.115, 0.13, 0.18]);
    const spine = this.joint('spine', p, 0, 0.12, 0);
    this.contour(spine, [[0.11, -0.12], [0.125, -0.06], [0.12, 0.05], [0.135, 0.22], [0.14, 0.32]], 0.85, 1.15, 0);
    const chest = this.joint('chest', spine, 0, 0.36, 0);
    // Dense continuous jacket surface, dented inward around the sternum.
    // A surface-mounted ring alone would leave the orb floating above armor.
    const chestProfile = [[0.125, -0.17], [0.155, -0.07], [0.175, 0.09], [0.17, 0.18], [0.10, 0.24], [0.065, 0.25]];
    const denseProfile = [];
    for (let i = 0; i < chestProfile.length - 1; i++) {
      const [r0, y0] = chestProfile[i], [r1, y1] = chestProfile[i + 1];
      const steps = Math.ceil((y1 - y0) / 0.008);
      for (let j = 0; j < steps; j++) denseProfile.push(new THREE.Vector2(THREE.MathUtils.lerp(r0, r1, j / steps), THREE.MathUtils.lerp(y0, y1, j / steps)));
    }
    denseProfile.push(new THREE.Vector2(...chestProfile.at(-1)));
    const chestGeometry = new THREE.LatheGeometry(denseProfile, 64);
    const vertices = chestGeometry.attributes.position;
    for (let i = 0; i < vertices.count; i++) {
      const x = vertices.getX(i), y = vertices.getY(i), z = vertices.getZ(i);
      const radius = Math.hypot(y - 0.055, z * 1.2);
      if (x > 0 && radius < 0.086) {
        const t = radius / 0.086;
        vertices.setX(i, x - 0.065 / 0.76 * (1 - t * t * (3 - 2 * t)));
      }
    }
    chestGeometry.computeVertexNormals();
    this.chestShell = this.mesh(chest, chestGeometry, this.armor, [0, 0, 0], [0.76, 1, 1.2]);
    this.chestShell.name = 'recessed-sternum-shell';
    this.chestSocket = this.mesh(chest, new THREE.TorusGeometry(0.072, 0.006, 10, 48), this.edge, [0.119, 0.055, 0]);
    this.chestSocket.rotation.y = Math.PI / 2;
    this.chestSocket.name = 'chest-core-socket';
    // Frontmost point is now 0.124, versus 0.184 on the old external orb.
    this.chestOrb = this.oval(chest, this.orbMaterial, [0.092, 0.055, 0], [0.032, 0.064, 0.064]);
    this.chestOrb.name = 'chest-energy-orb';
    this.orbLight = new THREE.PointLight(this.color, 0, 0.4, 2);
    this.orbLight.position.set(0.132, 0.055, 0);
    chest.add(this.orbLight);
    this.oval(spine, this.dark, [0, -0.035, 0], [0.126, 0.023, 0.177]);
    const neck = this.joint('neck', chest, 0, 0.25, 0);
    this.oval(neck, this.dark, [0, 0.025, 0], [0.055, 0.075, 0.06]);
    const head = this.joint('head', neck, 0, 0.16, 0);
    this.helmet = this.oval(head, this.armor, [-0.015, 0, 0], [0.139, 0.18, 0.143]);
    this.helmet.name = 'clean-helmet-shell';
    // Deliberately bare: no visor, eye strip, mask or face fixtures.
    for (const [suffix, side] of [['Lead', 1], ['Rear', -1]]) {
      const clav = this.joint(`clavicle${suffix}`, chest, 0, 0.15, side * 0.17);
      const shoulder = this.joint(`shoulder${suffix}`, clav, 0, 0, side * 0.04);
      this.ball(shoulder, 0.058);
      this.limb(shoulder, 0.39, 0.063, 0.043);
      const elbow = this.joint(`elbow${suffix}`, shoulder, 0, -0.39, 0);
      this.ball(elbow, 0.047);
      this.limb(elbow, 0.38, 0.052, 0.033);
      const wrist = this.joint(`wrist${suffix}`, elbow, 0, -0.38, 0);
      this.ball(wrist, 0.036);
      this.oval(wrist, this.dark, [0, -0.055, 0], [0.045, 0.075, 0.04]);
      if (suffix === 'Rear') {
        for (let i = 0; i < 3; i++) this.oval(wrist, this.armor, [0.012 + i * 0.013, -0.125, 0], [0.009, 0.045 - i * 0.006, 0.011]);
      }
      const hip = this.joint(`hip${suffix}`, p, 0, -0.09, side * 0.13);
      this.ball(hip, 0.075);
      this.limb(hip, 0.55, 0.087, 0.053);
      const knee = this.joint(`knee${suffix}`, hip, 0, -0.55, 0);
      this.ball(knee, 0.054);
      this.limb(knee, 0.53, 0.062, 0.033);
      const ankle = this.joint(`ankle${suffix}`, knee, 0, -0.53, 0);
      this.ball(ankle, 0.035);
      // Low-profile athletic upper over a genuinely flat rubber outsole.
      const upper = this.oval(ankle, this.armor, [0.025, -0.033, 0], [0.083, 0.028, 0.037]);
      upper.name = `minimal-shoe-upper-${suffix}`;
      const sole = new THREE.Shape();
      sole.moveTo(-0.065, -0.053);
      sole.lineTo(0.12, -0.053);
      sole.quadraticCurveTo(0.183, -0.053, 0.183, 0);
      sole.quadraticCurveTo(0.183, 0.053, 0.12, 0.053);
      sole.lineTo(-0.065, 0.053);
      sole.quadraticCurveTo(-0.087, 0.053, -0.087, 0);
      sole.quadraticCurveTo(-0.087, -0.053, -0.065, -0.053);
      const soleGeo = new THREE.ExtrudeGeometry(sole, { depth: 0.024, steps: 1, bevelEnabled: true, bevelSize: 0.006, bevelThickness: 0.006, bevelSegments: 1, curveSegments: 6 });
      soleGeo.rotateX(-Math.PI / 2);
      soleGeo.scale(0.62, 1, 0.68);
      const outsole = this.mesh(ankle, soleGeo, this.dark, [0, -0.069, 0]);
      outsole.name = `fencing-shoe-sole-${suffix}`;
    }
    const weapon = this.joint('weapon', this.bones.wristLead, 0, -0.065, 0);
    this.oval(weapon, this.dark, [0, 0.015, 0], [0.023, 0.09, 0.023]);
    // French grip, pommel, and shallow concave coquille. Weapon points down
    // in its own frame; aimWeapon aligns it independently of the elbow chain.
    this.oval(weapon, this.edge, [0, 0.1, 0], [0.03, 0.022, 0.03]);
    const guardGeo = new THREE.SphereGeometry(0.105, 24, 12, 0, Math.PI * 2, Math.PI / 2, Math.PI / 2);
    const guardMat = this.edge.clone();
    guardMat.side = THREE.DoubleSide;
    this.mesh(weapon, guardGeo, guardMat, [0, -0.075, 0], [1, 0.45, 1]);
    const rim = new THREE.Mesh(new THREE.TorusGeometry(0.105, 0.004, 6, 32), this.accent);
    rim.rotation.x = Math.PI / 2;
    rim.position.y = -0.075;
    weapon.add(rim);
    // Eight slim tapering segments flex independently under beat/touch load.
    this.bladeSegments = [];
    let parent = weapon;
    for (let i = 0; i < 8; i++) {
      const segment = new THREE.Group();
      segment.position.y = -0.11;
      parent.add(segment);
      const radius = 0.008 * (1 - i / 9);
      this.mesh(segment, new THREE.CylinderGeometry(radius, radius * 0.65, 0.11, 6), this.bladeMat, [0, -0.055, 0]);
      this.bladeSegments.push(segment);
      parent = segment;
    }
    this.bladeTip = new THREE.Object3D();
    this.bladeTip.position.y = -0.11;
    parent.add(this.bladeTip);
    this.oval(this.bladeTip, this.tipMaterial, [0, 0, 0], [0.006, 0.009, 0.006]);
    this.bladeBase = new THREE.Object3D();
    this.bladeBase.position.y = -0.11;
    weapon.add(this.bladeBase);
  }

  setOrbState(state) {
    if (!['neutral', 'win', 'lose'].includes(state)) throw new Error(`Unknown orb state: ${state}`);
    this.orbState = state;
    const off = state === 'lose';
    this.orbMaterial.color.setHex(off ? 0x081017 : this.color);
    this.orbMaterial.emissive.setHex(off ? 0x000000 : this.color);
    this.orbMaterial.emissiveIntensity = off ? 0 : state === 'win' ? 5 : 0.6;
    this.orbMaterial.toneMapped = state !== 'win';
    this.orbLight.intensity = off ? 0 : state === 'win' ? 0.16 : 0.012;
  }

  // Cancel leg pitch before applying outward foot yaw. Rear toes point toward
  // anatomical -Z, away from the lead leg, never inward across the stance.
  plant(suffix, footX, footY = 0.075, yaw = suffix === 'Rear' ? Math.PI / 2 : 0) {
    const p = this.bones.pelvis;
    const dx = footX - p.position.x;
    const dy = footY - (p.position.y - 0.09);
    const a = 0.55, b = 0.53;
    const d = THREE.MathUtils.clamp(Math.hypot(dx, dy), 0.1, a + b - 0.00001);
    const bend = Math.acos(THREE.MathUtils.clamp((d * d - a * a - b * b) / (2 * a * b), -1, 1));
    const hip = Math.atan2(dx, -dy) + Math.atan2(b * Math.sin(bend), a + b * Math.cos(bend));
    this.bones[`hip${suffix}`].rotation.z = hip;
    this.bones[`knee${suffix}`].rotation.z = -bend;
    this.bones[`ankle${suffix}`].quaternion.setFromAxisAngle(axisZ, bend - hip).multiply(footTurn.setFromAxisAngle(up, yaw));
  }

  aimWeapon(x, y, z) {
    this.group.updateMatrixWorld(true);
    v.set(x, y, z).normalize().transformDirection(this.group.matrixWorld);
    q.setFromUnitVectors(down, v);
    this.bones.weapon.parent.getWorldQuaternion(q2).invert();
    this.bones.weapon.quaternion.copy(q2.multiply(q));
  }

  // Two-bone weapon-arm IK. Author blade lines as a pair in world space;
  // never stretch limbs or move a detached sword to fake a collision.
  setBlade(base, direction) {
    const b = this.bones;
    const dir = direction.clone().normalize();
    const wrist = base.clone().addScaledVector(dir, -0.175);
    this.group.updateMatrixWorld(true);
    const shoulder = b.shoulderLead.getWorldPosition(new THREE.Vector3());
    const axis = wrist.clone().sub(shoulder);
    const distance = axis.length();
    if (distance > 0.76999 || distance < 0.02) throw new Error(`Unreachable weapon wrist: ${distance.toFixed(4)}`);
    axis.divideScalar(distance);
    // Choose the elbow plane that minimizes wrist deviation from the blade.
    // A fixed downward pole hyperflexes the wrist in low/prime parries.
    const pole = dir.clone().negate();
    pole.addScaledVector(axis, -pole.dot(axis));
    if (pole.lengthSq() < 1e-8) {
      pole.set(0, -1, 0.55).transformDirection(this.group.matrixWorld);
      pole.addScaledVector(axis, -pole.dot(axis));
    }
    pole.normalize();
    const along = (0.39 ** 2 - 0.38 ** 2 + distance ** 2) / (2 * distance);
    const elbow = shoulder.clone().addScaledVector(axis, along).addScaledVector(pole, Math.sqrt(Math.max(0, 0.39 ** 2 - along ** 2)));
    const orient = (bone, worldDirection) => {
      const parent = bone.parent.getWorldQuaternion(new THREE.Quaternion()).invert();
      bone.quaternion.copy(parent.multiply(new THREE.Quaternion().setFromUnitVectors(down, worldDirection.normalize())));
      bone.updateWorldMatrix(false, true);
    };
    orient(b.shoulderLead, elbow.clone().sub(shoulder));
    orient(b.elbowLead, wrist.clone().sub(elbow));
    orient(b.wristLead, dir.clone());
    b.weapon.quaternion.identity();
    this.bladeSegments.forEach(segment => segment.rotation.set(0, 0, 0));
    this.group.updateMatrixWorld(true);
  }

  pose(type, time = 0) {
    if (type === 'DEFLECT') type = 'DEFLECT_REACT';
    if (!poses.includes(type)) throw new Error(`Unknown pose: ${type}`);
    this.applyInterpolation(this.rest, this.rest, 0);
    const b = this.bones;
    const r = (name, x, y, z) => b[name].rotation.set(x, y, z);
    b.pelvis.position.set(0, 1.01, 0);
    // Stack the jacket over the pelvis: +X is forward, with pitch-only lean.
    r('spine', 0, 0, -0.025);
    r('chest', 0, 0, 0);
    r('neck', 0, 0, 0);
    // Cancel torso lean so guard looks level along the duel axis.
    r('head', 0, 0, 0.025);
    r('shoulderLead', -0.08, 0, 0.62);
    r('elbowLead', 0, 0, 1.05);
    // Relaxed athletic rear guard: elbow below shoulder, hand near the hip.
    r('shoulderRear', 0.18, -0.15, -0.25);
    r('elbowRear', -0.12, 0, 1.30);
    let lead = 0.40, rear = -0.40;
    let blade = [1, 0.16, -0.18];
    let flex = 0;
    let rearYaw = Math.PI / 2;
    switch (type) {
      case 'VICTORY':
      case 'DEFEAT':
      case 'ATTENTION': {
        const lost = type === 'DEFEAT';
        b.pelvis.position.set(0, lost ? 1.16 : 1.20, 0);
        r('spine', 0, 0, lost ? -0.17 : 0);
        r('head', 0, 0, lost ? -0.38 : 0);
        r('shoulderRear', 0.06, 0, -0.06);
        r('elbowRear', 0, 0, 0.12);
        r('shoulderLead', -0.10, 0, type === 'VICTORY' ? 2.65 : lost ? 0.12 : 0.42);
        r('elbowLead', 0, 0, type === 'VICTORY' ? 0.28 : lost ? 0.12 : 1.65);
        blade = type === 'VICTORY' ? [0.08, 1, 0.08] : lost ? [0.36, -0.88, 0.26] : [0.12, 1, 0.08];
        lead = rear = 0;
        rearYaw = 0;
        break;
      }
      case 'ON_GUARD': {
        // Classical high sixte/tierce: address the mask, not the sternum.
        blade = [0.84, 0.54, 0.12];
        const breath = Math.sin(time * 2.6);
        b.chest.position.y += breath * 0.006;
        b.pelvis.position.y += breath * 0.003;
        b.shoulderLead.rotation.z += breath * 0.008;
        break;
      }
      case 'SALUTE':
        r('shoulderLead', -0.08, 0, 0.50);
        r('elbowLead', 0, 0, 1.90);
        r('wristLead', 0, 0, -0.12);
        blade = [0.24, 0.96, -0.12];
        break;
      case 'WINDUP':
        // Fencing begins with the point: no sword-fighting backswing.
        b.pelvis.position.set(-0.055, 0.99, 0);
        r('shoulderLead', -0.06, 0, 0.88);
        r('elbowLead', 0, 0, 0.73);
        break;
      case 'STRIKE':
      case 'FOLLOW_THROUGH':
        b.pelvis.position.set(0.44, 0.75, 0);
        r('spine', 0, 0.08, -0.045);
        r('chest', 0, -0.08, 0);
        r('shoulderLead', 0, 0, 1.60);
        r('elbowLead', 0, 0, 0.015);
        r('shoulderRear', 0.08, 0, -0.99);
        r('elbowRear', 0, 0, -0.025);
        lead = 0.99; rear = -0.46;
        blade = [1, 0.025, -0.23];
        flex = type === 'FOLLOW_THROUGH' ? 0.045 : 0;
        break;
      case 'PARRY':
      case 'PARRY_QUARTE':
      case 'PARRY_PRIME':
        b.pelvis.position.set(-0.09, 0.99, 0);
        r('spine', 0, 0.10, 0.055);
        r('chest', 0, -0.10, 0);
        r('shoulderLead', type === 'PARRY_QUARTE' ? 0.48 : -0.22, 0, 0.72);
        r('elbowLead', 0, 0, type === 'PARRY_PRIME' ? 0.55 : 0.9);
        blade = type === 'PARRY_PRIME' ? [0.32, -0.85, -0.38] : type === 'PARRY_QUARTE' ? [0.48, 0.70, -0.5] : [0.52, 0.74, 0.3];
        lead = 0.32;
        break;
      case 'FEINT_EXTEND':
        b.pelvis.position.set(0.12, 0.99, 0);
        r('shoulderLead', -0.06, 0, 1.24);
        r('elbowLead', 0, 0, 0.36);
        lead = 0.54; rear = -0.28;
        blade = [1, -0.06, 0.08];
        break;
      case 'FEINT':
        b.pelvis.position.set(-0.15, 1.00, 0);
        r('spine', 0, 0.12, 0.06);
        r('shoulderLead', 0.07, 0, 0.93);
        r('elbowLead', 0, 0, 0.64);
        lead = 0.29; rear = -0.53;
        blade = [1, -0.22, -0.32];
        break;
      case 'HIT_REACT':
        b.pelvis.position.set(-0.27, 0.97, 0);
        r('spine', 0.04, 0.12, 0.20);
        r('chest', 0, -0.15, 0.10);
        r('head', 0, 0.10, 0.13);
        r('shoulderLead', 0.05, 0, 0.45);
        r('elbowLead', 0, 0, 0.65);
        r('shoulderRear', 0.28, 0, -0.45);
        r('elbowRear', 0, 0, 1.1);
        lead = 0.15; rear = -0.67;
        blade = [0.8, -0.7, -0.12];
        break;
      case 'DEFLECT_REACT':
        b.pelvis.position.set(-0.10, 1.00, 0);
        r('spine', 0, 0.12, 0.045);
        r('shoulderLead', -0.24, 0, 0.77);
        r('elbowLead', 0, 0, 1.05);
        blade = [0.65, 0.54, -0.4];
        flex = 0.09;
        break;
    }
    this.plant('Lead', lead);
    this.plant('Rear', rear, 0.075, rearYaw);
    this.footYaws = [0, rearYaw];
    this.aimWeapon(...blade);
    this.bladeSegments.forEach((segment, i) => { segment.rotation.z = flex * Math.sin((i + 1) * Math.PI / 9); });
    this.currentPose = type;
  }

  snapshot() {
    const snap = {};
    for (const [name, bone] of Object.entries(this.bones)) snap[name] = { pos: bone.position.clone(), quat: bone.quaternion.clone() };
    this.group.updateMatrixWorld(true);
    snap.bladeAim = this.group.getWorldQuaternion(new THREE.Quaternion()).invert().multiply(this.bones.weapon.getWorldQuaternion(new THREE.Quaternion()));
    snap.pose = this.currentPose;
    snap.footYaws = this.footYaws?.slice() || [0, Math.PI / 2];
    snap.bladeFlex = this.bladeSegments.map(segment => segment.rotation.z);
    return snap;
  }

  applyInterpolation(a, b, alpha) {
    alpha = THREE.MathUtils.clamp(alpha, 0, 1);
    if (alpha === 1 && b.pose) this.currentPose = b.pose;
    for (const [name, bone] of Object.entries(this.bones)) {
      bone.position.lerpVectors(a[name].pos, b[name].pos, alpha);
      bone.quaternion.slerpQuaternions(a[name].quat, b[name].quat, alpha);
    }
    if (a.feet && b.feet) {
      this.footYaws = a.footYaws.map((yaw, i) => THREE.MathUtils.lerp(yaw, b.footYaws[i], alpha));
      this.plant('Lead', THREE.MathUtils.lerp(a.feet[0], b.feet[0], alpha), 0.075, this.footYaws[0]);
      this.plant('Rear', THREE.MathUtils.lerp(a.feet[1], b.feet[1], alpha), 0.075, this.footYaws[1]);
    }
    if (a.bladeAim && b.bladeAim) {
      // A sword has one global aim, not the sum of independently interpolated
      // shoulder + elbow + wrist rolls (which can sweep it through a thigh).
      this.group.updateMatrixWorld(true);
      q.copy(a.bladeAim).slerp(b.bladeAim, alpha).premultiply(this.group.getWorldQuaternion(new THREE.Quaternion()));
      this.bones.weapon.parent.getWorldQuaternion(q2).invert();
      this.bones.weapon.quaternion.copy(q2.multiply(q));
    }
    this.bladeSegments.forEach((segment, i) => {
      segment.rotation.z = THREE.MathUtils.lerp(a.bladeFlex[i], b.bladeFlex[i], alpha);
    });
  }

  capture(type) {
    this.pose(type);
    const snap = this.snapshot();
    this.group.updateMatrixWorld(true);
    const inverse = this.group.matrixWorld.clone().invert();
    snap.feet = ['Lead', 'Rear'].map(side => this.bones[`ankle${side}`].getWorldPosition(new THREE.Vector3()).applyMatrix4(inverse).x);
    return snap;
  }

  // Render-time elastic rebound is separate from pose snapshots; no cumulative drift.
  animateBlade(time, amplitude = 0) {
    this.bladeSegments.forEach((segment, i) => {
      segment.rotation.x = amplitude * Math.sin(time * 65 - i * 0.6) * (i + 1) / 8;
    });
  }
}
