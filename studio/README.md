# BOTTAGON Studio — Best Seat

Live: https://jediui.com/studio/

## Run / test / deploy

```sh
npm ci
npm run dev
npm test
npm run build
cp -a dist/. /var/www/jediui.com/public/studio/
```

Vite base is `/studio/`. No external models, textures, fonts, or images.
Back up the deployed directory before copying; existing rollback snapshots are
in the ignored `deploy-backups/` directory. Unrelated project changes are untouched.

## Scene and outcome presentation

Two sleek procedural fencers in a dark void. No floor geometry, stage, grid,
result rings, floor circles, winner auras, viewport text sprites, spotlights,
or winner beam meshes. Stance/result messages remain outside canvas recordings.

`src/result.js` transitions both rigs over 850 ms into a shared camera-facing
plane, 1.6 units apart. The victor stands tall with sword hand above the head
and blade raised high. The defeated bot bows its head, slumps, and lowers its
sword beside its body. Ties use matching upright attention/salute poses.
The final tableau holds for 450 ms, then both fighters smoothly recover over
425 ms to their original duel positions/orientations and authentic ON_GUARD.
Timeline transitions await actual render-loop completion rather than assuming
a fixed number of frames; slow export rendering still includes the full result.
Reset also restores original duel positions/orientations. The active camera's horizontal direction at outcome
resolution defines the shared facing, including manually orbited cameras.

## Cameras

`src/camera.js` owns six aspect-aware presets:

- **God View · Best Seat** — the default. Elevated approximately 35 degrees,
  looking down along the duel's side with a small negative-X offset. Shows both
  right-handed weapon chains, blades, chest cores, and footwork. Guard torsos
  open slightly toward the viewing side without mirroring either bot's anatomy.
  Narrow screens increase both distance and elevation to preserve the angle.
- **Broadcast Left** — stable orthographic full-fighter framing.
- **Dynamic 3/4** — shallow front-left perspective.
- **Low Angle Dramatic** — below the fighters' aim point.
- **Close Clash** — intentionally tighter weapon-engagement framing.
- **Stadium View** — higher/wider elevated stands perspective.

OrbitControls stays active: mouse/touch drag rotates, wheel/pinch zooms, no pan.
Presets reset camera position, target, zoom, and pending drag inertia. User orbit
is not overwritten by the animation loop. Playback/recording freezes the exact current orbit and zoom, including pending
damping; camera preset buttons and OrbitControls are locked until completion. God View and Broadcast fit whole fighters and raised salutes
across tested portrait/square/wide aspect ratios.

## Rig / energy / foot contact

`src/rig.js` builds each fencer with 20 pivots, slim tapered limbs, a fitted
jacket, clean bare silver helmet with no visor, light strip, mask or fixtures, French grip, bell guard, and eight flexing blade
segments. Lower tessellation, narrower profiles, and removed chest/limb overlays
keep the silhouette clean. Local +X points at the opponent, +Y is up, and +Z is
the anatomical right-hand side. Both swords attach to the right wrist. Bot B
rotates 180 degrees about Y; it is never negatively scaled or reflected.

Each chest has one circular core: cobalt blue (`#0044ff`) for A, blood red
(`#990000`) for B. The continuous sternum mesh curves inward by ~0.065 units
behind a narrow socket rim. Core center X is 0.092 (previously 0.139); its
frontmost point is 0.124 (previously 0.184). Its material and
short-range local light are independent of the sword tip:

- **Neutral / tie / reset:** colored low energy, emissive intensity 0.6.
- **Win:** intense emissive intensity 5 with brighter local chest illumination.
- **Lose:** black emissive color, zero emission and zero local light. The unlit
  core stays physically present; the rest of the bot is not artificially dimmed.

The sword-tip marker is half the previous radius (0.006), with restrained
emission rather than an oversized glowing ball.

Shoes are compact minimal uppers over thin flat rounded outsoles: sole length
and width are reduced to 62% and 68% respectively. Sagittal two-bone IK cancels
leg rotation before applying ankle yaw. Rear toes turn **outward toward local
-Z**, away from the lead leg (the former -90° yaw incorrectly turned them inward).
Both rigs rotate without negative scale. Outcome feet align parallel beneath
the hips; foot yaw and IK interpolate together so soles remain level at Y=0.
Ankle height remains 0.075. Thigh + shin reach is 1.08.

