import * as THREE from 'three';
import { StudioRig } from './rig.js';
import { CAPTURE, TIMING, renderSize } from './capture.js';
import { FrameRecording } from './recording.js';
import { ContactSparks } from './sparks.js';
import { StudioCamera } from './camera.js';
import { RoundResult, resultLabel } from './result.js';
import { FIGHTER_COLORS, FIGHTER_NAMES } from './palette.js';
import { buildEngagement, bladeContact, wins, applyCombatFrame } from './combat.js';

const container = document.getElementById('viewport-container');
const canvas = document.getElementById('canvas3d');
const scene = new THREE.Scene();
scene.background = new THREE.Color(0x080c14);
const cameraRig = new StudioCamera(canvas);
let camera = cameraRig.camera;
const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, preserveDrawingBuffer: true });
renderer.setPixelRatio(1);
// No floor or scenery receives a useful shadow. Avoid a second full rig pass
// and a 2048² shadow map on software GPUs; direct/rim/core lighting is retained.
renderer.shadowMap.enabled = false;
renderer.shadowMap.type = THREE.PCFSoftShadowMap;
renderer.toneMapping = THREE.ACESFilmicToneMapping;
renderer.toneMappingExposure = 1.25;
scene.add(new THREE.HemisphereLight(0xa5c8ff, 0x1b1626, 1.3));
const key = new THREE.DirectionalLight(0xe2edff, 3.8);
key.position.set(-1, 6, 4);
key.castShadow = true;
key.shadow.mapSize.set(2048, 2048);
Object.assign(key.shadow.camera, { left: -4, right: 4, top: 4, bottom: -4, near: 0.5, far: 18 });
key.shadow.normalBias = 0.025;
key.shadow.bias = -0.0003;
scene.add(key);
const rim = new THREE.DirectionalLight(0x8b9fff, 4.5);
rim.position.set(1, 3, -4);
scene.add(rim);
const lights = FIGHTER_COLORS.map((color, i) => {
  const light = new THREE.PointLight(color, 13, 8, 2);
  light.position.set(i ? 2.7 : -2.7, 1.8, -0.6);
  scene.add(light);
  return light;
});
// Fighters and transient blade effects only: no arena or ground geometry.
const botA = new StudioRig(FIGHTER_COLORS[0], true);
const botB = new StudioRig(FIGHTER_COLORS[1], false);
botA.group.position.x = -1.10;
botB.group.position.x = 1.10;
scene.add(botA.group, botB.group);
const rigs = [botA, botB];
const roundResult = new RoundResult(scene, rigs);
function clearResult() { roundResult.clear(); }
function showResult(winner) { return roundResult.show(winner, camera); }
let currentStanceA = 'ROCK', currentStanceB = 'PAPER';
let isAnimating = false, busy = false, batch = false, idle = true;
let frameRecording = null;
const timelineNow = () => frameRecording ? frameRecording.time : performance.now();
let shake = 0, trailActive = false, bladeRebound = 0;
const guardSelect = document.getElementById('guard-line');
const status = document.getElementById('status-log');
const playBtn = document.getElementById('btn-play');
const recordBtn = document.getElementById('btn-record-one');
const batchBtn = document.getElementById('btn-batch-export');
const idleBtn = document.getElementById('btn-idle');
let engagementPhase = 'READY';
function cue(phase) { engagementPhase = phase; }
const cameraButtons = [...document.querySelectorAll('[data-camera]')];
cameraButtons.forEach(button => button.addEventListener('click', () => {
  if (busy) return;
  cameraRig.select(button.dataset.camera);
  camera = cameraRig.camera;
  cameraButtons.forEach(item => {
    const active = item === button;
    item.classList.toggle('active', active);
    item.setAttribute('aria-pressed', String(active));
  });
}));
const stanceButtons = [...document.querySelectorAll('[data-stance]')];
const ease = {
  smooth: t => t * t * (3 - 2 * t),
  in: t => t * t * t,
  out: t => 1 - (1 - t) ** 3,
  linear: t => t,
};
function log(message) { status.textContent = message; }
function lockControls(value) {
  busy = value;
  // Freeze the actual orbit/zoom, not a preset. Do not advance damping while locked.
  cameraRig.setLocked(value);
  for (const button of [...stanceButtons, playBtn, idleBtn, recordBtn, batchBtn, guardSelect, ...cameraButtons]) button.disabled = value;
  if (!value) resize();
}
function updateButtons(preview = true) {
  stanceButtons.forEach(button => button.classList.toggle('active', button.dataset.stance === (button.closest('#botA-btns') ? currentStanceA : currentStanceB)));
  document.getElementById('matchup-title').textContent = `${currentStanceA} / ${currentStanceB}`;
  document.getElementById('matchup-detail').textContent = currentStanceA === currentStanceB ? 'MUTUAL CLASH · NO ADVANTAGE' : `${wins(currentStanceA, currentStanceB) ? FIGHTER_NAMES[0] : FIGHTER_NAMES[1]} ADVANTAGE · ${wins(currentStanceA, currentStanceB) ? currentStanceA : currentStanceB} COUNTERS ${wins(currentStanceA, currentStanceB) ? currentStanceB : currentStanceA}`;
  if (preview) {
    clearResult();
    idle = false;
    const preview = buildEngagement(rigs, [currentStanceA, currentStanceB], guardSelect.value);
    const snapshots = preview.winner !== null && [currentStanceA, currentStanceB][preview.winner] === 'SCISSORS' ? preview.action : preview.contact;
    rigs.forEach((rig, i) => rig.applyInterpolation(snapshots[i], snapshots[i], 1));
    clearTrails();
  }
}
stanceButtons.forEach(button => button.addEventListener('click', () => {
  if (busy) return;
  if (button.closest('#botA-btns')) currentStanceA = button.dataset.stance;
  else currentStanceB = button.dataset.stance;
  updateButtons();
}));
guardSelect.addEventListener('change', () => updateButtons());
updateButtons(false);

