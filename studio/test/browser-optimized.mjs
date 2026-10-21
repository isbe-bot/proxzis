// CORTEX #504: actual frame-stepped master batches, lock/timing/decode proof.
import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const proofDir = process.env.PROOF_DIR || '/tmp/studio-504/master';
await mkdir(proofDir,{recursive:true});
const browser = await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_EXECUTABLE || '/usr/bin/chromium',args:['--no-sandbox','--enable-webgl','--use-angle=swiftshader','--enable-unsafe-swiftshader']});
const page = await browser.newPage({viewport:{width:1440,height:990},deviceScaleFactor:2});
const errors=[],reports=[];
page.on('pageerror',e=>errors.push(e.message));
page.on('console',m=>{if(m.type()==='error')errors.push(m.text());else if(m.text().startsWith('CLIP '))console.log(m.text());});
page.on('response',r=>{if(r.status()>=400)errors.push(`${r.status()} ${r.url()}`);});
try {
  await page.goto(process.env.STUDIO_URL || 'https://jediui.com/studio/',{waitUntil:'networkidle'});
  await page.waitForFunction(()=>window.studioDiagnostics?.().drawCalls>0);
  for(const preset of ['god','broadcast']) {
    await page.click(`[data-camera="${preset}"]`);
    await page.selectOption('#guard-line','PAPER');
    const before=await page.evaluate(()=>studioDiagnostics());
    assert.deepEqual([before.capture.width,before.capture.height],[720,660]);
    await page.locator('#canvas3d').screenshot({path:`${proofDir}/${preset}-guard.png`});
    const count=await page.locator('#clip-downloads a').count();
    await page.evaluate(()=>{
      window.optimizedProof={baseline:studioDiagnostics(),violations:[],holds:[],outcomes:{},start:null};
      const observer=new MutationObserver(()=>console.log('CLIP '+document.querySelector('#clip-downloads span')?.textContent));
      observer.observe(document.querySelector('#clip-downloads'),{childList:true});window.clipObserver=observer;
      document.querySelector('#btn-batch-export').click();
      const sample=()=>{
        const d=studioDiagnostics(),p=window.optimizedProof,b=p.baseline;
        if(d.batch) {
          if(d.camera!==b.camera||d.guardLine!==b.guardLine||d.orbit.zoom!==b.orbit.zoom||!d.orbit.locked||d.orbit.enabled||Math.hypot(...d.cameraPosition.map((v,i)=>v-b.cameraPosition[i]))>1e-6)p.violations.push(d.cameraPosition);
          if(d.engagementPhase==='OUTCOME'&&d.result.settled&&p.start===null) {
            p.start=d.capture.time;
            p.outcomes[d.stances.join('/')]=d.result.winner;
            if(!p.image)p.image=document.querySelector('#canvas3d').toDataURL('image/png');
          }
          if(d.engagementPhase==='RECOVER'&&p.start!==null){p.holds.push(d.capture.time-p.start);p.start=null;}
        }
        if(d.busy)requestAnimationFrame(sample);
      };requestAnimationFrame(sample);
    });
    assert.equal(await page.locator('[data-camera]:disabled').count(),6);
    await page.evaluate(()=>document.querySelector('[data-camera="close"]').dispatchEvent(new MouseEvent('click')));
    await page.waitForFunction(n=>!studioDiagnostics().busy&&document.querySelectorAll('#clip-downloads a').length===n+9,count,{timeout:1800000});
    const proof=await page.evaluate(()=>{window.clipObserver.disconnect();return {...window.optimizedProof,after:studioDiagnostics()};});
    assert.deepEqual(proof.violations,[]);
    assert.equal(Object.keys(proof.outcomes).length,9);
    assert.equal(proof.holds.length,9);
    proof.holds.forEach(ms=>assert.ok(ms>=440&&ms<=520,`logical hold ${ms}`));
    assert.deepEqual(proof.after.stances,proof.baseline.stances);
    assert.ok(Math.hypot(...proof.after.cameraPosition.map((v,i)=>v-proof.baseline.cameraPosition[i]))<1e-8,'No drift after camera unlock');
    proof.after.rigs.forEach(r=>assert.equal(r.pose,'ON_GUARD'));
    await writeFile(`${proofDir}/${preset}-outcome.png`,Buffer.from(proof.image.split(',')[1],'base64'));delete proof.image;
    const clips=await page.evaluate(async()=>Promise.all([...document.querySelectorAll('#clip-downloads a')].slice(0,9).map(async a=>{
      const blob=await(await fetch(a.href)).blob();const data=await new Promise(resolve=>{const r=new FileReader();r.onload=()=>resolve(r.result.split(',')[1]);r.readAsDataURL(blob)});
      const video=document.createElement('video');video.src=a.href;video.muted=true;
      await new Promise((resolve,reject)=>{video.onloadeddata=resolve;video.onerror=()=>reject(new Error('decode failed'))});
      const duration=video.duration;video.currentTime=duration-0.1;
      await new Promise((resolve,reject)=>{video.onseeked=resolve;video.onerror=()=>reject(new Error('seek failed'))});
      const decoded={width:video.videoWidth,height:video.videoHeight,duration,ready:video.readyState};video.removeAttribute('src');video.load();
      return {name:a.download,data,decoded,bytes:blob.size};
    })));
    for(const clip of clips){assert.deepEqual([clip.decoded.width,clip.decoded.height],[720,660]);assert.ok(clip.decoded.ready>=2);await writeFile(`${proofDir}/${preset}-${clip.name}`,Buffer.from(clip.data,'base64'));delete clip.data;}
    reports.push({preset,clips,proof});
    await writeFile(`${proofDir}/browser-report.json`,JSON.stringify({reports,errors},null,2));
    console.log(`PASS ${preset}: 9 decoded clips; holds ${proof.holds.join(',')}`);
  }
  assert.deepEqual(errors,[]);
  console.log(JSON.stringify({passed:18,errors,proofDir}));
} finally {await browser.close();}