**ON_GUARD:** bent knees, rear elbow below the shoulder, relaxed rear hand, and
high sixte/tierce point at mask/face level instead of horizontally at the chest.
Other authored poses: SALUTE, WINDUP, ROCK, FOLLOW_THROUGH, PAPER (Sixte),
PAPER_QUARTE, PAPER_PRIME, SCISSORS_EXTEND, SCISSORS, HIT_REACT, DEFLECT_REACT, VICTORY, DEFEAT, ATTENTION.
`aimWeapon()` aligns the blade independently of the arm chain.

## Playback / export

Playback, single WebM recording, and all-nine batch export share one timeline:

1. Raise both swords into a physical salute (420 ms), hold (380 ms).
2. Return to high guard (300 ms), hold (180 ms).
3. Point-first windup, extension, optional Scissors disengage and renewed attack.
4. Resolve the game outcome with blade sparks/recoil.
5. Turn/reposition both bots side by side facing the camera into victory/defeat
   poses, or shared attention/salute for ties (850 ms transition, 450 ms hold).
6. Smoothly recover both fighters to their original ON_GUARD pose, position,
   facing, foot yaw and neutral core energy (425 ms), then settle for 80 ms.

No countdown or start-text overlay. This is a physical salute, not a blade-touch
prelude. Defence is blade-only. The poses and rock-paper-scissors outcome rules
are authored animation, not collision detection or official fencing scoring.

Blade impacts emit 48 restrained directional particles with short tapered
streaks and a compact camera-facing flare. All channels (particles, streaks,
flare and contact light) use cobalt blue for Bot A wins, blood red for Bot B
wins, and neutral white/gold for ties. Particle size caps at 5 px, lifetime is
460 ms, flare starts at 0.28 units and contact-light peak is 3 (previously 65).
Deterministic opposing fans follow the attacking blade vector with a compact
velocity spread. GPU buffers and
materials are reused across all contacts; no textures or external assets.

All controls and camera orbit/zoom lock during execution. Batch export snapshots
the selected guard line once and passes it explicitly to all nine timelines. It
never selects a camera preset or resets camera position/target/zoom. Original
stance selections are restored after the batch, including on failure. Unlock
rebinds OrbitControls to the same camera/target/zoom, discarding stale drag
inertia so the view cannot drift once export finishes. WebM negotiates
VP8/VP9 and includes the physical outcome poses and chest cores. The drawing
buffer is capped at **720×660**, DPR 1, preserving the viewport's aspect ratio.
Empty-stage shadow mapping is disabled; direct/rim/core lighting remains.

**WebCodecs is the preferred exporter**: `src/recording.js` encodes every rendered
image with a consecutive 30 FPS timestamp and muxes a seekable WebM. During
export, tweens, holds, breathing, sparks, recoil and recovery use a frame-stepped
clock. Software GPUs take longer to export, but do not skip animation frames.
VP8 target bitrate is **400 kbps**; the encoder queue is bounded and each
VideoFrame is closed immediately. Final guard has an 80 ms settle plus a 100 ms
capture tail. Encoder errors propagate and all controls unlock on failure.
Canvas dimensions cannot change in the middle of an export/batch.

Browsers without WebCodecs fall back to MediaRecorder at 30 FPS/400 kbps,
with stream tracks released on success/failure. This fallback remains real-time
and cannot guarantee 30 distinct frames on a software GPU. The UI's requested
FPS is not evidence of actual cadence; verify decoded frame timestamps.

## Verification

```sh
npm test
npm run build
PLAYWRIGHT_MODULE=/path/to/playwright/index.mjs \
CHROMIUM_EXECUTABLE=/path/to/chromium \
node test/browser-smoke.mjs
```

Omit browser overrides when matching Playwright/Chromium are installed normally.
For immutable local checks, run `npm run build && npm run preview -- --host 127.0.0.1`
and set `STUDIO_URL=http://127.0.0.1:4173/studio/`. Avoid editing source against
a Vite dev-server smoke run: HMR resets page/recording state during the suite.

