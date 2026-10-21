import { Muxer, ArrayBufferTarget } from 'webm-muxer';
import { CAPTURE } from './capture.js';

// Animation time advances one frame per rendered/encoded image, never by wall
// time. Software GPUs may export slowly, but cannot drop choreography frames.
export class FrameRecording {
  static async supported(width, height) {
    if (!globalThis.VideoEncoder || !globalThis.VideoFrame) return false;
    try { return (await VideoEncoder.isConfigSupported(FrameRecording.config(width, height))).supported; }
    catch { return false; }
  }
  static config(width, height) {
    return { codec: 'vp8', width, height, bitrate: CAPTURE.bitrate, framerate: CAPTURE.fps, latencyMode: 'quality' };
  }
  constructor(width, height) {
    this.frame = 0;
    this.time = 0;
    this.waiters = [];
    this.target = new ArrayBufferTarget();
    this.muxer = new Muxer({ target: this.target, video: { codec: 'V_VP8', width, height, frameRate: CAPTURE.fps }, firstTimestampBehavior: 'offset' });
    this.encoder = new VideoEncoder({
      output: (chunk, metadata) => this.muxer.addVideoChunk(chunk, metadata),
      error: error => { this.error = error; },
    });
    this.encoder.configure(FrameRecording.config(width, height));
  }
  hold(ms) {
    if (this.error) return Promise.reject(this.error);
    return new Promise((resolve, reject) => this.waiters.push({ end: this.time + ms, resolve, reject }));
  }
  async capture(canvas) {
    if (this.error) throw this.error;
    // Bound encoder memory on devices with slow software VP8 encoders.
    if (this.encoder.encodeQueueSize > 2) await this.encoder.flush();
    const frame = new VideoFrame(canvas, { timestamp: Math.round(this.frame * 1e6 / CAPTURE.fps), duration: Math.round(1e6 / CAPTURE.fps) });
    try { this.encoder.encode(frame, { keyFrame: this.frame % (CAPTURE.fps * 2) === 0 }); }
    finally { frame.close(); }
    this.time = ++this.frame * 1000 / CAPTURE.fps;
    const due = this.waiters.filter(waiter => waiter.end <= this.time);
    this.waiters = this.waiters.filter(waiter => waiter.end > this.time);
    due.forEach(waiter => waiter.resolve());
  }
  async finish() {
    await this.encoder.flush();
    if (this.error) throw this.error;
    this.muxer.finalize();
    return new Blob([this.target.buffer], { type: 'video/webm' });
  }
  close() {
    if (this.encoder.state !== 'closed') this.encoder.close();
    this.waiters.forEach(waiter => waiter.resolve());
    this.waiters = [];
  }
}
