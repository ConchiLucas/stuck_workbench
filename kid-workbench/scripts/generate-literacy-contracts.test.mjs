import {test} from 'node:test';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {readFileSync} from 'node:fs';
test('generated contracts match three canonical types',()=>{
 execFileSync(process.execPath,['scripts/generate-literacy-contracts.mjs','--check']);
 const schema=JSON.parse(readFileSync('contracts/literacy/question-types.json','utf8'));
 assert.deepEqual(schema.types.map(t=>t.code),['glyph_sense','sense_char','write_char']);
 assert(!schema.types.find(t=>t.code==='write_char').materials.includes('sense'));
});