- **67 unit tests:** six cameras/five aspect ratios, full God/Broadcast outcome
  framing, orbit persistence/reset, all 225 rig pose-pair interpolations on both
  rigs, level soles/exact ground contact, right-handedness, outward rear-toe
  orientation, compact foot bounds, bare helmet, chest recess/energy, smooth
  side-by-side camera-facing outcomes, high victory/low defeat blades, tie salute,
  and reset/validation, exact smooth guard recovery, directional deterministic
  sparks, all outcome tints, restrained flare intensity, buffer reuse, fade-out
  and camera/damping freeze plus inertia-free unlock. No result lighting/beam objects may exist in the scene.
- Browser smoke: all nine outcomes, six presets, real mouse drag/wheel, mobile
  touch orbit/pinch, single WebM capture/decode/download, and two complete nine-clip batches (manually orbited/zoomed God View +
  Quarte and Broadcast + Prime). Monitors every clip for camera/guard lock,
  blocked camera interaction, ~450 ms logical outcome hold, guard recovery and restored
  stance selection. Fails on console errors, page exceptions, or HTTP errors.
- Screenshots: `/tmp/studio-god-guard.png`, `/tmp/studio-result-{0,1,tie}.png`,
  `/tmp/studio-clean-victor-hold.png`, `/tmp/studio-god-mobile.png`.
- Downloaded video proof: `/tmp/studio-god-view.webm`.
- Vite's existing >500 kB chunk advisory is non-blocking.

`window.studioDiagnostics()` is read-only and reports playback state, selected
camera, orbit, outcome/settled state, rig facing/position, blade endpoints, head pitch,
foot directions, core energy, right-handed
attachment checks, foot positions, and render statistics. It exposes no mutation
API. Screenshot and WebM evidence are test artifacts, not public site assets.

## Live outcome-pose release — 2026-10-20 22:54 UTC

Deployed to https://jediui.com/studio/. Public bundle SHA-256 exactly matches
`dist/assets/index-CzGQ5V45.js`:
`f33f1dafeddd1b4f5f16f49d26840690561bd19802d327a5b84452f08f1152df`.

- Winner spotlight/beam and visor strip geometry removed entirely.
- Persistent camera-facing, side-by-side VICTORY / DEFEAT / ATTENTION outcomes.
- Rear foot yaw corrected outward on both rigs; soles 38% shorter, 32% narrower.
- Broadcast framing expanded to include the full overhead victory sword.
- **25/25 unit tests**, production build, immutable-preview browser suite and
  full live Chromium suite passed. All nine outcomes, six presets, desktop
  mouse orbit/wheel, mobile touch orbit/pinch and playback passed.
- Live console/page/HTTP errors: **0**.
- Single VP9 WebM: **256,345 bytes**, **1080 × 1000**, **4.276 seconds**;
  browser decode/seek/download and ffprobe passed. Final frame at 4.05 seconds
  decoded with ffmpeg and visually inspected: requested tableau is in the video.
- All nine unique batch WebM exports passed; sizes **223,438–296,885 bytes**.
- Only build advisory: existing Vite >500 kB chunk warning.

Evidence: `/home/rico/.openclaw/agents/carmack/workspace/artifacts/bottagon-outcome-poses-20261020/`.
Rollback: `deploy-backups/20261020T225114Z-pre-outcome-poses/`.

## Historical releases (superseded by current outcome poses)

### Live result — 2026-10-20 22:30 UTC

Production build deployed and public JS checksum matched the build artifact.
All 20 unit tests and live Chromium smoke checks passed. All nine matchup
outcomes resolved correctly; all three deadlocks had no winner spotlight.
Mouse orbit/wheel, touch orbit/pinch, responsive God View and batch export passed.
Single VP9 WebM: **189,184 bytes**, **1080 × 1000**, approximately **4.0 seconds**;
loaded and decoded in-browser, downloaded, probed with ffprobe, and a winner
frame decoded with ffmpeg confirms the overhead beam is present in the video.
All nine batch clips were nonempty (175–261 KB). Console/page/HTTP errors: **0**.

Durable local proof directory:
`/home/rico/.openclaw/agents/carmack/workspace/artifacts/bottagon-god-view-20261020/`.
Rollback snapshot: `deploy-backups/20261020T222745Z-pre-god-view/`.

### Live aesthetic update — 2026-10-20 22:42 UTC

