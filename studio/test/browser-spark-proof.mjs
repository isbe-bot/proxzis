// Capture the actual drawing buffer on the first visible burst frame, rather
// than a compositor screenshot that may arrive after a subtle 460 ms effect.
import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const output = process.env.PROOF_DIR || '/tmp/studio-spark-proof';
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath: process.env.CHROMIUM_EXECUTABLE || undefined,
  args: ['--no-sandbox', '--use-angle=swiftshader', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1200, height: 850 } });
const errors = [];
page.on('pageerror', error => errors.push(error.message));
page.on('console', message => { if (message.type() === 'error') errors.push(message.text()); });
try {
  await page.goto(process.env.STUDIO_URL || 'https://jediui.com/studio/', { waitUntil: 'networkidle' });
  const results = [];
  for (const [a, b, winner, tint] of [
    ['ROCK', 'SCISSORS', 0, 0x0044ff],
    ['ROCK', 'PAPER', 1, 0x990000],
    ['ROCK', 'ROCK', null, 0xffdf9e],
  ]) {
    await page.click(`#botA-btns [data-stance="${a}"]`);
    await page.click(`#botB-btns [data-stance="${b}"]`);
    await page.evaluate(() => {
      window.sparkProof = null;
      const sample = () => {
        const d = studioDiagnostics();
        if (d.contact.visible && d.contact.life > 0) {
          window.sparkProof = { contact: d.contact, image: document.querySelector('#canvas3d').toDataURL('image/png') };
        } else requestAnimationFrame(sample);
      };
      requestAnimationFrame(sample);
      document.querySelector('#btn-play').click();
    });
    await page.waitForFunction(() => window.sparkProof, {}, { timeout: 20000 });
    const proof = await page.evaluate(() => window.sparkProof);
    assert.equal(proof.contact.winner, winner);
    assert.equal(proof.contact.tint, tint);
    const file = `${output}/sparks-${winner ?? 'tie'}.png`;
    await writeFile(file, Buffer.from(proof.image.split(',')[1], 'base64'));
    results.push({ a, b, winner, ...proof.contact, file });
    await page.waitForFunction(() => !studioDiagnostics().busy, {}, { timeout: 20000 });
  }
  assert.deepEqual(errors, []);
  console.log(JSON.stringify({ results, errors }, null, 2));
} finally { await browser.close(); }
