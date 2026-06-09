#!/usr/bin/env bash
# visual-qa.sh — Capture screenshots of HWC for visual regression testing
# Run on host: bash /home/sean/.hermes/hermes-web-computer/scripts/visual-qa.sh

set -e

BASE="/tmp/hwc-qa"
SCREENS="$BASE/screenshots"
BASELINES="$BASE/baselines"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$SCREENS" "$BASELINES"

echo "=== HWC Visual QA ==="
echo "Timestamp: $TIMESTAMP"
echo "Output: $SCREENS"
echo ""

# Check if server is running
if ! curl -sf http://localhost:3005/ > /dev/null 2>&1; then
    echo "ERROR: HWC not running at http://localhost:3005"
    echo "Start it with: cd /home/sean/.hermes/hermes-web-computer/backend && HERMES_HWC_ROOT=/home/sean/.hermes/hermes-web-computer ./agent-os server --port 3005 &"
    exit 1
fi

echo "[1/3] Launching chromium..."
# Use google-chrome-stable (already installed)
google-chrome-stable \
    --headless \
    --disable-gpu \
    --no-sandbox \
    --virtual-time-budget=10000 \
    --window-size=1440,900 \
    --screenshot="$SCREENS/screenshot-$TIMESTAMP.png" \
    --disable-web-security \
    http://localhost:3005 2>/dev/null

if [ -f "$SCREENS/screenshot-$TIMESTAMP.png" ]; then
    SIZE=$(stat -c%s "$SCREENS/screenshot-$TIMESTAMP.png")
    echo "✓ Screenshot saved: screenshot-$TIMESTAMP.png (${SIZE} bytes)"
    
    # Copy to baseline if it doesn't exist
    if [ ! -f "$BASELINES/baseline-default.png" ]; then
        cp "$SCREENS/screenshot-$TIMESTAMP.png" "$BASELINES/baseline-default.png"
        echo "✓ Stored as baseline"
    fi
else
    echo "ERROR: Screenshot failed"
    exit 1
fi

echo ""
echo "[2/3] Capturing additional views..."

# Capture at different window sizes
for size in "1280,720" "1920,1080"; do
    W=$(echo $size | cut -d, -f1)
    H=$(echo $size | cut -d, -f2)
    google-chrome-stable \
        --headless \
        --disable-gpu \
        --no-sandbox \
        --virtual-time-budget=8000 \
        --window-size="$size" \
        --screenshot="$SCREENS/screenshot-${W}x${H}-$TIMESTAMP.png" \
        --disable-web-security \
        http://localhost:3005 2>/dev/null && \
    echo "✓ Captured ${W}x${H}" || echo "✗ Failed ${W}x${H}"
done

echo ""
echo "[3/3] Comparing to baseline..."
if [ -f "$BASELINES/baseline-default.png" ]; then
    if command -v convert &> /dev/null; then
        # ImageMagick comparison
        WIDTH=$(identify -format "%w" "$SCREENS/screenshot-$TIMESTAMP.png" 2>/dev/null || echo "0")
        HEIGHT=$(identify -format "%h" "$SCREENS/screenshot-$TIMESTAMP.png" 2>/dev/null || echo "0")
        BASE_W=$(identify -format "%w" "$BASELINES/baseline-default.png" 2>/dev/null || echo "0")
        BASE_H=$(identify -format "%h" "$BASELINES/baseline-default.png" 2>/dev/null || echo "0")
        
        if [ "$WIDTH" = "$BASE_W" ] && [ "$HEIGHT" = "$BASE_H" ]; then
            # Same dimensions - pixel diff
            convert "$SCREENS/screenshot-$TIMESTAMP.png" "$BASELINES/baseline-default.png" \
                -resize 400x225! \
                -compare -compose src "$SCREENS/diff-$TIMESTAMP.png" 2>/dev/null && \
            echo "✓ Diff saved to diff-$TIMESTAMP.png" || echo "Diff comparison unavailable"
        else
            echo "⚠ Dimensions changed: current ${WIDTH}x${HEIGHT} vs baseline ${BASE_W}x${BASE_H}"
        fi
    else
        echo "⚠ ImageMagick not installed — install with: yay -S imagemagick"
    fi
fi

echo ""
echo "=== Results ==="
ls -la "$SCREENS/"
echo ""
echo "Done. Screenshots in: $SCREENS"