Removed the black face shell, wire cage and chin bib entirely. The silver helmet
now carries only a flush curved glowing eye strip. Shared team palette is cobalt
blue `#0044ff` versus blood red `#990000` across guards, visor/core/tip emission,
side lights, sword trails, winner spotlight/beam and matching UI controls.
Chest geometry contains an actual smooth recess; the core is embedded inside
its socket rather than laid over the jacket surface.

Verification: **22/22 unit tests**, production build, public bundle SHA-256 match,
and complete live Chromium suite passed. All nine outcomes, six cameras, desktop
mouse orbit/wheel, mobile touch orbit/pinch, single capture/decode/download and
nine unique batch WebM exports passed. Console/page/HTTP errors: **0**.
Single VP9 clip: **172,371 bytes**, **1080 × 1000**, **3.697 seconds**. Downloaded
clip probed with ffprobe and a winner frame decoded with ffmpeg and inspected.
Batch clips ranged from 100,684 to 223,613 bytes. Only build advisory is the
existing Vite >500 kB bundle warning.

Proof directory:
`/home/rico/.openclaw/agents/carmack/workspace/artifacts/bottagon-cobalt-blood-red-20261020/`.
Rollback: `deploy-backups/20261020T223919Z-pre-cobalt-blood-red/`.

## Live spark/settings refinement — 2026-10-20 23:27 UTC

Deployed to https://jediui.com/studio/. Public bundle `index-Bck9C50R.js`
SHA-256: `45a598ce13ab97aae4d57c2353cedc163e5d3f6e65b34c7f3dbe05b813444b81`.

- Winner-colored particles/streaks/flare/light; ties neutral gold. Particle count
  192 → 48; lifetime 820 → 460 ms; peak contact light 65 → 3; smaller flare,
  shorter streaks, tighter spread and 5 px maximum points.
- Exact manually orbited/zoomed camera is frozen, without preset reset. Camera
  buttons, gestures and guard selection lock for the full export. Snapshot guard
  is passed to every clip; original stance choices return afterward. Fresh
  controls discard stale drag inertia on unlock without changing the view.
- **30/30 unit tests**, production build and full live Chromium suite pass.
- All nine rules outcomes, six presets, desktop orbit/wheel and mobile touch/
  pinch pass. A single VP9 clip decoded/downloaded at **1080 × 1000**, **433,759
  bytes**. Its final guard frame was decoded with ffmpeg for visual inspection.
- **18 real batch clips**: God View + manual orbit/zoom + Quarte (357,311–481,958
  bytes), and Broadcast + manual orbit/zoom + Prime (402,220–600,015 bytes).
  Actual authored Paper poses matched the selected guard variants. Every clip retained
  camera/guard; camera interaction was blocked. No camera drift after unlock.
- All 18 side-by-side outcome holds measured **797.5–820.3 ms**, followed by
  smooth recovery and final ON_GUARD. Console/page/HTTP errors: **0**.
- Existing Vite >500 kB chunk advisory only; no build failure.

Evidence: `/home/rico/.openclaw/agents/carmack/workspace/artifacts/studio-spark-settings-20261020/`.
Rollback: `deploy-backups/20261020T231507Z-pre-spark-settings/`.
Early burst drawing-buffer captures can be reproduced with the same browser
module overrides using `node test/browser-spark-proof.mjs` (`PROOF_DIR` optional).

Additional live drawing-buffer proof passed for blue-win/red-win/tie bursts,
with zero console errors. Images: `sparks-0.png`, `sparks-1.png`, and
`sparks-tie.png` in the evidence directory above.

## Final combat-grounding audit — 2026-10-20 (CORTEX #502)

The master cut now authors **paired weapon lines**, not independent poses followed
by an approximate spark. `src/combat.js` is shared by playback and collision tests.

- Both procedural rigs have the same local right-handed shoulder/torso anatomy;
  the opponent is rotated, never reflected. Existing clean helmets, inset chest
  cores, minimal shoes, camera controls and no-ground/no-spotlight presentation remain.
- Weapon-arm two-bone IK preserves the 0.39/0.38 arm lengths. Its elbow plane
  minimizes wrist deviation; contact poses are tested below 75° of forearm/blade
  deviation, rather than allowing the previous reversed/hyperflexed guard wrist.
