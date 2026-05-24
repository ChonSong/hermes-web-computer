# Waybar Functional Specification

**Component:** Waybar (top bar), Workspaces, Dock (bottom bar), File Explorer panel, Bottom Panel  
**System:** Hyprland desktop environment — Illogical Impulse visual style  
**Document version:** 1.0

---

## 1. Overview

Waybar is the primary HUD layer for the Hyprland-based Hermes Web Computer desktop. It consists of:

1. **Top Bar (Waybar)** — workspace pill indicator + system tray
2. **Dock (Bottom Bar)** — floating centered app launcher
3. **Left Panel** — file explorer (VSCode-style sidebar)
4. **Bottom Panel** — terminal with tabbed sessions
5. **Workspace system** — 9 virtual workspaces mapped to dot indicators

The visual language is **Illogical Impulse** — frosted-glass floating panels with backdrop blur, purple accent highlights, and JetBrains Mono typography.

---

## 2. Top Bar (Waybar)

### 2.1 Position & Geometry

| Property | Value |
|----------|-------|
| Position | `fixed top-0 left-0 right-0` (full-width) |
| Height | `36px` |
| Background | Transparent (composited blur behind) |
| Z-index | `1000` |

### 2.2 Workspace Pill Indicator

The primary workspace UI is a **floating centered pill** that replaces a full-width bar.

| Property | Value |
|----------|-------|
| Container position | `fixed top-4 left-1/2 -translate-x-1/2` |
| Container z-index | `50` |
| Dimensions | `height: 36px`, padding `4px 20px` |
| Background | `rgba(18, 18, 26, 0.85)` |
| Border | `1px solid rgba(255, 255, 255, 0.08)` |
| Border-radius | `9999px` (full pill) |
| Backdrop-blur | `blur(20px)` |
| Box-shadow | `0 4px 16px rgba(0, 0, 0, 0.3)` |
| Displayed info | 9 workspace dots + separator + clock + agent status dot |

**Workspace Dots:**

| State | Appearance |
|-------|------------|
| Active (current WS) | `12px` circle, `bg-purple-400`, glow shadow `0 0 8px rgba(167, 139, 250, 0.6)` |
| Occupied (has windows) | `12px` circle, `bg-white/30`, no glow |
| Empty | `12px` circle, `bg-white/15`, no glow |
| Hover (any dot) | `bg-white/40` transition `150ms` |

**Separator:** `1px` wide × `16px` tall `rgba(255, 255, 255, 0.1)` vertical line between dots and clock.

**Clock:** Right side of pill, `text-[11px] font-mono text-white/60`, updates every second, format `HH:MM`.

**Agent Status Dot:**
- Idle: `w-2 h-2 rounded-full bg-gray-400`
- Processing/Thinking: `w-2 h-2 rounded-full bg-green-400 animate-pulse`
- Error: `w-2 h-2 rounded-full bg-red-400`

### 2.3 System Tray (Right-Aligned)

Located in the **top-right corner** of the screen, outside the workspace pill.

| Property | Value |
|----------|-------|
| Position | `fixed top-3 right-4` |
| Background | Same glass treatment as pill |
| Border-radius | `9999px` |
| Padding | `4px 12px` |
| Items | wifi icon, volume icon, battery icon, temperature icon, current time |

**Tray Item States:**
| Item | Normal | Muted/Disabled | Critical |
|------|--------|----------------|----------|
| Volume | `🔊` icon, white | `🔇` icon, white/40 | — |
| Wifi | `🌐` icon, white | `🌐` icon, red/40 | — |
| Battery | `🔋` icon, white/80 | — | `🪫` icon + pulsing red border |
| Temperature | `🌡️` icon, white/60 | — | `🔥` + red/60 above 80°C |

---

## 3. Dock (Bottom Bar)

### 3.1 Position & Geometry

| Property | Value |
|----------|-------|
| Position | `fixed bottom-3 left-1/2 -translate-x-1/2` |
| Z-index | `50` |
| Dimensions | auto-width, height `52px` |
| Background | `rgba(18, 18, 26, 0.7)` |
| Border | `1px solid rgba(255, 255, 255, 0.08)` |
| Border-radius | `9999px` (full pill) |
| Backdrop-blur | `blur(24px)` |
| Box-shadow | `0 8px 32px rgba(0, 0, 0, 0.4)` |
| Padding | `6px 16px` |

### 3.2 App Icons

Dock holds **11 app launchers** in a horizontal row:

| App | Icon | Launch command |
|-----|------|----------------|
| File Explorer | `📁` | Opens VSCode-style left panel |
| Terminal | `🖥️` | Opens terminal tile |
| Browser | `🌐` | Opens browser tile |
| Code Editor | `💻` | Opens VSCode tile |
| Chat | `💬` | Opens agent chat right panel |
| Settings | `⚙️` | Opens settings tile |
| Music | `🎵` | Opens music player tile |
| Calculator | `🔢` | Opens calculator |
| Calendar | `📅` | Opens calendar |
| Clock | `⏰` | Opens clock |
| Camera | `📷` | Opens camera |

