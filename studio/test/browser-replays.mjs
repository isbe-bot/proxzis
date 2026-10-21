import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const proofDir = process.env.PROOF_DIR || '/tmp/studio-504/replays';
await mkdir(proofDir,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_EXECUTABLE || '/usr/bin/chromium',args:['--no-sandbox']});
const page=await browser.newPage({viewport:{width:1440,height:1000}});
const errors=[],runs=[];
page.on('pageerror',e=>errors.push(e.message));
page.on('console',m=>{if(m.type()==='error')errors.push(m.text());});
page.on('response',r=>{if(r.status()>=400)errors.push(`${r.status()} ${r.url()}`);});
try {
  await page.goto(process.env.REPLAY_URL || 'https://jediui.com/studio/replays/',{waitUntil:'networkidle'});
  assert.equal(await page.locator('.trans-btn.active').getAttribute('data-transition'),'crossfade');
  for(const camera of ['god','broadcast']) {
    await page.locator(`.camera-tab[data-camera="${camera}"]`).click();
    await page.waitForFunction(()=>!isTransitioning&&document.querySelector('video.active-layer').videoWidth===720);
    await page.locator('.round-card').first().click();
    await page.waitForFunction(()=>!isTransitioning&&document.querySelector('video.active-layer').readyState>=2);
    await page.locator('[data-phase="recovery"]').click();
    const jump=await page.evaluate(()=>{const v=document.querySelector('video.active-layer');return {time:v.currentTime,duration:v.duration};});
    assert.ok(jump.time<jump.duration&&jump.time>jump.duration-0.8,'Recovery jump stays inside the shorter clip');
    await page.locator('.round-card').first().click();
    await page.waitForFunction(()=>!isTransitioning&&document.querySelector('video.active-layer').currentTime<1);
    await page.evaluate(()=>{
      window.replayProof={rounds:[],dimensions:[],emptyFrames:0};
      const tick=()=>{
        const v=document.querySelector('video.active-layer'),r=document.querySelector('.round-card.active')?.dataset.index;
        if(r&&!replayProof.rounds.includes(r))replayProof.rounds.push(r);
        if(v.readyState>=2&&!replayProof.dimensions.some(d=>d[0]===v.videoWidth&&d[1]===v.videoHeight))replayProof.dimensions.push([v.videoWidth,v.videoHeight]);
        if(!v.readyState)replayProof.emptyFrames++;
        if(document.querySelector('#round-indicator').textContent!=='MATCH COMPLETE')requestAnimationFrame(tick);
      };requestAnimationFrame(tick);
    });
    await page.locator('#btn-play-all').click();
    await page.waitForFunction(()=>document.querySelector('#round-indicator').textContent==='MATCH COMPLETE',null,{timeout:90000});
    const proof=await page.evaluate(()=>window.replayProof);
    assert.equal(proof.rounds.length,7);assert.deepEqual(proof.dimensions,[[720,660]]);
    await page.screenshot({path:`${proofDir}/${camera}-complete.png`});
    runs.push({camera,jump,...proof});
    console.log(`PASS ${camera}: seven rounds, dissolve, 720x660, recovery seek`);
  }
  assert.deepEqual(errors,[]);
  await writeFile(`${proofDir}/replay-report.json`,JSON.stringify({runs,errors},null,2));
} finally {await browser.close();}
