// Live guard alignment and all matchup/outcome smoke proof for CORTEX #503.
import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const proof = process.env.PROOF_DIR || '/tmp/studio-503-proof';
await mkdir(proof, { recursive: true });
const browser = await chromium.launch({headless:true, executablePath:process.env.CHROMIUM_EXECUTABLE || '/usr/bin/chromium', args:['--no-sandbox','--enable-webgl','--use-angle=swiftshader','--enable-unsafe-swiftshader']});
const page = await browser.newPage({viewport:{width:1440,height:1000}});
const errors = [], cases = [];
page.on('pageerror', e => errors.push(e.message));
page.on('console', m => { if(m.type()==='error') errors.push(m.text()); });
page.on('response', r => { if(r.status()>=400) errors.push(`${r.status()} ${r.url()}`); });
function guard(d) {
  assert.equal(d.idle,true);
  d.rigs.forEach((rig,i) => {
    assert.equal(rig.pose,'ON_GUARD');
    assert.ok(Math.abs(rig.headFacing[0]-(i?-1:1))<1e-8);
    assert.ok(Math.hypot(rig.headFacing[1],rig.headFacing[2])<1e-8);
    rig.torsoYaw.forEach(yaw => assert.equal(yaw,0));
    rig.feet.forEach(foot => assert.ok(Math.abs(foot[1]-0.075)<1e-6));
  });
}
try {
  await page.goto(process.env.STUDIO_URL || 'https://jediui.com/studio/',{waitUntil:'networkidle'});
  await page.waitForFunction(()=>window.studioDiagnostics?.().drawCalls>0);
  guard(await page.evaluate(()=>studioDiagnostics()));
  await page.screenshot({path:`${proof}/guard-god.png`});
  await page.click('[data-camera="broadcast"]');
  await page.screenshot({path:`${proof}/guard-broadcast.png`});
  await page.click('[data-camera="god"]');
  for(const guardLine of ['PAPER','PAPER_QUARTE','PAPER_PRIME']) {
    await page.selectOption('#guard-line',guardLine);
    for(const a of ['ROCK','SCISSORS','PAPER']) for(const b of ['ROCK','SCISSORS','PAPER']) {
      await page.click(`#botA-btns [data-stance="${a}"]`);
      await page.click(`#botB-btns [data-stance="${b}"]`);
      await page.evaluate(()=>{
        window.outcomeProof=null;
        document.querySelector('#btn-play').click();
        function sample() {
          const d=studioDiagnostics();
          if(d.result.settled&&d.engagementPhase==='OUTCOME') window.outcomeProof=d;
          if(d.busy) requestAnimationFrame(sample);
        }
        requestAnimationFrame(sample);
      });
      await page.waitForFunction(()=>!studioDiagnostics().busy,null,{timeout:20000});
      const d=await page.evaluate(()=>({guard:studioDiagnostics(),outcome:window.outcomeProof}));
      guard(d.guard);
      assert.ok(d.outcome,'Must settle side-by-side outcome');
      const headings=d.outcome.rigs.map(rig=>rig.facing);
      assert.ok(Math.hypot(...headings[0].map((v,i)=>v-headings[1][i]))<1e-8);
      d.outcome.rigs.forEach(rig=>rig.feet.forEach(foot=>assert.ok(Math.abs(foot[1]-0.075)<1e-6)));
      cases.push({guardLine,a,b,winner:d.outcome.result.winner,guard:d.guard.rigs.map(r=>({headFacing:r.headFacing,torsoYaw:r.torsoYaw}))});
      console.log(`PASS ${guardLine} ${a}/${b}`);
    }
  }
  assert.deepEqual(errors,[]);
  await writeFile(`${proof}/live-results.json`,JSON.stringify({url:page.url(),cases,errors},null,2));
  console.log(JSON.stringify({passed:cases.length,errors,proof}));
} finally { await browser.close(); }