**Icon States:**
| State | Behavior |
|-------|----------|
| Default | `32px × 32px` circle, `text-2xl`, no scale |
| Hover | `scale(1.1)`, `150ms ease-out` transition |
| Active (focused window) | Small `6px` purple dot (`#a78bfa`) centered below icon |
| Pressed | `scale(0.95)` briefly |

### 3.3 Dock Interaction

- **Click:** Launch app or focus existing window
- **Middle-click:** Launch new instance regardless of existing window
- **Right-click:** Context menu (placeholder for now)

---

## 4. Workspaces

### 4.1 Workspace Model

Hyprland supports **10 workspaces** (`0` through `9`); workspace `0` is typically the scratchpad.

| Workspace | Default purpose |
|----------|-----------------|
| `1` | Terminal / dev environment |
| `2` | Web browser |
| `3` | Code editor (VSCode) |
| `4` | Chat / agent |
| `5` | File explorer |
| `6-9` | General purpose |

### 4.2 Workspace Switching

- Keyboard: `Shift+1` through `Shift+9` switch to workspaces `1`–`9`
- Waybar dots: clicking a workspace dot switches to that workspace
- Animation: smooth transition `200ms ease-out`

### 4.3 Workspace Indicators in Waybar

The 9 workspace dots in the pill indicator show real-time state:
- Dot `i` filled = workspace `i` is active or occupied
- Active workspace dot glows purple (`#a78bfa` with `0 0 8px` shadow)
- Other occupied dots are `white/30`
- Empty dots are `white/15`

---

## 5. Left Panel — File Explorer

### 5.1 Position & Geometry

| Property | Value |
|----------|-------|
| Position | `fixed top-12 left-3 bottom-20` |
| Width | `240px` (default), resizable `160px`–`400px` |
| Background | `rgba(18, 18, 26, 0.8)` |
| Border | `1px solid rgba(255, 255, 255, 0.08)` |
| Border-radius | `16px` (rounded-2xl) |
| Backdrop-blur | `blur(24px)` |
| Box-shadow | `0 8px 32px rgba(0, 0, 0, 0.4)` |
| Z-index | `40` |

### 5.2 File Explorer Layout

