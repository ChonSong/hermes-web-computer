#!/usr/bin/env python3
"""
visual_compare.py — Pixel-level visual regression for HWC
Compares current screenshots against baselines using PIL

Usage:
    python3 visual_compare.py --capture       # Take new screenshots
    python3 visual_compare.py --compare      # Compare to baselines
    python3 visual_compare.py --baseline     # Store current as baseline
"""

import os
import sys
import json
import subprocess
from pathlib import Path
from datetime import datetime
from typing import Optional, Tuple, Dict

try:
    from PIL import Image, ImageChops, ImageStat
except ImportError:
    print("ERROR: Pillow not installed. Run: pip install Pillow")
    sys.exit(1)

BASE_DIR = Path(os.environ.get('HWC_QA_BASE', '/tmp/hwc-qa'))
SCREENSHOTS_DIR = BASE_DIR / 'screenshots'
BASELINES_DIR = BASE_DIR / 'baselines'
RESULTS_DIR = BASE_DIR / 'results'

THRESHOLD = 5.0  # Max % pixel difference before failure


def ensure_dirs():
    """Create necessary directories."""
    for d in [SCREENSHOTS_DIR, BASELINES_DIR, RESULTS_DIR]:
        d.mkdir(parents=True, exist_ok=True)


def capture_with_chrome(url: str = 'http://localhost:3113') -> Optional[Path]:
    """Capture screenshot using google-chrome-stable."""
    timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
    output_path = SCREENSHOTS_DIR / f'screenshot-{timestamp}.png'
    
    cmd = [
        'google-chrome-stable',
        '--headless',
        '--disable-gpu',
        '--no-sandbox',
        '--virtual-time-budget=10000',
        '--window-size=1440,900',
        f'--screenshot={output_path}',
        '--disable-web-security',
        url
    ]
    
    try:
        result = subprocess.run(cmd, capture_output=True, timeout=30)
        if result.returncode == 0 and output_path.exists():
            return output_path
        else:
            print(f"Chrome capture failed: {result.stderr.decode()}")
            return None
    except subprocess.TimeoutExpired:
        print("Chrome capture timed out")
        return None
    except FileNotFoundError:
        print("google-chrome-stable not found. Install chromium first.")
        return None


def pixel_diff(img1: Image.Image, img2: Image.Image) -> Tuple[float, Image.Image]:
    """Calculate percentage of differing pixels between two images."""
    # Resize to same size if needed
    if img1.size != img2.size:
        img2 = img2.resize(img1.size, Image.LANCZOS)
    
    # Convert to grayscale for comparison
    g1 = img1.convert('L')
    g2 = img2.convert('L')
    
    # Get absolute difference
    diff = ImageChops.difference(g1, g2)
    
    # Calculate percentage of non-zero pixels
    stat = ImageStat.Stat(diff)
    # stat.mean gives average pixel difference (0-255)
    # We consider any pixel with >10 difference as "different"
    different_pixels = sum(1 for v in stat.mean if v > 10)
    total_pixels = len(stat.mean)
    
    diff_percent = (different_pixels / total_pixels) * 100 if total_pixels > 0 else 0
    
    return diff_percent, diff


def analyze_screenshot(path: Path) -> Dict:
    """Analyze a screenshot for visual metrics."""
    img = Image.open(path)
    
    # Color analysis
    colors = img.getcolors(maxcolors=1000000)
    unique_colors = len(colors) if colors else 0
    
    # Brightness analysis
    gray = img.convert('L')
    stat = ImageStat.Stat(gray)
    avg_brightness = stat.mean[0] if stat.mean else 0
    
    # Check for blank areas (might indicate rendering issues)
    width, height = img.size
    
    return {
        'width': width,
        'height': height,
        'unique_colors': unique_colors,
        'avg_brightness': round(avg_brightness, 2),
        'file_size': path.stat().st_size
    }


