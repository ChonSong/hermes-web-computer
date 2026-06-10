#!/usr/bin/env bash
# setup-xpra.sh — Install & configure Xpra for HWC native GUI app tiles
#
# Run this ON THE HOST (EndeavourOS), NOT inside the Docker container.
# SSH to host, then: bash ~/.hermes/hermes-web-computer/scripts/setup-xpra.sh
#
# What this does:
#   1. Installs xpra (AUR) + Xvfb (if missing)
#   2. Creates runtime directories
#   3. Creates a systemd --user service for auto-start on boot
#   4. Starts the xpra server on display :10 with HTML5 web bridge on port 9453
#   5. Verifies it's running and the HWC backend can proxy it
#
# Usage:
#   bash setup-xpra.sh               # full install + start
#   bash setup-xpra.sh --status      # check if xpra is running
#   bash setup-xpra.sh --stop        # stop the xpra server
#   bash setup-xpra.sh --restart     # restart the xpra server
#   bash setup-xpra.sh --log         # tail the xpra log

set -euo pipefail

# ─── Config ───────────────────────────────────────────────────────────────
DISPLAY_NUM=10
DISPLAY=":${DISPLAY_NUM}"
XPRA_HTTP_PORT=9453                      # Xpra HTML5 web bridge port
HWC_PORT=3005                            # HWC Go backend port
HWC_REPO="$HOME/.hermes/hermes-web-computer"
SYSTEMD_SERVICE="hwc-xpra"
XPRA_LOG_DIR="$HOME/.local/share/xpra"
XPRA_SOCK_DIR="/run/user/$(id -u)/xpra"
XPRA_LOG="$XPRA_LOG_DIR/server-${DISPLAY_NUM}.log"

# ─── Colors ───────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()  { echo -e " ${GREEN}✓${NC} $1"; }
warn(){ echo -e " ${YELLOW}⚠${NC} $1"; }
fail(){ echo -e " ${RED}✗${NC} $1"; }

# ─── Helpers ──────────────────────────────────────────────────────────────

check_command() {
    if ! command -v "$1" &>/dev/null; then
        fail "$1 is not installed"
        return 1
    fi
    ok "$1 found at $(command -v $1)"
}

ensure_dirs() {
    mkdir -p "$XPRA_LOG_DIR" "$XPRA_SOCK_DIR" 2>/dev/null || true
}

# ─── Install ───────────────────────────────────────────────────────────────

install_xpra() {
    echo ""
    echo "═══ Installing Xpra ═══"

    # xpra is in the AUR — use yay (EndeavourOS default)
    if command -v yay &>/dev/null; then
        echo "Installing xpra from AUR via yay..."
        yay -S --noconfirm --needed xpra 2>&1 | tail -5
    elif command -v paru &>/dev/null; then
        echo "Installing xpra from AUR via paru..."
        paru -S --noconfirm --needed xpra 2>&1 | tail -5
    else
        # Fallback: manual AUR install
        warn "No AUR helper found — installing xpra manually from AUR"
        cd /tmp
        rm -rf xpra-bin
        git clone --depth=1 https://aur.archlinux.org/xpra-bin.git
        cd xpra-bin
        makepkg -si --noconfirm 2>&1 | tail -5
        cd "$OLDPWD"
    fi

    check_command xpra

    # Ensure Xvfb is available (used by xpra for headless rendering)
    if ! command -v Xvfb &>/dev/null; then
        echo "Installing xorg-server-xvfb..."
        sudo pacman -S --noconfirm xorg-server-xvfb 2>&1 | tail -3
    fi
    check_command Xvfb
}

# ─── Start Server ──────────────────────────────────────────────────────────

start_server() {
    echo ""
    echo "═══ Starting Xpra server on display ${DISPLAY} ═══"

    ensure_dirs

    # Check if already running
    if xpra list | grep -q "display ${DISPLAY}" 2>/dev/null; then
        ok "Xpra server already running on ${DISPLAY}"
        return 0
    fi

    # Kill any stale xpra on this display
    xpra stop "${DISPLAY}" 2>/dev/null || true
    sleep 1

    # Start xpra server with:
    #   --no-daemonize: systemd manages backgrounding
    #   --html=on: enables the HTML5 web client
    #   --bind-web=9453: web client listens on this port
    #   --start=xterm: launch a terminal on start (feels alive)
    #   --idle-timeout=0: never auto-shutdown
    #   --server-Idle-Timeout=0: same
    #   --exit-with-children: stop when last child exits
    #   --xvfb="...": use a virtual framebuffer at 1920x1080
    echo "Starting: xpra start ${DISPLAY} --bind-web=${XPRA_HTTP_PORT} --html=on ..."

    nohup xpra start "${DISPLAY}" \
        --daemon \
        --no-daemonize \
        --socket-dir="${XPRA_SOCK_DIR}" \
        --html=on \
        --bind-web="0.0.0.0:${XPRA_HTTP_PORT}" \
        --start=xterm \
        --idle-timeout=0 \
        --server-Idle-Timeout=0 \
        --exit-with-children \
        --xvfb="Xvfb ${DISPLAY} -screen 0 1920x1080x24 -ac" \
        > "${XPRA_LOG}" 2>&1 &

    # Wait for socket to appear
    XPRA_SOCK="${XPRA_SOCK_DIR}/${DISPLAY_NUM}"
    for i in $(seq 1 15); do
        if [ -S "$XPRA_SOCK" ] 2>/dev/null; then
            ok "Xpra socket ready at ${XPRA_SOCK}"
            break
        fi
        sleep 1
    done

    # Verify HTTP bridge
    sleep 2
    if curl -sf "http://localhost:${XPRA_HTTP_PORT}/" >/dev/null 2>&1; then
        ok "Xpra HTML5 bridge responding at http://localhost:${XPRA_HTTP_PORT}"
    else
        warn "Xpra HTTP bridge not responding yet — may need a moment to initialize"
        warn "Check: ${XPRA_LOG}"
    fi

    # Verify xpra is actually running
    if xpra list 2>/dev/null | grep -q "display ${DISPLAY}"; then
        ok "Xpra server confirmed running on ${DISPLAY}"
    else
        fail "Xpra server failed to start — check ${XPRA_LOG}"
        tail -20 "$XPRA_LOG" 2>/dev/null || true
        return 1
    fi
}