// Fixed-size ribbon buffers. All coordinates are world-space; no per-frame
// geometry/material creation and no trail linking across separate attacks.
const TRAIL_SAMPLES = 12;
const trails = rigs.map(rig => {
  const positions = new Float32Array((TRAIL_SAMPLES - 1) * 18);
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3).setUsage(THREE.DynamicDrawUsage));
  const material = new THREE.MeshBasicMaterial({ color: rig.color, side: THREE.DoubleSide, transparent: true, opacity: 0.12, blending: THREE.AdditiveBlending, depthWrite: false });
  const mesh = new THREE.Mesh(geometry, material);
  mesh.frustumCulled = false;
  scene.add(mesh);
  return { mesh, positions, samples: Array.from({ length: TRAIL_SAMPLES }, () => [new THREE.Vector3(), new THREE.Vector3()]), count: 0 };
});
function clearTrails() { for (const trail of trails) { trail.count = 0; trail.mesh.visible = false; } }
function updateTrails() {
  for (let j = 0; j < rigs.length; j++) {
    const trail = trails[j];
    if (!trailActive) { trail.mesh.visible = false; trail.count = 0; continue; }
    const recycled = trail.samples.pop();
    rigs[j].bladeBase.getWorldPosition(recycled[0]);
    rigs[j].bladeTip.getWorldPosition(recycled[1]);
    trail.samples.unshift(recycled);
    trail.count = Math.min(trail.count + 1, TRAIL_SAMPLES);
    let offset = 0;
    for (let i = 0; i < trail.count - 1; i++) {
      const [a, b] = trail.samples[i], [c, d] = trail.samples[i + 1];
      for (const v of [a, b, c, b, d, c]) { v.toArray(trail.positions, offset); offset += 3; }
    }
    trail.mesh.geometry.setDrawRange(0, (trail.count - 1) * 6);
    trail.mesh.geometry.attributes.position.needsUpdate = true;
    trail.mesh.visible = true;
  }
}
const contactSparks = new ContactSparks(scene);
let lastContact = null;
function impact(engagement) {
  if (engagement.kind === 'read') {
    lastContact = { kind: 'read', distance: null, point: null, fractions: null };
    return;
  }
  const hit = engagement.kind === 'blade' ? bladeContact(...rigs) : null;
  if (hit && hit.distance > 0.0001) throw new Error(`Blade contact missed by ${hit.distance.toFixed(4)}`);
  const point = hit ? hit.point : rigs[engagement.winner].bladeTip.getWorldPosition(new THREE.Vector3());
  const attacker = rigs[engagement.winner ?? 0];
  const vector = attacker.bladeTip.getWorldPosition(new THREE.Vector3()).sub(attacker.bladeBase.getWorldPosition(new THREE.Vector3()));
  contactSparks.burst(point, vector, engagement.winner);
  lastContact = { kind: engagement.kind, point: point.toArray(), distance: hit?.distance ?? point.distanceTo(engagement.point), fractions: hit?.fractions ?? null };
  shake = engagement.winner === null ? 0.018 : 0.025;
}

