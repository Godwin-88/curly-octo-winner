// Codemod: revive the 36 "dead" admin pages that declared
//   const token = ''; // TODO: Get from auth context
// and therefore never fetched any data (the loader early-returns on an empty
// token). Replaces the dead const with the shared useAuth() hook and adds the
// import. Run once: node scripts/fix-dead-token-pages.mjs
import { readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const ROOT = 'web/app';
const DEAD = "const token = ''; // TODO: Get from auth context";
const REPLACEMENT = 'const { token } = useAuth();';

let fixed = 0;

function walk(dir) {
  for (const entry of readdirSync(dir)) {
    const p = join(dir, entry);
    const s = statSync(p);
    if (s.isDirectory()) {
      walk(p);
    } else if (p.endsWith('.tsx')) {
      let src = readFileSync(p, 'utf8');
      if (!src.includes(DEAD)) continue;
      if (!src.includes("@/lib/auth")) {
        src = src.replace("'use client';", "'use client';\n\nimport { useAuth } from '@/lib/auth';");
      }
      src = src.split(DEAD).join(REPLACEMENT);
      writeFileSync(p, src);
      fixed++;
      console.log('fixed', p);
    }
  }
}

walk(ROOT);
console.log(`done: ${fixed} files fixed`);