```
┌─ File Explorer ───────────────┐
│ 📁   explorer                      │ ← Header (minimal, icon + title)
├─────────────────────────────────┤
│ ▼ 📁 src                         │ ← Folder (expanded, bold)
│   📄 app.ts                      │
│   📄 index.html                  │
│ ▼ 📁 components                  │ ← Folder (collapsed)
│ ▶ 📁 utils                       │ ← Folder (collapsed)
├─────────────────────────────────┤
│ 🖥️ Open Terminal    `Ctrl+``     │ ← Action bar at bottom
└─────────────────────────────────┘
```

### 5.3 File Entry States

| State | Appearance |
|-------|------------|
| Default | `text-gray-300 text-sm`, file icon |
| Hover | `bg-white/5` |
| Selected | `bg-purple-500/10 border-l-2 border-purple-400` |
| Folder (expanded) | `▼` chevron, folder icon filled |
| Folder (collapsed) | `▶` chevron, folder icon outline |
| Modified | `●` indicator dot `orange-400` |

### 5.4 Collapse Behavior

Left panel slides out to the left edge, leaving only a **3px accent line** (`border-l-2 border-purple-500/30`) visible. Clicking the accent line re-expands the panel `250ms ease-out`.

### 5.5 Context Menu (File Entry)

| Item | Action |
|------|--------|
| Open | Open file in editor |
| Open in Terminal | `cd` to file's directory in new terminal |
| Rename | Inline rename |
| Delete | Confirm dialog → delete |
| Copy Path | Copy absolute path to clipboard |

---

## 6. Bottom Panel — Terminal

### 6.1 Position & Geometry

| Property | Value |
|----------|-------|
| Position | `fixed bottom-3 right-3 left-3` (full-width minus margins) |
| Height | `240px` (default), resizable `120px`–`600px` |
| Background | `rgba(18, 18, 26, 0.85)` |
| Border | `1px solid rgba(255, 255, 255, 0.08)` |
| Border-radius | `16px` (top corners only) |
| Backdrop-blur | `blur(20px)` |
| Z-index | `30` |

### 6.2 Terminal Tabs

```
┌─ ● bash ── zsh ── node ──────────────────── [+] ─┐
│ ┌─ tab 1 ──┐ ┌─ tab 2 ──┐ ┌─ tab 3 ──┐           │
│ │          │ │          │ │          │           │
│ │ terminal │ │ terminal │ │ terminal │           │
│ │ content  │ │ content  │ │ content  │           │
│ └──────────┘ └──────────┘ └──────────┘           │
└───────────────────────────────────────────────────┘
```

**Tab Properties:**
| State | Appearance |
|-------|------------|
| Active tab | `bg-[#1e1e2e]` top bar, `border-t-2 border-purple-400` |
| Inactive tab | `bg-[#12121a]/60`, `text-gray-400` |
| Tab hover | `bg-[#1e1e2e]/50` |
| New tab button | `+` icon, `text-gray-500` hover `text-white` |

**Tab actions:**
- **Click tab:** Switch active terminal
- **Middle-click tab:** Close tab
- **Right-click tab:** Context menu (Rename, Duplicate, Close others)
- **`Ctrl+``** toggle bottom panel visibility

### 6.3 Terminal Content

- Background: fully transparent (tile handles the glass background)
- Font: `JetBrains Mono`, `13px`
- Prompt: custom prompt showing `user@machine`
- Scrollback: `10,000` lines
- Selection color: `bg-purple-500/30`

---

## 7. Right Panel — Agent Chat (Brief)

Included here because it shares the floating panel language.

| Property | Value |
|----------|-------|
| Position | `fixed top-12 right-3 bottom-20` |
| Width | `320px` |
| Background | Same as left panel |
| Z-index | `40` |

Message display: flat left-aligned text, user = `text-white`, agent = `text-purple-300`. Input at bottom: floating pill `rounded-full bg-white/5 border border-white/10`.

---

## 8. Shared Visual Language

### 8.1 Glass Panel Token Summary

```
Background:        rgba(18, 18, 26, 0.80–0.85)
Border:            1px solid rgba(255, 255, 255, 0.08)
Border-radius:     16px  (panels) / 9999px (pills/dots)
Backdrop-blur:     20–32px
Box-shadow:        0 8px 32px rgba(0, 0, 0, 0.4)
Accent color:      #a78bfa (purple-400)
Active border:     rgba(167, 139, 250, 0.5)
Glow shadow:       0 0 12px rgba(167, 139, 250, 0.3)
```

### 8.2 Typography

| Element | Font | Size | Color |
|---------|------|------|-------|
| Workspace clock | JetBrains Mono | `11px` | `white/60` |
| Panel headers | Inter | `12px` | `white/50` |
| File names | Inter | `13px` | `gray-300` |
| Terminal | JetBrains Mono | `13px` | `white/90` |
| Dock icons | Emoji / SVG | `20px` | inherit |

### 8.3 Resize Handles

- Between panels: `2px` wide, accent color `rgba(167, 139, 250, 0.4)`
- Cursor: `ew-resize`
- Drag animation: `200ms ease-out`

---

## 9. Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Shift+1–9` | Switch to workspace `1–9` |
| `Shift+←/→` | Move window to adjacent workspace |
| `Ctrl+K` | Command palette |
| `Ctrl+B` | Toggle file explorer left panel |
| `` Ctrl+` `` | Toggle terminal bottom panel |
| `Ctrl+,` | Open settings |
| `Alt+←/→` | Switch between panels |
| `Shift+Q` | Close focused tile |
| `Shift+F` | Fullscreen focused tile |
| `Shift+Space` | Float / tile toggle |

---

## 10. Component Inventory Summary

| Component | File | Status |
|-----------|------|--------|
| `WorkspacePill` | `WorkspacePill.svelte` | Implemented |
| `Dock` | `Dock.svelte` | Implemented |
| `LeftPanel` (file explorer) | `LeftPanel.svelte` | Implemented |
| `BottomPanel` (terminal) | `BottomPanel.svelte` | Implemented |
| `RightPanel` (agent chat) | `RightPanel.svelte` | Implemented |
| `ResizeHandle` | `ResizeHandle.svelte` | Implemented |
| `SystemTray` | `SystemTray.svelte` | Implemented |
| Glass CSS tokens | `glass.css` | Implemented |

---

## 11. Reference Screenshots

- **Hyprland reference:** `end-4/dots-hyprland` — Illogical Impulse style showing Waybar with system tray, workspace dots, VSCode file explorer, and terminal tabs
- **Our implementation:** Screenshot comparison via Playwright → vision model at `e2e/tests/visual/baseline/`

---

## 12. Open Questions / Deferred Items

| Item | Description |
|------|-------------|
| System tray real data | Wifi/battery/temperature currently show static icons — need real backend data via Hyprland IPC |
| Middle-click dock | Placeholder — context menu not wired |
| Terminal drag-to-resize | Implemented but not yet tied to session persistence |
| Agent chat voice button | Pulsing circle design ready but voice capture not implemented |
| Monaco editor theme | Custom theme matching Illogical Impulse palette not yet applied |