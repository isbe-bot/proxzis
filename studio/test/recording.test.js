import test from 'node:test';
import assert from 'node:assert/strict';
import { FrameRecording } from '../src/recording.js';

test('frame-stepped export encodes each frame with monotonic 30fps timestamps and bounded queue', async () => {
  const saved = { VideoEncoder: globalThis.VideoEncoder, VideoFrame: globalThis.VideoFrame };
  const frames = [], encoded = [];
  globalThis.VideoFrame = class {
    constructor(canvas, options) { Object.assign(this, options); this.canvas = canvas; frames.push(this); }
    close() { this.closed = true; }
  };
  globalThis.VideoEncoder = class {
    static async isConfigSupported(config) { return { supported: config.codec === 'vp8' }; }
    constructor() { this.state = 'unconfigured'; this.encodeQueueSize = 0; }
    configure(config) { this.config = config; this.state = 'configured'; }
    encode(frame, options) { encoded.push({ timestamp:frame.timestamp, ...options }); }
    async flush() { this.flushed = true; this.encodeQueueSize = 0; }
    close() { this.state = 'closed'; }
  };
  try {
    assert.equal(await FrameRecording.supported(720,660),true);
    const recording = new FrameRecording(720,660);
    let settled = false;
    const hold = recording.hold(450).then(()=> { settled = true; });
    for(let i=0;i<13;i++) await recording.capture({});
    assert.equal(settled,false);
    recording.encoder.encodeQueueSize = 3;
    await recording.capture({}); await hold;
    assert.equal(recording.encoder.flushed,true);
    for(let i=14;i<90;i++) await recording.capture({});
    assert.equal(recording.time,3000);
    assert.equal(frames.length,90);
    frames.forEach(frame=>assert.equal(frame.closed,true));
    encoded.forEach((frame,i)=>assert.equal(frame.timestamp,Math.round(i*1e6/30)));
    assert.equal(encoded[60].keyFrame,true);
    recording.error = new Error('encode failed');
    await assert.rejects(recording.capture({}),/encode failed/);
    await assert.rejects(recording.hold(100),/encode failed/);
    recording.close(); assert.equal(recording.encoder.state,'closed');
  } finally { Object.assign(globalThis,saved); }
});
