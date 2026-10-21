import {defineConfig,devices} from '@playwright/test';
export default defineConfig({testDir:'tests/e2e',use:{...devices['Desktop Chrome'],channel:'chrome',headless:true},timeout:60000,reporter:'list'});
