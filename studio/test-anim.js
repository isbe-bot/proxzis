import * as THREE from 'three';
import { StudioRig } from './src/rig.js';

const rigA = new StudioRig(0x38bdf8, true);
const rigB = new StudioRig(0xef4444, false);

rigA.pose('ROCK');
console.log("botA after ROCK pose, shoulderLead rot:", rigA.bones.shoulderLead.rotation);
