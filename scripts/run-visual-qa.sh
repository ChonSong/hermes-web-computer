#!/bin/bash
# run-visual-qa.sh — Runs on host via SSH for nightly visual QA
# Captures screenshot, compares to baseline, logs results

BASE_DIR="/tmp/hwc-qa"
LOG_FILE="$BASE_DIR/qa-results.log"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

mkdir -p "$BASE_DIR/screenshots" "$BASE_DIR/baselines" "$BASE_DIR/results"

log() {
    echo "[$TIMESTAMP] $1" | tee -a "$LOG_FILE"
}

# Check if HWC is running
if ! curl -sf http://localhost:3005/ > /dev/null 2>&1; then
    log "ERROR: HWC not running at localhost:3005"
    exit 1
fi

log "=== Starting Visual QA ==="

# Capture screenshot
SCREENSHOT="$BASE_DIR/screenshots/screenshot-$(date '+%Y%m%d_%H%M%S').png"

google-chrome-stable \
    --headless \
    --disable-gpu \
    --no-sandbox \
    --virtual-time-budget=10000 \
    --window-size=1440,900 \
    --screenshot="$SCREENSHOT" \
    --disable-web-security \
    http://localhost:3005 2>/dev/null

if [ ! -f "$SCREENSHOT" ]; then
    log "ERROR: Screenshot capture failed"
    exit 1
fi

SIZE=$(stat -c%s "$SCREENSHOT")
log "Screenshot saved: $SCREENSHOT (${SIZE} bytes)"

# Compare to baseline
BASELINE="$BASE_DIR/baselines/baseline-default.png"

if [ -f "$BASELINE" ]; then
    # Use ImageMagick for pixel comparison if available
    if command -v compare &> /dev/null; then
        DIFF="$BASE_DIR/results/diff-$(date '+%Y%m%d_%H%M%S').png"
        RESULT=$(compare -metric AE "$SCREENSHOT" "$BASELINE" "$DIFF" 2>&1)
        
        # RESULT is the number of differing pixels
        TOTAL_PIXELS=$((1440 * 900))
        DIFF_PERCENT=$(awk "BEGIN {printf \"%.2f\", ($RESULT / $TOTAL_PIXELS) * 100}")
        
        log "Pixel diff: $RESULT pixels ($DIFF_PERCENT%)"
        
        if [ "$DIFF_PERCENT" > "1.0" ]; then
            log "⚠ REGRESSION: Visual change detected ($DIFF_PERCENT%)"
            log "Diff image: $DIFF"
            
            # Copy current to latest for inspection
            cp "$SCREENSHOT" "$BASE_DIR/screenshots/latest.png"
        else
            log "✓ PASS: No significant visual regression"
        fi
    else
        # Fallback: size-based comparison
        BASELINE_SIZE=$(stat -c%s "$BASELINE")
        SIZE_DIFF=$(awk "BEGIN {printf \"%.2f\", abs($SIZE - $BASELINE_SIZE) / $BASELINE_SIZE * 100}")
        
        log "Size-based diff: $SIZE_DIFF% (baseline: $BASELINE_SIZE bytes)"
        
        if [ $(echo "$SIZE_DIFF > 5" | bc) -eq 1 ]; then
            log "⚠ Possible regression: size changed by $SIZE_DIFF%"
        else
            log "✓ PASS: Size within tolerance"
        fi
    fi
else
    log "INFO: No baseline found — storing this capture as baseline"
    cp "$SCREENSHOT" "$BASELINE"
fi

log "=== Visual QA Complete ==="
echo "" >> "$LOG_FILE"