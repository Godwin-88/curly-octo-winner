import nextCoreWebVitals from 'eslint-config-next/core-web-vitals';
import nextTypescript from 'eslint-config-next/typescript';

// eslint-config-next v16 ships native flat configs (Linter.Config[]).
const eslintConfig = [
  ...nextCoreWebVitals,
  ...nextTypescript,
  {
    ignores: ['.next/**', 'node_modules/**', 'out/**', 'next-env.d.ts'],
  },
  {
    rules: {
      // --- Tracked legacy debt (warn, not error) ---
      // The pre-Phase-2 codebase has ~111 `any` usages, ~39 effect-deps
      // deviations and ~34 setState-in-effect instances. They remain visible
      // (and countable via `npx eslint .`) but do not block CI until the
      // React Query / typing refactor lands (Phase 3). New regressions in
      // everything else — rules-of-hooks, immutability, a11y, next rules —
      // are hard errors.
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': [
        'warn',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      'react-hooks/exhaustive-deps': 'warn',
      'react-hooks/set-state-in-effect': 'warn',
    },
  },
];

export default eslintConfig;