function tween(from, to, duration, easing = ease.smooth, disengage = false, drive = false) {
  return new Promise((resolve, reject) => {
    const start = timelineNow();
    function step() {
      const now = timelineNow();
      try {
        if (frameRecording?.error) throw frameRecording.error;
        const t = Math.min((now - start) / duration, 1);
        rigs.forEach((rig, i) => {
          const alpha = easing(t);
          applyCombatFrame(rig, from[i], to[i], t, alpha, {
            disengage: disengage && [currentStanceA, currentStanceB][i] === 'SCISSORS', drive,
          });
        });
        if (t < 1) requestAnimationFrame(step); else resolve();
      } catch (error) { reject(error); }
    }
    requestAnimationFrame(step);
  });
}
const hold = ms => frameRecording ? frameRecording.hold(ms) : new Promise(resolve => setTimeout(resolve, ms));
async function playSequence(a, b, guardLine = guardSelect.value) {
  if (isAnimating) return;
  isAnimating = true;
  clearResult();
  idle = false;
  try {
    const stances = [a, b];
    const capture = types => rigs.map((rig, i) => rig.capture(types[i] === 'PAPER' ? guardLine : types[i]));
    const guard = capture(['ON_GUARD', 'ON_GUARD']);
    const salute = capture(['SALUTE', 'SALUTE']);
    const windup = capture(stances.map(s => s === 'ROCK' ? 'WINDUP' : 'ON_GUARD'));
    const engagement = buildEngagement(rigs, stances, guardLine);
    const { action, contact, winner } = engagement;
    const tie = winner === null;
    const hasScissors = stances.includes('SCISSORS');
    const counter = !tie && stances[winner] === 'SCISSORS';
    // Never visit an unopposed full thrust before its defensive interception.
    const bait = capture(stances.map(s => s === 'SCISSORS' ? 'SCISSORS_EXTEND' : s === 'ROCK' ? 'WINDUP' : 'PAPER'));
    const reactions = stances.map((s, i) => tie ? 'DEFLECT_REACT' : i === winner ? (s === 'PAPER' ? 'PAPER' : 'ON_GUARD') : stances[winner] === 'PAPER' ? 'DEFLECT_REACT' : 'HIT_REACT');
    const react = capture(reactions);
    // Withdraw the point from the jacket while the struck torso recoils.
    if (engagement.kind === 'body') {
      rigs[winner].applyInterpolation(contact[winner], guard[winner], 0.12);
      react[winner] = rigs[winner].snapshot();
      react[winner].feet = contact[winner].feet.map((x, j) => THREE.MathUtils.lerp(x, guard[winner].feet[j], 0.12));
    }
    bladeRebound = 0;
    lastContact = null;
    log(`PHYSICAL SALUTE → BLADE TOUCH · ${a} / ${b}`);
    rigs.forEach((rig, i) => rig.applyInterpolation(guard[i], guard[i], 0));
    clearTrails();
    cue('SALUTE');
    await tween(guard, salute, 420, ease.smooth);
    cue('READY');
    await hold(380);
    cue('SET');
    await tween(salute, guard, 300, ease.out);
    await hold(180);
    cue('ENGAGE');
    await tween(guard, windup, 130, ease.out);
    trailActive = true;
    if (hasScissors) {
      await tween(windup, bait, 190, ease.out);
      await tween(bait, counter || tie ? action : contact, 185, counter || tie ? ease.smooth : ease.in, true, !counter && !tie);
      if (counter) {
        await hold(80);
        await tween(action, contact, 185, ease.in, true, true);
      }
    } else {
      await tween(windup, contact, tie && a === 'PAPER' ? 360 : 230, tie && a === 'PAPER' ? ease.smooth : ease.in, false, a === 'ROCK' || b === 'ROCK');
    }
    cue('CLASH');
    trailActive = false;
    const mutualRead = tie && a !== 'ROCK';
    impact(engagement);
    log(tie ? (mutualRead ? 'MUTUAL READ · NO COMMIT' : 'CLASH · MUTUAL REBOUND') : `${winner === 0 ? FIGHTER_NAMES[0] : FIGHTER_NAMES[1]} · ${stances[winner] === 'PAPER' ? 'BLADE DEFLECTION' : 'CONFIRMED HIT'}`);
    // Hold exact contact geometry before any elastic blade recoil.
    await hold(55);
    bladeRebound = engagement.kind === 'blade' ? 0.065 : 0.025;
    await tween(contact, mutualRead ? guard : react, 180, ease.out);
    await hold(420);
    cue('OUTCOME');
    await showResult(tie ? null : winner);
    await hold(TIMING.outcomeHoldMs);
    cue('RECOVER');
    await roundResult.returnToGuard();
    await hold(TIMING.guardHoldMs);
    idle = true;
    log(`${resultLabel(tie ? null : winner)} · ON GUARD / READY`);
  } finally {
    trailActive = false;
    clearTrails();
    isAnimating = false;
    cue('READY');
  }
}
playBtn.addEventListener('click', async () => {
  if (busy) return;
  lockControls(true);
  try { await playSequence(currentStanceA, currentStanceB); }
  catch (error) { log(`Playback failed: ${error.message}`); console.error(error); }
  finally { lockControls(false); }
});
idleBtn.addEventListener('click', () => {
  clearResult();
  idle = true;
  rigs.forEach(rig => rig.pose('ON_GUARD'));
  clearTrails();
  log('ON GUARD · FOOT LOCK / BREATHING / TARGET TRACK');
});

