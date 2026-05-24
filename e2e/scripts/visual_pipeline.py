#!/usr/bin/env python3
"""
visual_pipeline.py — Multi-agent visual QA pipeline for HWC
Orchestrates: capture → diff → repair → verify → commit

Usage:
    python3 visual_pipeline.py --capture          # Capture screenshots
    python3 visual_pipeline.py --diff            # Compare to baselines
    python3 visual_pipeline.py --reference      # Compare to reference image
    python3 visual_pipeline.py --full             # Run full pipeline
    python3 visual_pipeline.py --repair          # Generate repair plan
"""

import os
import sys
import json
import base64
import subprocess
from pathlib import Path
from datetime import datetime
from typing import Optional, Tuple, Dict, List

try:
    from PIL import Image, ImageChops, ImageStat, ImageDraw, ImageFont
except ImportError:
    print("ERROR: Pillow not installed. Run: pip install Pillow")
    sys.exit(1)

# Paths
HOST = "sean@172.19.0.1"
SSH_KEY = "/home/hermeswebui/.hermes/container_key"
BASE_URL = "http://localhost:3113"
BASE_DIR = "/tmp/hwc-qa"
SCP_BASE = f"{HOST}:{BASE_DIR}"

SCREENSHOTS_DIR = Path(BASE_DIR) / 'screenshots'
BASELINES_DIR = Path(BASE_DIR) / 'baselines'
REFERENCES_DIR = Path(BASE_DIR) / 'references'
RESULTS_DIR = Path(BASE_DIR) / 'results'

REGRESSION_THRESHOLD = 1.0  # % diff before regression failure
REFERENCE_THRESHOLD = 0.85  # similarity score (0-1) before repair needed


def ssh(cmd: str, timeout: int = 30) -> str:
    """Run command on host via SSH."""
    result = subprocess.run(
        f"ssh -i {SSH_KEY} -o StrictHostKeyChecking=no -o LogLevel=ERROR sean@172.19.0.1 \"{cmd}\"",
        shell=True, capture_output=True, text=True, timeout=timeout
    )
    return result.stdout + result.stderr


def scp(src: str, dst: str) -> None:
    """SCP file from host to local."""
    subprocess.run(
        f"scp -i {SSH_KEY} {SCP_BASE}{src} {dst}",
        shell=True, capture_output=True, timeout=30
    )


def ensure_dirs():
    for d in [SCREENSHOTS_DIR, BASELINES_DIR, REFERENCES_DIR, RESULTS_DIR]:
        d.mkdir(parents=True, exist_ok=True)


def capture_screenshot(viewport: str = "1440x900") -> Optional[Path]:
    """Capture screenshot using chrome on host."""
    timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
    w, h = viewport.split('x')
    
    # Build command with separate remote path
    remote_path = f"/tmp/hwc-qa/screenshots/capture-{timestamp}.png"
    
    # Run chrome on host, capture stderr to suppress VAAPI errors
    result = ssh(
        f"google-chrome-stable --headless --disable-gpu --no-sandbox "
        f"--virtual-time-budget=12000 --window-size={viewport} "
        f"--screenshot={remote_path} --disable-web-security {BASE_URL} "
        f"2>/dev/null && test -f {remote_path} && echo 'OK'"
    )
    
    if 'OK' in result:
        local_path = SCREENSHOTS_DIR / f"current-{viewport}.png"
        scp(f"/screenshots/capture-{timestamp}.png", str(local_path))
        ssh(f"rm -f {remote_path}")
        return local_path if local_path.exists() else None
    return None


