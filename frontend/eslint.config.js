import svelte from 'eslint-plugin-svelte';
import tsParser from '@typescript-eslint/parser';
export default [
 { ignores: ['.svelte-kit/**','build/**'] },
 ...svelte.configs.recommended,
 { files: ['**/*.svelte'], languageOptions: { parserOptions: { parser: tsParser } }, rules: { 'svelte/no-navigation-without-resolve': 'off', 'svelte/require-each-key': 'off' } }
];