# ─── Stop Server ───────────────────────────────────────────────────────────

stop_server() {
    echo ""
    echo "═══ Stopping Xpra server ═══"
    if xpra stop "${DISPLAY}" 2>/dev/null; then
        ok "Xpra server stopped"
    else
        warn "No Xpra server running on ${DISPLAY}"
    fi
}

# ─── Systemd Service ──────────────────────────────────────────────────────

install_systemd_service() {
    echo ""
    echo "═══ Installing systemd user service ═══"

    mkdir -p "$HOME/.config/systemd/user"

    cat > "$HOME/.config/systemd/user/${SYSTEMD_SERVICE}.service" << 'SERVICEEOF'
[Unit]
Description=HWC Xpra Server — native GUI app escape hatch
Documentation=https://xpra.org
After=network.target

[Service]
Type=forking
ExecStartPre=/bin/bash -c 'mkdir -p /run/user/%U/xpra'
ExecStart=xpra start :10 --daemon --no-daemonize --socket-dir=/run/user/%U/xpra --html=on --bind-web=0.0.0.0:9453 --start=xterm --idle-timeout=0 --server-Idle-Timeout=0 --exit-with-children --xvfb="Xvfb :10 -screen 0 1920x1080x24 -ac"
ExecStop=xpra stop :10
ExecReload=xpra restart :10
Restart=on-failure
RestartSec=10
StandardOutput=append:%h/.local/share/xpra/server-10.log
StandardError=inherit

[Install]
WantedBy=default.target
SERVICEEOF

    systemctl --user daemon-reload

    if systemctl --user is-enabled "${SYSTEMD_SERVICE}" &>/dev/null; then
        ok "Systemd service already enabled"
    else
        systemctl --user enable "${SYSTEMD_SERVICE}"
        ok "Systemd service enabled (auto-start on login)"
    fi

    if systemctl --user is-active "${SYSTEMD_SERVICE}" &>/dev/null; then
        ok "Systemd service is running"
    else
        systemctl --user start "${SYSTEMD_SERVICE}"
        ok "Systemd service started"
    fi
}

# ─── Verify All ────────────────────────────────────────────────────────────

verify() {
    echo ""
    echo "═══ Verification ═══"

    # 1. xpra binary
    check_command xpra

    # 2. Xvfb
    check_command Xvfb

    # 3. Xpra server process
    if xpra list 2>/dev/null | grep -q "display ${DISPLAY}"; then
        ok "Xpra server running on ${DISPLAY}"
        xpra list 2>/dev/null
    else
        warn "Xpra server not running"
    fi

    # 4. HTTP bridge
    if curl -sf "http://localhost:${XPRA_HTTP_PORT}/" >/dev/null 2>&1; then
        ok "Xpra HTML5 at http://localhost:${XPRA_HTTP_PORT}"
    else
        warn "Xpra HTML5 not responding on port ${XPRA_HTTP_PORT}"
    fi

    # 5. HWC backend (needed to proxy xpra)
    if curl -sf "http://localhost:${HWC_PORT}/" >/dev/null 2>&1; then
        ok "HWC backend running on port ${HWC_PORT}"
    else
        warn "HWC backend not running on port ${HWC_PORT}"
        echo "  Start it: cd ${HWC_REPO}/backend && go build -o /tmp/hwc-server ./cmd/server/ && HERMES_HWC_ROOT=${HWC_REPO} nohup /tmp/hwc-server >/tmp/hwc-server.log 2>&1 &"
    fi

    # 6. Systemd service
    if systemctl --user is-enabled "${SYSTEMD_SERVICE}" &>/dev/null 2>&1; then
        ok "Systemd service ${SYSTEMD_SERVICE} is enabled"
    fi

    # 7. Socket
    if [ -S "${XPRA_SOCK_DIR}/${DISPLAY_NUM}" ] 2>/dev/null; then
        ok "Xpra socket at ${XPRA_SOCK_DIR}/${DISPLAY_NUM}"
    else
        warn "Xpra socket not found (may not be started yet)"
    fi
}