const downloadURLs = [];
function addDownload(filename, blob) {
  const url = URL.createObjectURL(blob);
  downloadURLs.push(url);
  const item = document.createElement('div');
  item.className = 'clip-item';
  const label = document.createElement('span');
  label.textContent = filename;
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.textContent = 'Download';
  item.append(label, link);
  document.getElementById('clip-downloads').prepend(item);
}
async function recordMatchup(a, b, guardLine = guardSelect.value) {
  if (await FrameRecording.supported(canvas.width, canvas.height)) {
    const recording = new FrameRecording(canvas.width, canvas.height);
    frameRecording = recording;
    recordBtn.classList.add('recording');
    try {
      await playSequence(a, b, guardLine);
      await hold(CAPTURE.drainMs);
      frameRecording = null;
      const blob = await recording.finish();
      if (!blob.size) throw new Error('Encoder produced an empty clip.');
      addDownload(`${a.toLowerCase()}_vs_${b.toLowerCase()}.webm`, blob);
      log(`EXPORTED ${a} / ${b} · ${(blob.size / 1024).toFixed(0)} KB · ${CAPTURE.fps} FPS`);
    } finally {
      frameRecording = null;
      recording.close();
      recordBtn.classList.remove('recording');
    }
    return;
  }
  if (!canvas.captureStream || !window.MediaRecorder) throw new Error('WebM capture is not supported in this browser.');
  const mimeType = ['video/webm;codecs=vp8', 'video/webm;codecs=vp9', 'video/webm'].find(type => MediaRecorder.isTypeSupported(type));
  if (!mimeType) throw new Error('No supported WebM encoder.');
  const stream = canvas.captureStream(CAPTURE.fps);
  let recorder;
  try {
    recorder = new MediaRecorder(stream, { mimeType, videoBitsPerSecond: CAPTURE.bitrate });
    const chunks = [];
    const stopped = new Promise((resolve, reject) => {
      recorder.ondataavailable = event => { if (event.data.size) chunks.push(event.data); };
      recorder.onstop = () => resolve(new Blob(chunks, { type: mimeType }));
      recorder.onerror = event => reject(event.error || new Error('Encoder failed.'));
    });
    // Attach a handler immediately; encoding errors may precede sequence completion.
    stopped.catch(() => {});
    recordBtn.classList.add('recording');
    recorder.start();
    await playSequence(a, b, guardLine);
    // Submit the exact final guard, then allow a few capture ticks to drain.
    // The bounded buffer no longer needs the old 1.2-second software-GPU tail.
    renderer.render(scene, camera);
    renderer.getContext().finish();
    stream.getVideoTracks()[0]?.requestFrame?.();
    // Readback makes the final drawing buffer observable to captureStream.
    await new Promise(resolve => canvas.toBlob(resolve));
    await hold(CAPTURE.drainMs);
    if (recorder.state !== 'inactive') recorder.stop();
    const blob = await stopped;
    if (!blob.size) throw new Error('Encoder produced an empty clip.');
    addDownload(`${a.toLowerCase()}_vs_${b.toLowerCase()}.webm`, blob);
    log(`EXPORTED ${a} / ${b} · ${(blob.size / 1024).toFixed(0)} KB`);
  } finally {
    if (recorder && recorder.state !== 'inactive') recorder.stop();
    stream.getTracks().forEach(track => track.stop());
    recordBtn.classList.remove('recording');
  }
}
recordBtn.addEventListener('click', async () => {
  if (busy) return;
  lockControls(true);
  try { await recordMatchup(currentStanceA, currentStanceB); }
  catch (error) { log(`Capture failed: ${error.message}`); console.error(error); }
  finally { lockControls(false); }
});
batchBtn.addEventListener('click', async () => {
  if (busy) return;
  const selectedStances = [currentStanceA, currentStanceB];
  const selectedGuard = guardSelect.value;
  batch = true;
  lockControls(true);
  try {
    for (const a of ['ROCK', 'SCISSORS', 'PAPER']) for (const b of ['ROCK', 'SCISSORS', 'PAPER']) {
      currentStanceA = a; currentStanceB = b;
      updateButtons(false);
      await recordMatchup(a, b, selectedGuard);
    }
    log('BATCH COMPLETE · ALL 9 WEBM CLIPS READY');
  } catch (error) { log(`Batch stopped: ${error.message}`); console.error(error); }
  finally {
    [currentStanceA, currentStanceB] = selectedStances;
    updateButtons(false);
    batch = false;
    lockControls(false);
  }
});
window.addEventListener('beforeunload', () => downloadURLs.forEach(url => URL.revokeObjectURL(url)));