def pixel_analysis(img1: Path, img2: Path) -> Dict:
    """Detailed pixel-level analysis between two images."""
    im1 = Image.open(img1)
    im2 = Image.open(img2)
    
    # Ensure same size
    if im1.size != im2.size:
        im2 = im2.resize(im1.size, Image.LANCZOS)
    
    # Grayscale diff
    g1, g2 = im1.convert('L'), im2.convert('L')
    diff = ImageChops.difference(g1, g2)
    
    # Histogram analysis
    hist = diff.histogram()
    total_pixels = im1.size[0] * im1.size[1]
    
    # Count pixels at each threshold
    thresholds = [3, 5, 10, 20, 30, 50]
    diff_counts = {}
    for t in thresholds:
        diff_counts[t] = sum(hist[i] * (i > t) for i in range(len(hist)))
    
    diff_percent = (diff_counts[10] / total_pixels) * 100
    
    # Color analysis — get dominant colors in changed regions
    # (simplified: just get overall color stats)
    r1, g1_b, b1 = im1.convert('RGB').split()
    r2, g2_b, b2 = im2.convert('RGB').split()
    
    return {
        'width': im1.size[0],
        'height': im1.size[1],
        'total_pixels': total_pixels,
        'diff_percent': round(diff_percent, 4),
        'diff_counts': {k: round(v, 2) for k, v in diff_counts.items()},
        'passed': diff_percent <= REGRESSION_THRESHOLD
    }


def generate_diff_image(img1: Path, img2: Path, output: Path) -> None:
    """Generate annotated diff image showing changes."""
    im1 = Image.open(img1)
    im2 = Image.open(img2)
    
    if im1.size != im2.size:
        im2 = im2.resize(im1.size, Image.LANCZOS)
    
    g1, g2 = im1.convert('L'), im2.convert('L')
    diff = ImageChops.difference(g1, g2)
    
    # Resize diff for visibility (4x)
    diff_large = diff.resize((im1.size[0] * 2, im1.size[1] * 2), Image.NEAREST)
    
    # Create composite: left half = current, right half = diff
    combined = Image.new('RGB', (im1.size[0] * 3, im1.size[1] * 2))
    
    # Current (left)
    combined.paste(im1.resize((im1.size[0], im1.size[1]), Image.LANCZOS), (0, 0))
    # Baseline/reference (middle)
    combined.paste(im2.resize((im1.size[0], im1.size[1]), Image.LANCZOS), (im1.size[0], 0))
    # Diff (bottom, full width)
    diff_upscaled = Image.new('RGB', (im1.size[0], im1.size[1]))
    for x in range(im1.size[0]):
        for y in range(im1.size[1]):
            p = diff.getpixel((x, y))
            if p > 10:
                diff_upscaled.putpixel((x, y), (255, 0, 0))
            elif p > 3:
                diff_upscaled.putpixel((x, y), (255, 255, 0))
    combined.paste(diff_upscaled.resize((im1.size[0] * 2, im1.size[1]), Image.NEAREST), (0, im1.size[1]))
    
    combined.save(output)


