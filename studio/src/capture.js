// Bound actual GPU work, not just the encoded stream. Preserve the viewport
// aspect and camera/orbit, while avoiding HiDPI multi-megapixel WebGL buffers.
export const CAPTURE = Object.freeze({ maxWidth: 720, maxHeight: 660, fps: 30, bitrate: 400000, drainMs: 100 });
export const TIMING = Object.freeze({ outcomeHoldMs: 450, recoverySeconds: 0.425, guardHoldMs: 80 });
export function renderSize(width, height) {
  const scale = Math.min(1, CAPTURE.maxWidth / Math.max(1, width), CAPTURE.maxHeight / Math.max(1, height));
  // Even dimensions also work with downstream YUV encoders.
  return [width, height].map(size => Math.max(2, Math.floor(size * scale / 2) * 2));
}
