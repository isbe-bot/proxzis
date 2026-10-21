import test from 'node:test';
import assert from 'node:assert/strict';
import { CAPTURE, TIMING, renderSize } from '../src/capture.js';
import { RoundResult } from '../src/result.js';
import { StudioRig } from '../src/rig.js';

test('GPU drawing buffer is capped, even and aspect preserving on HiDPI/large viewports', () => {
  for (const [w, h] of [[1080,1000],[2160,2000],[390,490],[1920,480],[300,2000]]) {
    const [width,height] = renderSize(w,h);
    assert.ok(width <= CAPTURE.maxWidth && height <= CAPTURE.maxHeight);
    assert.equal(width % 2, 0); assert.equal(height % 2, 0);
    assert.ok(Math.abs(width/height-w/h) < 0.025);
  }
  assert.deepEqual(renderSize(1080,990), [720,660]);
  assert.equal(CAPTURE.fps, 30);
  assert.ok(CAPTURE.bitrate <= 1000000 && CAPTURE.drainMs <= 150);
});

test('recovery resolves in 425ms with a short readable result and guard tail', async () => {
  const rigs = [new StudioRig(),new StudioRig(undefined,false)];
  const result = new RoundResult(null, rigs);
  result.show(0); result.update(1);
  const recovered = result.returnToGuard();
  result.update(0.4); assert.equal(result.settled, false);
  result.update(0.025); await recovered;
  assert.equal(result.phase,'ready');
  assert.equal(TIMING.outcomeHoldMs,450);
  assert.equal(TIMING.guardHoldMs,80);
});
