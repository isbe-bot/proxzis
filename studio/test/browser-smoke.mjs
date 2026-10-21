// PLAYWRIGHT_MODULE may point to an existing Playwright install; no added runtime dependency.
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
const proofDir = process.env.PROOF_DIR || '/tmp/studio-master-recordings';
await mkdir(proofDir, { recursive: true });
const browser = await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_EXECUTABLE || undefined,args:['--no-sandbox','--enable-webgl','--use-angle=swiftshader','--enable-unsafe-swiftshader']});
const page = await browser.newPage({viewport:{width:1440,height:1000},hasTouch:true});
const errors=[];
page.on('pageerror',e=>errors.push(e.message));
page.on('console',m=>{if(m.type()==='error')errors.push(m.text());});
page.on('response',r=>{if(r.status()>=400)errors.push(`${r.status()} ${r.url()}`);});
const wins=(a,b)=>(a==='ROCK'&&b==='SCISSORS')||(a==='SCISSORS'&&b==='PAPER')||(a==='PAPER'&&b==='ROCK');
try {
 await page.goto(process.env.STUDIO_URL || 'https://jediui.com/studio/',{waitUntil:'networkidle'});
 await page.waitForFunction(()=>window.studioDiagnostics?.().drawCalls>0);
 const initialStyle=await page.evaluate(()=>studioDiagnostics());
 assert.equal(initialStyle.camera,'god');
 initialStyle.rigs.forEach((rig,i)=>{
  assert.equal(rig.color,[0x0044ff,0x990000][i]);
  assert.equal(rig.visor,false);
  assert.equal(rig.mask,false);
  assert.ok(rig.coreDepth<0.1&&rig.coreFront<=0.125);
 });
 await page.screenshot({path:'/tmp/studio-god-guard.png'});
 assert.equal(await page.locator('#engagement-cue').count(),0);
 assert.equal(await page.locator('#round-result').count(),0);
 for(const camera of ['god','broadcast','dynamic','low','close','stadium']){
  await page.click(`[data-camera="${camera}"]`);
  const view=await page.evaluate(()=>studioDiagnostics());
  assert.equal(view.camera,camera);
  assert.ok(view.cameraPosition[0]<0);
 }
 // Real mouse drag/wheel and clean preset resets for both camera projections.
 const distance=(a,b)=>Math.hypot(...a.map((v,i)=>v-b[i]));
 for(const preset of ['god','broadcast']) {
  await page.click(`[data-camera="${preset}"]`);
  const initial=await page.evaluate(()=>studioDiagnostics());
  const box=await page.locator('#canvas3d').boundingBox();
  const x=box.x+box.width/2,y=box.y+box.height/2;
  await page.mouse.move(x,y);await page.mouse.down();
  await page.mouse.move(x+110,y+50,{steps:12});await page.mouse.up();
  await page.waitForTimeout(600);
  const rotated=await page.evaluate(()=>studioDiagnostics());
  assert.ok(distance(initial.cameraPosition,rotated.cameraPosition)>0.2,`${preset} mouse orbit`);
  await page.mouse.wheel(0,-350);await page.waitForTimeout(500);
  const zoomed=await page.evaluate(()=>studioDiagnostics());
  assert.ok(preset==='broadcast'?zoomed.orbit.zoom>rotated.orbit.zoom:
   distance(zoomed.cameraPosition,zoomed.orbit.target)<distance(rotated.cameraPosition,rotated.orbit.target),`${preset} wheel zoom`);
  await page.click(`[data-camera="${preset}"]`);
  const reset=await page.evaluate(()=>studioDiagnostics());
  assert.ok(distance(initial.cameraPosition,reset.cameraPosition)<1e-8);
  assert.equal(reset.orbit.zoom,1);
  assert.ok(distance(initial.orbit.target,reset.orbit.target)<1e-8);
 }
 await page.screenshot({path:'/tmp/studio-broadcast-guard.png'});
 await page.click('[data-camera="god"]');
 const cases=[];
 for(const a of ['ROCK','SCISSORS','PAPER'])for(const b of ['ROCK','SCISSORS','PAPER']){
  await page.click(`#botA-btns [data-stance="${a}"]`);await page.click(`#botB-btns [data-stance="${b}"]`);
  assert.equal((await page.evaluate(()=>studioDiagnostics())).viewportTextSprites,0);
  // A Playwright click can wait through brief phases on a contended software GPU.
  // Dispatch in-page so observation begins immediately after the async handler starts.
  await page.evaluate(()=>{
    window.frameProof={};
    const sample=()=>{
      const d=studioDiagnostics();
      if(d.engagementPhase==='CLASH'&&!window.frameProof.contact)window.frameProof.contact={diagnostics:d,image:document.querySelector('#canvas3d').toDataURL('image/png')};
      if(d.engagementPhase==='OUTCOME'&&d.result.settled&&!window.frameProof.outcome)window.frameProof.outcome={diagnostics:d,image:document.querySelector('#canvas3d').toDataURL('image/png')};
      if(d.busy)requestAnimationFrame(sample);
    };
    document.querySelector('#btn-play').click();
    requestAnimationFrame(sample);
  });
  await page.waitForFunction(()=>studioDiagnostics().engagementPhase==='SALUTE');
  assert.equal((await page.evaluate(()=>studioDiagnostics())).viewportTextSprites,0);
  if(a!=='SCISSORS'||b!=='SCISSORS'){
  await page.waitForFunction(()=>studioDiagnostics().contact.visible&&studioDiagnostics().contact.life>0.25);
  const contact=await page.evaluate(()=>studioDiagnostics().contact);
  assert.equal(contact.particles,48);
  assert.equal(contact.tint,a===b?0xffdf9e:wins(a,b)?0x0044ff:0x990000);
  assert.ok(contact.light<=3);
  assert.equal(contact.streaks,true);
  assert.ok(contact.light>0);
  if(a==='ROCK'&&b==='ROCK')await page.screenshot({path:'/tmp/studio-contact-burst.png'});
  const measured=await page.evaluate(()=>studioDiagnostics().contact.measured);
  assert.ok(measured.distance<1e-5,`${a}/${b}: actual contact gap ${measured.distance}`);
  }
  await page.waitForFunction(()=>studioDiagnostics().result.active);
  let d=await page.evaluate(()=>studioDiagnostics());
  const winner=a===b?null:wins(a,b)?0:1;
  assert.equal(d.result.winner,winner);
  assert.equal(d.viewportTextSprites,0);
  assert.equal(d.result.spotlight.visible,false);
  assert.equal(d.result.beamVisible,false);
  assert.equal(d.result.spotlight.intensity,0);
  d.rigs.forEach((rig,i)=>{
   assert.equal(rig.rightHanded,true);
   assert.equal(rig.orb.state,winner===null?'neutral':winner===i?'win':'lose');
   assert.equal(rig.orb.intensity,winner===null?0.6:winner===i?5:0);
   if(winner!==null&&winner!==i)assert.equal(rig.orb.emissive,0);
   rig.feet.forEach(foot=>assert.ok(Math.abs(foot[1]-0.075)<0.025));
  });
  await page.waitForFunction(()=>studioDiagnostics().engagementPhase==='OUTCOME'&&studioDiagnostics().result.settled,{},{timeout:15000});
  d=await page.evaluate(()=>studioDiagnostics());
  d.rigs.forEach((rig,i)=>{
   assert.equal(rig.pose,winner===null?'ATTENTION':winner===i?'VICTORY':'DEFEAT');
   assert.ok(rig.facing[2]>0.99);
   assert.ok(Math.abs(rig.position[0]-(i?0.8:-0.8))<0.02);
   assert.ok(winner!==null&&i!==winner?rig.bladeTip[1]<rig.bladeBase[1]-0.75:rig.bladeTip[1]>rig.bladeBase[1]+0.8);
   if(winner!==null&&i!==winner)assert.ok(rig.headPitch < -0.3);
   if(winner!==null&&i===winner)assert.ok(rig.bladeBase[1]>2);
   assert.ok(distance(rig.footDirections[0],rig.footDirections[1])<1e-6);
  });
  assert.ok(distance(d.rigs[0].facing,d.rigs[1].facing)<1e-6);
  assert.equal(d.sceneryMeshes,0);
  if((a==='ROCK'&&b==='PAPER')||(a==='ROCK'&&b==='SCISSORS')||a===b&&a==='ROCK'){
    await page.screenshot({path:`/tmp/studio-result-${winner??'tie'}.png`});
  }
  // Slow software-GPU screenshots may finish after the 425 ms recovery.
  await page.waitForFunction(()=>studioDiagnostics().engagementPhase==='RECOVER'||!studioDiagnostics().busy);
  await page.waitForFunction(()=>!studioDiagnostics().busy,{},{timeout:15000});
  const recovered=await page.evaluate(()=>studioDiagnostics());
  assert.equal(recovered.result.active,false);
  assert.equal(recovered.result.phase,'ready');
  assert.equal(recovered.idle,true);
  recovered.rigs.forEach((rig,i)=>{
   assert.equal(rig.pose,'ON_GUARD');
   assert.equal(rig.orb.state,'neutral');
   assert.ok(Math.abs(rig.position[0]-(i?1.1:-1.1))<1e-8);
   assert.ok(Math.abs(rig.facing[0]-(i?-1:1))<1e-8);
   assert.ok(rig.bladeTip[1]>rig.bladeBase[1]);
  });
  if(a==='ROCK'&&b==='SCISSORS')await page.screenshot({path:'/tmp/studio-returned-guard.png'});
  const frames=await page.evaluate(()=>window.frameProof);
  for(const [phase,frame] of Object.entries(frames)) {
    await writeFile(`${proofDir}/${a.toLowerCase()}-${b.toLowerCase()}-${phase}.png`,Buffer.from(frame.image.split(',')[1],'base64'));
    await writeFile(`${proofDir}/${a.toLowerCase()}-${b.toLowerCase()}-${phase}.json`,JSON.stringify(frame.diagnostics,null,2));
  }
  assert.ok(frames.contact&&frames.outcome,`${a}/${b}: captured contact and outcome`);
  cases.push({a,b,winner,recovered:true});
 }
 await page.click('#btn-idle');
 assert.equal((await page.evaluate(()=>studioDiagnostics())).result.active,false);
 // Capture an actual clip, not just a DOM screenshot.
 await page.click('#botA-btns [data-stance="ROCK"]');await page.click('#botB-btns [data-stance="SCISSORS"]');
 await page.click('#btn-record-one');
 await page.waitForFunction(()=>document.querySelector('#clip-downloads a'),{},{timeout:180000});
 const clip=await page.evaluate(async()=>{const a=document.querySelector('#clip-downloads a');const b=await(await fetch(a.href)).blob();return {name:a.download,bytes:b.size,type:b.type};});
 assert.ok(clip.bytes>10000);
 assert.match(clip.type,/video\/webm/);
 const decoded=await page.evaluate(async()=>{
  const a=document.querySelector('#clip-downloads a');
  const video=document.createElement('video');video.src=a.href;video.muted=true;
  await new Promise((resolve,reject)=>{video.onloadeddata=resolve;video.onerror=()=>reject(new Error('WebM decode failed'));});
  video.currentTime=3.5;
  await new Promise((resolve,reject)=>{video.onseeked=resolve;video.onerror=()=>reject(new Error('WebM seek failed'));});
  return {width:video.videoWidth,height:video.videoHeight,ready:video.readyState};
 });
 assert.ok(decoded.width>0&&decoded.height>0&&decoded.ready>=2);
 const downloadPromise=page.waitForEvent('download');
 await page.locator('#clip-downloads a').first().click();
 const download=await downloadPromise;
 await download.saveAs('/tmp/studio-god-view.webm');
 // Two actual nine-clip exports preserve both projection types, manual orbit/zoom,
 // non-default parries and original stance selection throughout every clip.
 const batchRuns=[];
 for (const [preset,guardLine] of [['god','PAPER_QUARTE'],['broadcast','PAPER_PRIME']]) {
  await page.waitForFunction(()=>!studioDiagnostics().busy);
  await page.click(`[data-camera="${preset}"]`);
  await page.selectOption('#guard-line',guardLine);
  const box=await page.locator('#canvas3d').boundingBox();
  const x=box.x+box.width/2,y=box.y+box.height/2;
  await page.mouse.move(x,y);await page.mouse.down();
  await page.mouse.move(x+65,y+20,{steps:8});await page.mouse.up();
  await page.mouse.wheel(0,-120);await page.waitForTimeout(1200);
  await page.evaluate(()=>{
   const d=studioDiagnostics();
   window.batchProof={before:d,violations:[],contacts:{},clips:{},guardPoses:[],holds:[],outcomeStart:null};
   // Baseline at the actual click, not before Playwright's actionability wait.
   document.querySelector('#btn-batch-export').addEventListener('click',()=>{window.batchProof.before=studioDiagnostics();},{capture:true,once:true});
   window.batchProofTimer=setInterval(()=>{
    const d=studioDiagnostics(),p=window.batchProof;
    if(!d.batch)return;
    const changed=d.camera!==p.before.camera||d.guardLine!==p.before.guardLine||
     Math.hypot(...d.cameraPosition.map((v,i)=>v-p.before.cameraPosition[i]))>0.0001||
     d.orbit.zoom!==p.before.orbit.zoom||d.orbit.enabled||!d.orbit.locked;
    if(changed&&p.violations.length<3)p.violations.push({position:d.cameraPosition,baseline:p.before.cameraPosition,camera:d.camera,guardLine:d.guardLine,zoom:d.orbit.zoom,locked:d.orbit.locked,enabled:d.orbit.enabled});
    p.clips[d.stances.join('/')]=d.guardLine;
    if(d.engagementPhase==='CLASH'&&d.contact.measured)p.contacts[d.stances.join('/')]=d.contact.measured;
    d.rigs.forEach(rig=>{if(rig.pose.startsWith('PAPER')&&!p.guardPoses.includes(rig.pose))p.guardPoses.push(rig.pose);});
    if(d.engagementPhase==='OUTCOME'&&d.result.settled&&p.outcomeStart===null)p.outcomeStart=d.capture.time;
    if(d.engagementPhase==='RECOVER'&&p.outcomeStart!==null){p.holds.push(d.capture.time-p.outcomeStart);p.outcomeStart=null;}
   },20);
  });
  const count=await page.locator('#clip-downloads a').count();
  await page.click('#btn-batch-export');
  assert.equal(await page.locator('[data-camera]:disabled').count(),6);
  assert.equal(await page.locator('#guard-line').isDisabled(),true);
  // Even synthetic preset events and real orbit/zoom gestures cannot alter capture.
  await page.evaluate(()=>document.querySelector('[data-camera="close"]').dispatchEvent(new MouseEvent('click')));
  await page.mouse.move(x,y);await page.mouse.down();await page.mouse.move(x+90,y+30,{steps:4});await page.mouse.up();
  await page.mouse.wheel(0,300);
  await page.waitForFunction(count=>!studioDiagnostics().busy&&document.querySelectorAll('#clip-downloads a').length===count+9,count,{timeout:1800000});
  const proof=await page.evaluate(()=>{clearInterval(window.batchProofTimer);return {...window.batchProof,after:studioDiagnostics()};});
  assert.deepEqual(proof.violations,[],`${preset}: frozen camera/guard line throughout capture`);
  assert.equal(Object.keys(proof.clips).length,9);
  assert.equal(Object.keys(proof.contacts).length,9);
  Object.values(proof.contacts).forEach(contact=>assert.ok(contact.kind==='read'||contact.distance<1e-5));
  assert.deepEqual(proof.guardPoses,[guardLine],'Actual authored guard pose must match the selected variant');
  assert.ok(Object.values(proof.clips).every(p=>p===guardLine));
  assert.equal(proof.holds.length,9);
  proof.holds.forEach(ms=>assert.ok(ms>=420&&ms<550,`Outcome hold ${ms}ms`));
  assert.deepEqual(proof.after.stances,proof.before.stances);
  assert.equal(proof.after.guardLine,guardLine);
  assert.equal(proof.after.orbit.enabled,true);
  assert.ok(distance(proof.after.cameraPosition,proof.before.cameraPosition)<0.0001,'No stale inertia resumes after export');
  proof.after.rigs.forEach(rig=>assert.equal(rig.pose,'ON_GUARD'));
  const clips=await page.evaluate(async()=>Promise.all([...document.querySelectorAll('#clip-downloads a')].slice(0,9).map(async a=>({name:a.download,bytes:(await(await fetch(a.href)).blob()).size}))));
  assert.equal(new Set(clips.map(c=>c.name)).size,9);
  clips.forEach(clip=>assert.ok(clip.bytes>10000));
  const recordings = await page.evaluate(async () => Promise.all([...document.querySelectorAll('#clip-downloads a')].slice(0,9).map(async a => {
    const blob = await (await fetch(a.href)).blob();
    const data = await new Promise(resolve => { const reader = new FileReader(); reader.onload = () => resolve(reader.result.split(',')[1]); reader.readAsDataURL(blob); });
    const video = document.createElement('video'); video.src = a.href; video.muted = true;
    await new Promise((resolve,reject) => { video.onloadeddata = resolve; video.onerror = () => reject(new Error(`${a.download}: decode failed`)); });
    video.currentTime = 3.5;
    await new Promise((resolve,reject) => { video.onseeked = resolve; video.onerror = () => reject(new Error(`${a.download}: seek failed`)); });
    const decoded = { width: video.videoWidth, height: video.videoHeight, ready: video.readyState };
    video.removeAttribute('src'); video.load();
    return { name: a.download, data, decoded };
  })));
  for (const recording of recordings) {
    assert.ok(recording.decoded.width>0 && recording.decoded.height>0 && recording.decoded.ready>=2);
    await writeFile(`${proofDir}/${preset}-${recording.name}`, Buffer.from(recording.data,'base64'));
  }
  batchRuns.push({preset,guardLine,decoded:recordings.map(({name,decoded})=>({name,...decoded})),guardPoses:proof.guardPoses,holds:proof.holds,clips,violations:proof.violations,cameraPosition:proof.after.cameraPosition,zoom:proof.after.orbit.zoom});
 }
 // Exercise responsive camera selection and playback.
 await page.setViewportSize({width:390,height:844});
 await page.click('[data-camera="god"]');
 // Native touch events: one finger rotates; a spreading two-finger gesture zooms.
 await page.locator('#canvas3d').scrollIntoViewIfNeeded();
 const box=await page.locator('#canvas3d').boundingBox();
 const x=box.x+box.width/2,y=box.y+box.height/2;
 const cdp=await page.context().newCDPSession(page);
 const touch=(type,points)=>cdp.send('Input.dispatchTouchEvent',{type,touchPoints:points.map(([x,y],id)=>({x,y,id}))});
 const initial=await page.evaluate(()=>studioDiagnostics());
 await touch('touchStart',[[x,y]]);
 for(let i=1;i<=6;i++)await touch('touchMove',[[x+i*10,y+i*3]]);
 await touch('touchEnd',[]);await page.waitForTimeout(600);
 const rotated=await page.evaluate(()=>studioDiagnostics());
 assert.ok(distance(initial.cameraPosition,rotated.cameraPosition)>0.2,'touch orbit');
 await touch('touchStart',[[x-30,y],[x+30,y]]);
 for(let i=1;i<=6;i++)await touch('touchMove',[[x-30-i*6,y],[x+30+i*6,y]]);
 await touch('touchEnd',[]);await page.waitForTimeout(500);
 const pinched=await page.evaluate(()=>studioDiagnostics());
 assert.ok(distance(pinched.cameraPosition,pinched.orbit.target)<distance(rotated.cameraPosition,rotated.orbit.target),'pinch zoom');
 await page.click('[data-camera="god"]');
 await page.click('#btn-play');
 await page.waitForFunction(()=>!studioDiagnostics().busy,{},{timeout:15000});
 await page.screenshot({path:'/tmp/studio-god-mobile.png',fullPage:true});
 assert.deepEqual(errors,[]);
 console.log(JSON.stringify({url:page.url(),cases,clip,decoded,batchRuns,errors,diagnostics:await page.evaluate(()=>studioDiagnostics())},null,2));
} finally {await browser.close();}