- Paper contact is on the defender's **23% forte** and incoming **70% foible**.
  Sixte, quarte and prime each have an explicit interception line. The incoming
  point stops short of the jacket; low parries route around the weapon arm.
- Committed hits raycast the actual recessed jacket/core surface. Scissors first
  draws a false Paper guard, disengages outside the arm, then renews its attack. The
  false defensive hand moves offline, so the hit does not pass through a forearm.
- Rock/Rock uses an actual crossed-blade contact; Paper/Paper a measured
  tip probe; Scissors/Scissors has hesitation with **no invented collision or sparks**.
- Snapshot interpolation resolves one group-local weapon aim independently of
  shoulder/elbow/wrist quaternion blending, avoiding thigh/forearm sword sweeps.
- Hip drive leads the point extension slightly; traveling feet lift modestly
  rather than both feet hopping. Soles remain level, with no ground penetration.
  Contact holds for 55 ms before elastic recoil; successful thrusts withdraw
  instead of continuing through the opponent's torso.
- Sparks use closest points across all eight actual blade segments, including
  flex, or the actual body-touch point. Contact diagnostics expose position,
  gap and blade fractions. Cobalt/blood-red winner tint and restrained gold clash
  tint are preserved. Outcome and guard recovery keep the existing smooth,
  camera-facing high victory salute / bowed low-blade defeat silhouettes.

### Verification

`npm test` includes all nine matchups in all three guard variants, 240 samples
per combat transition, both full body meshes, recoil, result transitions and guard
recovery (over 600,000 blade-segment samples). No unintended self/opponent body
intersection is allowed. The intended rounded 9 mm tip touching its target jacket
at the first recoil sample is contact, not an unintended body crossing.

The browser suite exercises all nine default outcomes plus all nine recordings
in both alternate guard variants, camera lock, desktop mouse/mobile touch controls,
result holds and exact on-guard restoration. Every batch clip is saved and must
**decode and seek**, not merely have nonzero bytes. Set `PROOF_DIR` to retain all
WebMs and the contact/outcome PNGs plus numerical diagnostics:

```bash
PLAYWRIGHT_MODULE=/path/to/playwright/index.mjs \
CHROMIUM_EXECUTABLE=/usr/bin/chromium \
STUDIO_URL=https://jediui.com/studio/ \
PROOF_DIR=/path/to/proof node test/browser-smoke.mjs
```

This remains deterministic procedural choreography, not a rigid-body simulator,
biomechanical certification or a claim that every arbitrary camera/browser can
never reveal a visual issue. Verification combines dense geometry sweeps, actual
rendered frames and real video decoding. Vite's existing >500 kB chunk advisory
is nonfatal; the production build succeeds without compiler errors.

To independently decode **every frame** of retained batch recordings and extract
its actual last frame (requires system `ffmpeg` and `ffprobe`):

```bash
node test/verify-webm.mjs /path/to/proof
```

## Optimized master export — CORTEX #504 (2026-10-21)

God View moves closer with a higher aim point; Broadcast uses a tighter
orthographic envelope. Conservative mesh-bound sweeps sample **128,100
view-frames** through all 27 choreographies, outcomes and recovery at five
aspects. Full swords, helmets and soles retain at least 2.45% edge clearance.

Regenerate the 18 default-guard God/Broadcast master clips:

```sh
PLAYWRIGHT_MODULE=/path/to/playwright/index.mjs \
STUDIO_URL=https://jediui.com/studio/ \
PROOF_DIR=/tmp/studio-504-master node test/browser-optimized.mjs
EXPECT_FPS=30 node test/verify-webm.mjs /tmp/studio-504-master
PLAYWRIGHT_MODULE=/path/to/playwright/index.mjs node test/browser-replays.mjs
```

The optimized suite checks HiDPI buffer limits, both actual nine-clip batches,
camera/guard locks, each logical outcome hold, final on-guard restoration, browser
decode/seek, and console/page/HTTP errors. `EXPECT_FPS=30` additionally verifies
**every decoded timestamp interval**, dimensions and size bounds, rather than
trusting WebM FPS metadata. Export may take several minutes on this VPS's
software renderer. Preserve `replays/` when copying the build to production.