def compare_to_baseline(screenshot_path: Path, baseline_path: Path) -> Dict:
    """Compare a screenshot against its baseline."""
    current = Image.open(screenshot_path)
    baseline = Image.open(baseline_path)
    
    diff_percent, diff_img = pixel_diff(current, baseline)
    
    # Generate annotated diff image
    diff_output = RESULTS_DIR / f'diff-{screenshot_path.stem}.png'
    diff_img.save(diff_output)
    
    return {
        'current': str(screenshot_path),
        'baseline': str(baseline_path),
        'diff_percent': round(diff_percent, 2),
        'diff_image': str(diff_output),
        'passed': diff_percent <= THRESHOLD
    }


def capture_command():
    """Capture screenshots for all viewports."""
    ensure_dirs()
    
    viewports = [
        ('1440x900', 1440, 900),
        ('1280x720', 1280, 720),
        ('1920x1080', 1920, 1080),
    ]
    
    results = []
    for name, w, h in viewports:
        print(f"\nCapturing {name}...")
        # Note: Chrome window-size flag handles the viewport
        path = capture_with_chrome()
        if path:
            print(f"  ✓ Saved: {path.name}")
            
            # Copy to timestamped location for this viewport
            viewport_path = SCREENSHOTS_DIR / f'screenshot-{name}.png'
            path.rename(viewport_path)
            
            # Store as baseline if none exists
            baseline_path = BASELINES_DIR / f'baseline-{name}.png'
            if not baseline_path.exists():
                import shutil
                shutil.copy(viewport_path, baseline_path)
                print(f"  ✓ Stored as baseline: baseline-{name}.png")
            
            results.append(str(viewport_path))
        else:
            print(f"  ✗ Failed")
    
    return results


def compare_command():
    """Compare current screenshots to baselines."""
    ensure_dirs()
    
    baselines = list(BASELINES_DIR.glob('baseline-*.png'))
    if not baselines:
        print("⚠ No baselines found. Run --capture first to create baselines.")
        return
    
    results = []
    for baseline in baselines:
        name = baseline.stem.replace('baseline-', '')
        
        # Find corresponding current screenshot
        current = SCREENSHOTS_DIR / f'screenshot-{name}.png'
        if not current.exists():
            print(f"⚠ No current screenshot for {name}")
            continue
        
        print(f"\nComparing {name}...")
        result = compare_to_baseline(current, baseline)
        
        status = "✓ PASS" if result['passed'] else "✗ FAIL"
        print(f"  Diff: {result['diff_percent']}% (threshold: {THRESHOLD}%)")
        print(f"  {status}")
        
        if not result['passed']:
            print(f"  Diff image: {result['diff_image']}")
        
        results.append(result)
    
    # Summary
    print("\n" + "="*50)
    print("SUMMARY")
    print("="*50)
    passed = sum(1 for r in results if r['passed'])
    failed = sum(1 for r in results if not r['passed'])
    print(f"Passed: {passed}/{len(results)}")
    print(f"Failed: {failed}/{len(results)}")
    
    # Save results JSON
    results_path = RESULTS_DIR / f'results-{datetime.now().strftime("%Y%m%d_%H%M%S")}.json'
    with open(results_path, 'w') as f:
        json.dump(results, f, indent=2)
    print(f"\nResults saved to: {results_path}")
    
    return results


def baseline_command():
    """Store current screenshots as baselines."""
    ensure_dirs()
    
    screenshots = list(SCREENSHOTS_DIR.glob('screenshot-*.png'))
    if not screenshots:
        print("⚠ No screenshots found. Run --capture first.")
        return
    
    import shutil
    for screenshot in screenshots:
        baseline = BASELINES_DIR / f'baseline-{screenshot.name}'
        shutil.copy(screenshot, baseline)
        print(f"✓ Stored: {baseline.name}")
    
    print(f"\nStored {len(screenshots)} baselines")


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)
    
    cmd = sys.argv[1]
    
    if cmd == '--capture':
        capture_command()
    elif cmd == '--compare':
        compare_command()
    elif cmd == '--baseline':
        baseline_command()
    elif cmd == '--help':
        print(__doc__)
    else:
        print(f"Unknown command: {cmd}")
        print(__doc__)
        sys.exit(1)


if __name__ == '__main__':
    main()