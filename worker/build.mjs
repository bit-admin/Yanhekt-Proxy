import { build } from 'esbuild';
import JavaScriptObfuscator from 'javascript-obfuscator';
import { writeFileSync, mkdirSync } from 'fs';

// Bundle TypeScript to JavaScript
const result = await build({
  entryPoints: ['src/index.ts'],
  bundle: true,
  minify: true,
  format: 'esm',
  target: 'es2022',
  write: false,
  outfile: 'dist/index.js',
});

const bundledCode = result.outputFiles[0].text;

// Obfuscate the bundled code
const obfuscatedCode = JavaScriptObfuscator.obfuscate(bundledCode, {
  compact: true,
  controlFlowFlattening: true,
  controlFlowFlatteningThreshold: 0.75,
  deadCodeInjection: true,
  deadCodeInjectionThreshold: 0.4,
  debugProtection: false,
  disableConsoleOutput: false,
  identifierNamesGenerator: 'hexadecimal',
  log: false,
  numbersToExpressions: true,
  renameGlobals: false,
  selfDefending: false,
  simplify: true,
  splitStrings: true,
  splitStringsChunkLength: 10,
  stringArray: true,
  stringArrayCallsTransform: true,
  stringArrayCallsTransformThreshold: 0.75,
  stringArrayEncoding: ['base64'],
  stringArrayIndexShift: true,
  stringArrayRotate: true,
  stringArrayShuffle: true,
  stringArrayWrappersCount: 2,
  stringArrayWrappersChainedCalls: true,
  stringArrayWrappersParametersMaxCount: 4,
  stringArrayWrappersType: 'function',
  stringArrayThreshold: 0.75,
  transformObjectKeys: true,
  unicodeEscapeSequence: false,
}).getObfuscatedCode();

// Write output
mkdirSync('dist', { recursive: true });
writeFileSync('dist/index.js', obfuscatedCode);

console.log('Build complete: dist/index.js');
