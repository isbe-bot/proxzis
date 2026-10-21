// Run after browser-smoke.mjs: fully decode every retained recording and save
// its last decoded frame. Requires ffmpeg/ffprobe, not a new npm dependency.
import { readdir, readFile, writeFile } from 'node:fs/promises';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { join } from 'node:path';
import assert from 'node:assert/strict';
const directory = process.argv[2];
if (!directory) throw new Error('Usage: node test/verify-webm.mjs <proof-directory>');
function run(program, args) {
  const result = spawnSync(program, args, { encoding: 'utf8', maxBuffer: 8*1024*1024 });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${program}: ${result.stderr}`);
  return result.stdout;
}
const files = (await readdir(directory)).filter(name => name.endsWith('.webm')).sort();
assert.ok(files.length >= 9, 'Need all nine matchup recordings');
const reports = [];
for (const file of files) {
  const path = join(directory, file);
  const data = await readFile(path);
  const probe = JSON.parse(run('ffprobe', ['-v','error','-count_frames','-show_entries','stream=codec_name,width,height,r_frame_rate,avg_frame_rate,nb_read_frames','-show_entries','format=format_name,duration','-of','json',path]));
  const video = probe.streams[0];
  const frames = Number(video.nb_read_frames);
  assert.match(probe.format.format_name, /webm/);
  assert.ok(data.length > 10000 && frames >= 10 && video.width > 0 && video.height > 0, file);
  if (process.env.EXPECT_FPS) {
    const fps = Number(process.env.EXPECT_FPS);
    const timestamps = JSON.parse(run('ffprobe',['-v','error','-select_streams','v:0','-show_entries','frame=best_effort_timestamp_time','-of','json',path])).frames.map(frame=>Number(frame.best_effort_timestamp_time));
    assert.equal(video.r_frame_rate,`${fps}/1`,`${file}: advertised rate`);
    timestamps.slice(1).forEach((time,i)=>assert.ok(Math.abs(time-timestamps[i]-1/fps)<0.0011,`${file}: dropped/irregular frame ${i}`));
    assert.equal(video.width,720); assert.equal(video.height,660);
    assert.ok(data.length<400000,`${file}: oversized export ${data.length}`);
    video.timestampIntervalsVerified = timestamps.length-1;
  }
  run('ffmpeg', ['-v','error','-xerror','-i',path,'-f','null','-']);
  run('ffmpeg', ['-v','error','-xerror','-i',path,'-vf',`select=eq(n\\,${frames-1}),scale=640:-1`,'-frames:v','1','-update','1','-y',`${path}-final.png`]);
  reports.push({ file, bytes: data.length, duration: Number(probe.format.duration), sha256: createHash('sha256').update(data).digest('hex'), ...video, decoded: true, finalFrame: `${file}-final.png` });
}
const report = { clips: reports.length, decodeErrors: 0, reports };
await writeFile(join(directory,'webm-verification.json'), JSON.stringify(report,null,2));
console.log(JSON.stringify(report,null,2));