function resize() {
  // A stream/encoder cannot change dimensions mid-clip. Apply pending resize
  // on unlock instead; preserve the exact selected camera for the whole batch.
  if (busy) return;
  cameraRig.resize(container.clientWidth / container.clientHeight);
  renderer.setSize(...renderSize(container.clientWidth, container.clientHeight), false);
}
new ResizeObserver(resize).observe(container);
// Apply the initial preset at the measured viewport aspect.
resize();
cameraRig.select(cameraRig.preset);
let previous = performance.now();
async function animate(now) {
  const elapsed = frameRecording ? 1 / CAPTURE.fps : (now - previous) / 1000;
  const dt = Math.min(elapsed, 0.05);
  previous = now;
  const time = timelineNow() / 1000;
  if (idle && !isAnimating) {
    botA.pose('ON_GUARD', time);
    botB.pose('ON_GUARD', time + 1.4);
  }
  cameraRig.update(time, shake);
  shake *= Math.exp(-dt * 18);
  contactSparks.update(elapsed, camera);
  bladeRebound *= Math.exp(-dt * 12);
  rigs.forEach(rig => rig.animateBlade(time, bladeRebound));
  scene.updateMatrixWorld(true);
  updateTrails();
  roundResult.update(elapsed);
  renderer.render(scene, camera);
  try {
    if (frameRecording) await frameRecording.capture(canvas);
  } catch (error) {
    // Keep the render loop alive so the sequence can unwind and release locks.
    console.error(error);
    frameRecording.error = error;
    frameRecording.time += 1000 / CAPTURE.fps;
    frameRecording.waiters.splice(0).forEach(waiter => waiter.reject(error));
  }
  requestAnimationFrame(animate);
}
requestAnimationFrame(animate);
log('VOID STUDIO ONLINE · PHYSICAL SALUTE / BLADE TOUCH');
// Small read-only inspection surface for smoke tests; no scene mutation API.
window.studioDiagnostics = () => ({
  animating: isAnimating, busy, batch, idle, engagementPhase,
  camera: cameraRig.preset, projection: camera.type,
  result: { active: roundResult.active, winner: roundResult.winner,
    settled: roundResult.settled, phase: roundResult.phase, spotlight: { visible: false, intensity: 0 }, beamVisible: false },
  contact: { visible: contactSparks.group.visible, particles: contactSparks.positions.length / 3,
    life: contactSparks.life, measured: lastContact, origin: contactSparks.flare.position.toArray(), flare: contactSparks.flare.material.uniforms.power.value,
    light: contactSparks.light.intensity, tint: contactSparks.tint.getHex(), winner: contactSparks.winner, streaks: true },
  viewportTextSprites: scene.children.filter(object => object.isSprite).length,
  orbit: { locked: !!cameraRig.locked, enabled: cameraRig.controls.enabled, target: cameraRig.controls.target.toArray(), zoom: camera.zoom },
  cameraPosition: camera.position.toArray(),
  sceneryMeshes: scene.children.filter(object => object.isMesh && !trails.some(trail => trail.mesh === object)).length,
  stances: [currentStanceA, currentStanceB], guardLine: guardSelect.value,
  capture: { width: canvas.width, height: canvas.height, fps: CAPTURE.fps, bitrate: CAPTURE.bitrate,
    deterministic: !!frameRecording, frame: frameRecording?.frame ?? null, time: timelineNow() },
  drawCalls: renderer.info.render.calls,
  triangles: renderer.info.render.triangles,
  rigs: rigs.map(rig => ({ color: rig.color, visor: !!rig.group.getObjectByName('colored-eye-strip'), mask: !!rig.group.getObjectByName('woven-mask'),
    position: rig.group.position.toArray(),
    facing: new THREE.Vector3(1, 0, 0).applyQuaternion(rig.group.quaternion).toArray(),
    bladeBase: rig.bladeBase.getWorldPosition(new THREE.Vector3()).toArray(),
    bladeTip: rig.bladeTip.getWorldPosition(new THREE.Vector3()).toArray(),
    headPitch: rig.bones.head.rotation.z,
    headFacing: new THREE.Vector3(1, 0, 0).applyQuaternion(rig.bones.head.getWorldQuaternion(new THREE.Quaternion())).toArray(),
    torsoYaw: [rig.bones.spine.rotation.y, rig.bones.chest.rotation.y],
    footDirections: ['Lead', 'Rear'].map(side => new THREE.Vector3(1, 0, 0).applyQuaternion(rig.bones[`ankle${side}`].getWorldQuaternion(new THREE.Quaternion())).toArray()),
    coreDepth: rig.chestOrb.position.x, coreFront: rig.chestOrb.position.x + rig.chestOrb.scale.x, joints: Object.keys(rig.bones).length, pose: rig.currentPose,
    orb: { state: rig.orbState, intensity: rig.orbMaterial.emissiveIntensity, emissive: rig.orbMaterial.emissive.getHex() },
    rightHanded: rig.bones.weapon.parent === rig.bones.wristLead && rig.bones.clavicleLead.position.z > 0,
    feet: ['Lead', 'Rear'].map(side => rig.bones[`ankle${side}`].getWorldPosition(new THREE.Vector3()).toArray()),
  })),
});