def analyze_css_issues(img1: Path, img2: Path) -> List[Dict]:
    """Analyze visual differences to infer CSS issues (heuristic)."""
    im1 = Image.open(img1).convert('RGB')
    im2 = Image.open(img2).convert('RGB')
    
    if im1.size != im2.size:
        im2 = im2.resize(im1.size, Image.LANCZOS)
    
    # Divide into regions (top, middle, bottom, left, right, center)
    w, h = im1.size
    regions = {
        'top': (0, 0, w, h//4),
        'middle': (0, h//4, w, 3*h//4),
        'bottom': (0, 3*h//4, w, h),
        'left': (0, 0, w//4, h),
        'right': (3*w//4, 0, w, h),
        'center': (w//4, h//4, 3*w//4, 3*h//4),
    }
    
    issues = []
    for name, (x1, y1, x2, y2) in regions.items():
        region1 = im1.crop((x1, y1, x2, y2))
        region2 = im2.crop((x1, y1, x2, y2))
        
        g1, g2 = region1.convert('L'), region2.convert('L')
        diff = ImageChops.difference(g1, g2)
        stat = ImageStat.Stat(diff)
        
        # Average brightness diff
        avg_diff = sum(stat.mean) / len(stat.mean) if stat.mean else 0
        
        if avg_diff > 5:  # Significant difference
            issues.append({
                'region': name,
                'avg_pixel_diff': round(avg_diff, 2),
                'likely_issue': _infer_issue(name, avg_diff)
            })
    
    return issues


def _infer_issue(region: str, magnitude: float) -> str:
    """Heuristic: guess CSS issue from region and magnitude."""
    # This is a simplified heuristic — real version would use ML
    infer = {
        'top': f"Background gradient or top bar styling — diff magnitude {magnitude:.1f}",
        'bottom': f"Dock or bottom panel — check backdrop-blur and icon spacing",
        'left': f"Left sidebar — panel opacity or border color",
        'right': f"Right panel — chat area styling",
        'center': f"Tile grid area — check shadow depth and border radius",
        'middle': f"General layout — possible padding/margin changes"
    }
    return infer.get(region, f"Unknown region — diff {magnitude:.1f}")


def capture_command(viewports: List[str] = None):
    """Capture screenshots at specified viewports."""
    if viewports is None:
        viewports = ["1440x900", "1280x720", "1920x1080"]
    
    ensure_dirs()
    results = []
    
    print("=== CAPTURE ===")
    for vp in viewports:
        print(f"\nCapturing {vp}...")
        path = capture_screenshot(vp)
        if path:
            print(f"  ✓ Saved: {path.name} ({path.stat().st_size} bytes)")
            
            # Store as baseline if none
            baseline = BASELINES_DIR / f"baseline-{vp}.png"
            if not baseline.exists():
                import shutil
                shutil.copy(path, baseline)
                print(f"  ✓ Stored as baseline")
            
            results.append(str(path))
        else:
            print(f"  ✗ Failed")
    
    return results


def diff_command():
    """Compare current to baselines (regression check)."""
    ensure_dirs()
    
    baselines = list(BASELINES_DIR.glob("baseline-*.png"))
    if not baselines:
        print("⚠ No baselines found. Run --capture first.")
        return []
    
    print("=== REGRESSION DIFF ===")
    results = []
    
    for baseline in baselines:
        vp = baseline.stem.replace("baseline-", "")
        current = SCREENSHOTS_DIR / f"current-{vp}.png"
        
        if not current.exists():
            print(f"\n⚠ No current screenshot for {vp}")
            continue
        
        print(f"\nComparing {vp}...")
        analysis = pixel_analysis(current, baseline)
        
        status = "✓ PASS" if analysis['passed'] else "✗ FAIL REGRESSION"
        print(f"  Diff: {analysis['diff_percent']}% (threshold: {REGRESSION_THRESHOLD}%)")
        print(f"  {status}")
        
        # Generate diff image
        diff_path = RESULTS_DIR / f"regression-diff-{vp}.png"
        generate_diff_image(current, baseline, diff_path)
        print(f"  Diff image: {diff_path}")
        
        analysis['viewport'] = vp
        analysis['current'] = str(current)
        analysis['baseline'] = str(baseline)
        analysis['diff_image'] = str(diff_path)
        results.append(analysis)
        
        if not analysis['passed']:
            print(f"  ⚠ REGRESSION DETECTED")
    
    return results


def reference_command(reference_path: str):
    """Compare current to reference image (quality check)."""
    ensure_dirs()
    
    ref = Path(reference_path)
    if not ref.exists():
        print(f"⚠ Reference image not found: {reference_path}")
        return None
    
    current = SCREENSHOTS_DIR / "current-1440x900.png"
    if not current.exists():
        print("⚠ No current screenshot. Run --capture first.")
        return None
    
    print(f"=== REFERENCE COMPARE ===")
    print(f"Reference: {ref}")
    print(f"Current:   {current}")
    
    analysis = pixel_analysis(current, ref)
    issues = analyze_css_issues(current, ref)
    
    similarity = max(0, 100 - analysis['diff_percent']) / 100
    needs_repair = similarity < REFERENCE_THRESHOLD
    
    print(f"\nSimilarity: {similarity:.2%} (threshold: {REFERENCE_THRESHOLD:.2%})")
    print(f"Status: {'✗ NEEDS REPAIR' if needs_repair else '✓ ACCEPTABLE'}")
    
    if issues:
        print(f"\n{len(issues)} issue region(s) detected:")
        for issue in issues:
            print(f"  [{issue['region']}] {issue['likely_issue']}")
    
    # Generate annotated diff
    diff_path = RESULTS_DIR / "reference-diff.png"
    generate_diff_image(current, ref, diff_path)
    print(f"\nDiff image: {diff_path}")
    
    return {
        'similarity': similarity,
        'needs_repair': needs_repair,
        'issues': issues,
        'diff_image': str(diff_path),
        'analysis': analysis
    }


def repair_plan(issues: List[Dict]) -> str:
    """Generate a repair plan from detected issues."""
    plan = [
        "# CSS Repair Plan",
        "",
        "Based on visual diff analysis, the following CSS fixes are recommended:",
        ""
    ]
    
    region_to_component = {
        'top': 'App.svelte (background gradient, top bar)',
        'bottom': 'Dock.svelte (bottom dock bar)',
        'left': 'LeftPanel.svelte (sidebar with Files/Apps tabs)',
        'right': 'RightPanel.svelte (agent chat panel)',
        'center': 'Tile.svelte (tile grid container)',
        'middle': 'MiddlePanel.svelte (main tiling area)'
    }
    
    for issue in issues:
        region = issue['region']
        component = region_to_component.get(region, 'Unknown')
        plan.append(f"## {region.upper()} Region")
        plan.append(f"**Component:** {component}")
        plan.append(f"**Issue:** {issue['likely_issue']}")
        plan.append(f"**Pixel diff:** {issue['avg_pixel_diff']}")
        plan.append("")
        plan.append("**Recommended actions:**")
        plan.append(f"1. Check `bg-` and `opacity-` classes in {component}")
        plan.append(f"2. Verify `backdrop-blur-*` and `shadow-*` values")
        plan.append(f"3. Check border-radius consistency")
        plan.append("")
    
    return "\n".join(plan)


def full_pipeline():
    """Run full visual QA pipeline."""
    print("=" * 60)
    print("HWC VISUAL QA PIPELINE")
    print("=" * 60)
    
    # 1. Capture
    print()
    captures = capture_command()
    
    # 2. Regression diff (against baselines)
    print()
    regression_results = diff_command()
    
    # Summary
    print()
    print("=" * 60)
    print("PIPELINE SUMMARY")
    print("=" * 60)
    
    regressions_detected = sum(1 for r in regression_results if not r['passed'])
    print(f"Regression check: {len(regression_results)} viewports tested")
    print(f"Regressions detected: {regressions_detected}")
    
    # Save results
    results_file = RESULTS_DIR / f"pipeline-results-{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
    full_results = {
        'timestamp': datetime.now().isoformat(),
        'captures': captures,
        'regression_results': regression_results,
        'regressions_detected': regressions_detected
    }
    with open(results_file, 'w') as f:
        json.dump(full_results, f, indent=2)
    print(f"\nFull results: {results_file}")
    
    return full_results


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)
    
    cmd = sys.argv[1]
    
    if cmd == '--capture':
        viewports = sys.argv[2:] if len(sys.argv) > 2 else None
        capture_command(viewports)
    elif cmd == '--diff':
        diff_command()
    elif cmd == '--reference':
        if len(sys.argv) < 3:
            print("Usage: --reference <path-to-reference-image>")
            sys.exit(1)
        result = reference_command(sys.argv[2])
        if result and result['issues']:
            plan = repair_plan(result['issues'])
            print("\n" + "=" * 60)
            print("REPAIR PLAN")
            print("=" * 60)
            print(plan)
    elif cmd == '--full':
        full_pipeline()
    elif cmd == '--repair':
        # Generate repair plan from stored diffs
        print("Run --reference <image> first to generate repair plan")
    else:
        print(f"Unknown command: {cmd}")
        print(__doc__)


if __name__ == '__main__':
    main()