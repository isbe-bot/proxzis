import * as THREE from 'three';
import { StudioRig } from './src/rig.js';

const rigA = new StudioRig(0x38bdf8, true);
const rigB = new StudioRig(0xef4444, false);

rigA.pose('ON_GUARD');
const snapA = rigA.snapshot();
rigA.pose('ROCK');
const snapB = rigA.snapshot();

console.log("lerp start");
rigA.applyInterpolation(snapA, snapB, 0.5);
console.log("lerp done. Pelvis pos:", rigA.bones.pelvis.position);