# ─── Status ────────────────────────────────────────────────────────────────

status() {
    echo "═══ Xpra Status ═══"
    echo ""

    echo "Binary: $(command -v xpra 2>/dev/null || echo 'NOT INSTALLED')"
    if command -v xpra &>/dev/null; then
        echo "Version: $(xpra --version 2>/dev/null || xpra version 2>/dev/null | head -1)"
    fi
    echo ""
    echo "Servers:"
    xpra list 2>/dev/null || echo "  (none running)"
    echo ""
    echo "HTTP bridge:"
    curl -sI "http://localhost:${XPRA_HTTP_PORT}/" 2>/dev/null | head -3 || echo "  Not responding on port ${XPRA_HTTP_PORT}"
    echo ""
    echo "Systemd service:"
    systemctl --user status "${SYSTEMD_SERVICE}" 2>/dev/null | grep -E "Loaded|Active|Process" || echo "  Service not installed"
    echo ""
    echo "Socket:"
    ls -la "${XPRA_SOCK_DIR}/${DISPLAY_NUM}" 2>/dev/null || echo "  No socket found"
    echo ""
    echo "Log: ${XPRA_LOG}"
    echo ""
    echo "Windows:"
    if command -v xpra &>/dev/null; then
        xpra list-windows --display="${DISPLAY}" 2>/dev/null || echo "  (server not running or no windows)"
    fi
}

# ─── Log ───────────────────────────────────────────────────────────────────

show_log() {
    if [ -f "$XPRA_LOG" ]; then
        tail -40 "$XPRA_LOG"
    else
        echo "No log file at $XPRA_LOG"
    fi
}

# ─── Quick Test ────────────────────────────────────────────────────────────

quick_test() {
    echo ""
    echo "═══ Quick Xpra Smoke Test ═══"

    # 1. Launch a window (xterm)
    echo "Starting xterm via xpra..."
    DISPLAY="${DISPLAY}" xterm -e "echo 'Xpra is working!' && sleep 3" &
    XPID=$!
    sleep 2

    # 2. Check that xpra sees it
    echo "Windows on ${DISPLAY}:"
    xpra list-windows --display="${DISPLAY}" 2>/dev/null || echo "  (no windows)"
    echo ""

    # 3. Open the HTML5 client in Chrome (optional)
    echo "To view this window in the browser:"
    echo "  google-chrome-stable http://localhost:${XPRA_HTTP_PORT}/"
    echo ""
    echo "Or open an HWC XpraTile at:"
    echo "  http://localhost:${HWC_PORT}/ (if HWC backend is running)"
    echo ""

    wait $XPID 2>/dev/null || true
    ok "Test complete"
}

# ─── Main ──────────────────────────────────────────────────────────────────

main() {
    echo "╔═══════════════════════════════════════════╗"
    echo "║    HWC Xpra Setup — EndeavourOS           ║"
    echo "╚═══════════════════════════════════════════╝"
    echo "Display: ${DISPLAY}  |  Web port: ${XPRA_HTTP_PORT}"

    case "${1:-install}" in
        install)
            install_xpra
            start_server
            install_systemd_service
            verify
            echo ""
            ok "Xpra setup complete"
            echo ""
            echo "Next steps:"
            echo "  1. Test with: bash $0 --test"
            echo "  2. Open http://localhost:${XPRA_HTTP_PORT} in a browser"
            echo "  3. Launch an XpraTile in HWC to use native GUI apps"
            echo "  4. Check status anytime: bash $0 --status"
            ;;
        --install)
            install_xpra
            ;;
        --start)
            start_server
            ;;
        --stop)
            stop_server
            ;;
        --restart)
            stop_server
            sleep 2
            start_server
            ;;
        --status)
            status
            ;;
        --log)
            show_log
            ;;
        --test)
            quick_test
            ;;
        --verify)
            verify
            ;;
        --service)
            install_systemd_service
            ;;
        --help|-h)
            echo ""
            echo "Usage: bash setup-xpra.sh [OPTION]"
            echo ""
            echo "  (no option)   Full install + start + systemd service"
            echo "  --install     Only install xpra from AUR"
            echo "  --start       Start xpra server on display :10"
            echo "  --stop        Stop xpra server"
            echo "  --restart     Restart xpra server"
            echo "  --status      Show xpra status"
            echo "  --log         Tail xpra server log"
            echo "  --test        Quick smoke test (launch xterm, check windows)"
            echo "  --verify      Verify all components"
            echo "  --service     Install/enable systemd user service"
            echo "  --help, -h    Show this help"
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: bash setup-xpra.sh [--install|--start|--stop|--restart|--status|--log|--test|--verify|--service|--help]"
            exit 1
            ;;
    esac
}

main "$@"
