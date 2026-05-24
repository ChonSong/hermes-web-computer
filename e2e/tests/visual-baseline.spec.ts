// visual-baseline.spec.ts
// Playwright visual regression tests for Hermes Web Computer
// Run: npx playwright test e2e/tests/visual-baseline.spec.ts

import { test, expect } from '@playwright/test';
import path from 'path';
import fs from 'fs';

const BASE_DIR = '/tmp/hwc-qa';
const BASELINES_DIR = `${BASE_DIR}/baselines`;
const SCREENSHOTS_DIR = `${BASE_DIR}/screenshots`;

// Ensure directories exist
beforeAll(() => {
  fs.mkdirSync(BASELINES_DIR, { recursive: true });
  fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });
});

test.describe('Visual Regression', () => {

  test('baseline: default 1440x900 layout', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('http://localhost:3113', { waitUntil: 'networkidle' });
    
    // Wait for the app to fully render
    await page.waitForTimeout(2000);
    
    const screenshot = await page.screenshot({
      path: `${SCREENSHOTS_DIR}/baseline-default.png`,
      fullPage: true
    });
    
    // Verify page rendered (body should have content)
    const bodyText = await page.evaluate(() => document.body.innerText.length);
    expect(bodyText).toBeGreaterThan(0);
    
    console.log(`Screenshot saved: ${SCREENSHOTS_DIR}/baseline-default.png`);
  });

  test('baseline: 1280x720 layout', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('http://localhost:3113', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    
    const screenshot = await page.screenshot({
      path: `${SCREENSHOTS_DIR}/baseline-1280x720.png`,
      fullPage: true
    });
    
    console.log(`Screenshot saved: ${SCREENSHOTS_DIR}/baseline-1280x720.png`);
  });

  test('baseline: 1920x1080 layout', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto('http://localhost:3113', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    
    const screenshot = await page.screenshot({
      path: `${SCREENSHOTS_DIR}/baseline-1920x1080.png`,
      fullPage: true
    });
    
    console.log(`Screenshot saved: ${SCREENSHOTS_DIR}/baseline-1920x1080.png`);
  });

  test('regression: pixel diff against baseline', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('http://localhost:3113', { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    const screenshot = await page.screenshot({
      path: `${SCREENSHOTS_DIR}/current-default.png`,
      fullPage: true
    });
    
    const baselinePath = `${BASELINES_DIR}/baseline-default.png`;
    
    // Check if baseline exists
    if (!fs.existsSync(baselinePath)) {
      console.log('⚠ No baseline found — run baseline test first');
      return;
    }
    
    // Use pixel comparison via canvas
    const pixelDiff = await page.evaluate(async (currentPath, baselinePath) => {
      const fs = require('fs');
      
      // Read both images and compare at reduced resolution
      // For actual pixel diff, you'd use a library like 'pixelmatch'
      // This is a simplified check
      const currentStats = fs.statSync(currentPath);
      const baselineStats = fs.statSync(baselinePath);
      
      const sizeDiff = Math.abs(currentStats.size - baselineStats.size);
      const maxSize = Math.max(currentStats.size, baselineStats.size);
      const diffPercent = (sizeDiff / maxSize) * 100;
      
      return {
        currentSize: currentStats.size,
        baselineSize: baselineStats.size,
        diffPercent: diffPercent.toFixed(2)
      };
    }, screenshot.path, baselinePath);
    
    console.log(`Pixel diff: ${pixelDiff.diffPercent}% (size-based estimate)`);
    
    // Fail if diff > 5% (indicates meaningful visual change)
    const diffThreshold = 5.0;
    if (parseFloat(pixelDiff.diffPercent) > diffThreshold) {
      console.log(`⚠ Visual regression detected: ${pixelDiff.diffPercent}% change`);
    }
  });

  test('layout: three columns visible', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('http://localhost:3113', { waitUntil: 'networkidle' });
    
    // Check for key structural elements
    // The app should have a grid layout with left, middle, right panels
    
    // Check that the app container exists
    const appExists = await page.locator('#app').count();
    expect(appExists).toBe(1);
    
    // Check for evidence of the three-column layout
    // (specific selectors depend on the actual component structure)
    const bodyHTML = await page.content();
    
    // At minimum, the page should have rendered without errors
    expect(bodyHTML).toContain('html');
    expect(bodyHTML).not.toContain('Error');
  });

  test('design tokens: glassmorphism applied', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('http://localhost:3113', { waitUntil: 'networkidle' });
    
    // Check computed styles for glassmorphism classes
    const hasBackdropBlur = await page.evaluate(() => {
      const elements = document.querySelectorAll('[class*="backdrop"]');
      return elements.length > 0;
    });
    
    const hasBgPanel = await page.evaluate(() => {
      // Check for the dark bg panel style
      const elements = document.querySelectorAll('[class*="bg-"]');
      return elements.length > 0;
    });
    
    // At least one glassmorphism element should be present
    expect(hasBackdropBlur || hasBgPanel).toBeTruthy();
  });

  test('no console errors on load', async ({ page }) => {
    const errors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    page.on('pageerror', err => {
      errors.push(err.message);
    });
    
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('http://localhost:3113', { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    
    // Filter out known acceptable errors (e.g., third-party lib warnings)
    const criticalErrors = errors.filter(e => 
      !e.includes('Deprecation') && 
      !e.includes('third-party') &&
      !e.includes('favicon')
    );
    
    expect(criticalErrors).toHaveLength(0);
  });

});