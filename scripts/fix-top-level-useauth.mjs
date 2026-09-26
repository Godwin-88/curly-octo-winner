// One-shot Phase 2 fix for a Phase 1 codemod defect: six procurement pages
// ended up with `const { token } = useAuth();` at MODULE top level, which
// crashes at runtime (useAuth must run inside a component). This moves the
// call to the first line of the default-exported component.
import fs from 'fs';

const files = [
  'app/(admin)/procurement/page.tsx',
  'app/(admin)/procurement/orders/page.tsx',
  'app/(admin)/procurement/payments/page.tsx',
  'app/(admin)/procurement/receipts/page.tsx',
  'app/(admin)/procurement/requisitions/page.tsx',
  'app/(admin)/procurement/suppliers/page.tsx',
];

for (const f of files) {
  let src = fs.readFileSync(f, 'utf8');
  if (!/^const \{ token \} = useAuth\(\);/m.test(src)) {
    console.log('SKIP', f);
    continue;
  }
  src = src.replace(/^const \{ token \} = useAuth\(\);\n\n/m, '');
  src = src.replace(/^(export default function \w+\(\) \{)$/m, '$1\n  const { token } = useAuth();');
  fs.writeFileSync(f, src);
  console.log('FIXED', f);
}
