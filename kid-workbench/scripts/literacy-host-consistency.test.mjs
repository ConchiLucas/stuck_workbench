import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';

const hosts = [
 'literacy-app/src/pages/PracticePage.tsx',
 'task-admin/frontend/src/pages/GenerationPage.tsx',
 'content-admin/frontend/src/features/literacy/MaterialPracticePreview.tsx',
];
test('formal hosts use shared player without demo or online evaluation', () => {
 for (const path of hosts) {
  const source = readFileSync(path,'utf8');
  assert.match(source, /@kid-workbench\/literacy-player/,path);
  assert.doesNotMatch(source, /demoBanks|loadCharacterData|speechSynthesis|SpeechSynthesisUtterance/,path);
 }
});
test('three builds use local shared packages', () => {
 for (const dir of ['literacy-app','task-admin/frontend','content-admin/frontend']) {
  const pkg=JSON.parse(readFileSync(`${dir}/package.json`,'utf8'));
  for (const name of ['literacy-player','literacy-contract']) assert.match(pkg.dependencies[`@kid-workbench/${name}`],/^file:/,dir);
 }
});
