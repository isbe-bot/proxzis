import * as THREE from 'three';
import { StudioRig } from './src/rig.js';

const rigA = new StudioRig(0x38bdf8, true);
const rigB = new StudioRig(0xef4444, false);

rigA.pose('STRIKE');
console.log("botA after STRIKE pose, shoulderLead rot:", rigA.bones.shoulderLead.rotation);
