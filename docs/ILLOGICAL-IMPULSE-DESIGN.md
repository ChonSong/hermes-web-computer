# Illogical Impulse — Visual Design Translation Plan

**Source:** end-4/dots-hyprland (14.4k stars) — "illogical-impulse" style
**Target:** hermes-web-computer (Svelte 5 + Tailwind, browser-based)

---

## Design DNA

| Property | Hyprland (Desktop) | Our Browser Translation |
|----------|-------------------|------------------------|
| **Background** | Abstract fluid wallpaper (deep purple/violet) | CSS radial gradient `#0a0a0f` → `#1a0a2e` + subtle animated grain overlay |
| **Transparency** | Blur-behind-windows (Gaussian 20-40px) | `backdrop-filter: blur(20px)` on panels, `bg-opacity-80` bases |
| **Border Radius** | 12-20px on everything | `rounded-2xl` (16px) on panels, `rounded-full` on dock pills |
| **Shadows** | Soft diffuse drop shadows | `shadow-[0_8px_32px_rgba(0,0,0,0.4)]` on floating elements |
| **Window Borders** | Thin 1px accent-colored border | `border border-white/10` + active `border-purple-500/50` |
| **Active Window** | Accent-colored border glow | `ring-2 ring-purple-500/30 ring-offset-0` |
| **Typography** | Inter (UI), JetBrains Mono (terminal) | Same — already using JetBrains Mono for terminal |
| **Accent Color** | Dynamic (changes with wallpaper) | Primary: `#a78bfa` (purple-400), secondary: `#34d399` (emerald-400) |
| **Text Colors** | White on dark, high contrast | `text-white` primary, `text-gray-400` secondary, `text-gray-500` tertiary |

---

## Component Mapping

### 1. Top Bar → Workspace Pill Indicator
**Hyprland:** Thin top bar with workspace pills + system tray
**Our translation:**
- Floating pill at top-center, NOT full-width bar
- `rounded-full` with `backdrop-blur-xl bg-white/5 border border-white/10`
- Workspace dots: small circles, active = purple fill + glow, inactive = white/20
- System info on right: time, network indicator, agent status dot (green=processing, gray=idle)
- Height: 32px, padding: 4px 16px

```html
<!-- Workspace pill -->
<div class="fixed top-2 left-1/2 -translate-x-1/2 z-50
  backdrop-blur-xl bg-white/5 border border-white/10 rounded-full
  px-4 py-1 flex items-center gap-3 shadow-[0_4px_16px_rgba(0,0,0,0.3)]">
  <!-- workspace dots -->
  <button class="w-2 h-2 rounded-full bg-purple-400 shadow-[0_0_8px_rgba(167,139,250,0.6)]" />
  <button class="w-2 h-2 rounded-full bg-white/20 hover:bg-white/40 transition-colors" />
  <!-- separator -->
  <div class="w-px h-3 bg-white/10" />
  <!-- system info -->
  <span class="text-[10px] text-white/60 font-mono">14:32</span>
  <div class="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
</div>
```

### 2. Left Panel → File Explorer
**Hyprland:** Floating translucent panel with blur
**Our translation:**
- Not a full-height solid panel — floating card with margins
- Top margin: 44px (below workspace pill), bottom: 8px
- Left margin: 8px, width: 240px
- `backdrop-blur-2xl bg-[#12121a]/80 border border-white/10 rounded-2xl`
- Header: minimal, just icon + label, no heavy borders
- File entries: hover = `bg-white/5`, selected = `bg-purple-500/10 border-l-2 border-purple-400`
- Collapse: slides out, leaving only a thin 3px accent edge

### 3. Middle Pane → Tiling Window Manager
**Hyprland:** Master/stack layout with smooth animations
**Our translation:**
- Panels float with 4px gaps between them (not touching)
- Each tile: `backdrop-blur-xl bg-[#12121a]/90 border border-white/10 rounded-2xl shadow-[0_8px_32px_rgba(0,0,0,0.3)]`
- Active tile gets `border-purple-500/50 ring-1 ring-purple-500/20`
- Layout gap: 4px → `gap-1` with parent `p-1`
- Transitions: `transition-all duration-200 ease-out` on resize
- Terminal: fully transparent bg, colored text, no visible border when focused (border on tile)
- Editor: Monaco with custom theme matching the palette

### 4. Right Panel → Agent Chat
**Hyprland:** Floating panel, same style as left
**Our translation:**
- Mirror of left panel but on right side
- Width: 320px (wider for chat)
- Messages: no bubbles, just left-aligned text with subtle color coding
  - User: `text-white`
  - Agent: `text-purple-300`
- Input area: floating pill at bottom of panel, `rounded-full bg-white/5 border border-white/10`
- Voice button: pulsing circle when recording, `bg-red-500/20 border border-red-500/30`
- Collapse: same slide-out behavior

### 5. Dock / Bottom Bar
**Hyprland:** Floating pill dock, centered at bottom
**Our translation:**
- `fixed bottom-3 left-1/2 -translate-x-1/2 z-50`
- `backdrop-blur-2xl bg-[#12121a]/70 border border-white/10 rounded-full`
- `px-4 py-2 flex items-center gap-3 shadow-[0_8px_32px_rgba(0,0,0,0.4)]`
- App icons: 32x32 circles with hover scale + glow
- Active app indicator: small dot below icon, purple
- Icons: `📁` `🖥️` `💬` `🌐` `🎤` `📊` — use simple emoji or SVG
- Hover: `scale-110` with `transition-transform duration-150`

### 6. Command Palette
**Hyprland:** Centered launcher popup, large, frosted glass
**Our translation:**
- Overlay: `fixed inset-0 bg-black/60 backdrop-blur-sm`
- Panel: centered, `w-[520px] max-h-[60vh]`
- `backdrop-blur-3xl bg-[#16161e]/95 border border-white/15 rounded-2xl`
- Input: borderless, `text-xl text-white placeholder-white/30`
- Results: `text-sm`, hover = `bg-white/5 rounded-lg`, selected = `bg-purple-500/15`
- Keyboard hint badges: `bg-white/10 text-white/50 text-[10px] px-1.5 py-0.5 rounded`

### 7. Notifications / Toasts
**Hyprland:** Floating notification cards
**Our translation:**
- Top-right, stacked, `w-80`
- `backdrop-blur-2xl bg-[#16161e]/90 border border-white/10 rounded-xl`
- `p-3 shadow-[0_8px_24px_rgba(0,0,0,0.4)]`
- Slide in from right with `animate-[slideIn_0.2s_ease-out]`

---

## Color Palette

```css
/* Illogical Impulse — Browser Translation */
:root {
  --bg-primary: #0a0a0f;
  --bg-panel: rgba(18, 18, 26, 0.85);
  --bg-panel-hover: rgba(30, 30, 42, 0.6);
  --bg-dock: rgba(18, 18, 26, 0.7);
  --bg-input: rgba(255, 255, 255, 0.05);

  --border-subtle: rgba(255, 255, 255, 0.08);
  --border-default: rgba(255, 255, 255, 0.1);
  --border-active: rgba(167, 139, 250, 0.5);
  --border-glow: rgba(167, 139, 250, 0.2);

  --accent-primary: #a78bfa;   /* purple-400 */
  --accent-secondary: #34d399; /* emerald-400 */
  --accent-warm: #fb923c;      /* orange-400 */
  --accent-danger: #f87171;    /* red-400 */

  --text-primary: #ffffff;
  --text-secondary: rgba(255, 255, 255, 0.6);
  --text-tertiary: rgba(255, 255, 255, 0.35);
  --text-muted: rgba(255, 255, 255, 0.2);

  --shadow-panel: 0 8px 32px rgba(0, 0, 0, 0.4);
  --shadow-float: 0 4px 16px rgba(0, 0, 0, 0.3);
  --glow-active: 0 0 12px rgba(167, 139, 250, 0.3);

  --radius-sm: 8px;
  --radius-md: 12px;
  --radius-lg: 16px;
  --radius-xl: 20px;
  --radius-full: 9999px;

  --blur-sm: 8px;
  --blur-md: 16px;
  --blur-lg: 24px;
  --blur-xl: 32px;
}
```

---

## Tailwind Config Additions

```js
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        panel: {
          DEFAULT: 'rgba(18, 18, 26, 0.85)',
          hover: 'rgba(30, 30, 42, 0.6)',
        },
        glass: {
          border: 'rgba(255, 255, 255, 0.1)',
          'border-active': 'rgba(167, 139, 250, 0.5)',
        },
      },
      backdropBlur: {
        xs: '4px',
      },
      boxShadow: {
        panel: '0 8px 32px rgba(0, 0, 0, 0.4)',
        float: '0 4px 16px rgba(0, 0, 0, 0.3)',
        glow: '0 0 12px rgba(167, 139, 250, 0.3)',
      },
      borderRadius: {
        panel: '16px',
      },
      animation: {
        'slide-in': 'slideIn 0.2s ease-out',
        'fade-in': 'fadeIn 0.15s ease-out',
        'pulse-glow': 'pulseGlow 2s ease-in-out infinite',
      },
      keyframes: {
        slideIn: {
          '0%': { transform: 'translateX(100%)', opacity: '0' },
          '100%': { transform: 'translateX(0)', opacity: '1' },
        },
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        pulseGlow: {
          '0%, 100%': { boxShadow: '0 0 8px rgba(167, 139, 250, 0.2)' },
          '50%': { boxShadow: '0 0 16px rgba(167, 139, 250, 0.4)' },
        },
      },
    },
  },
}
```

---

## Implementation Phases

### Phase A: Foundation (2-3h)
1. Update `tailwind.config.js` with custom colors, shadows, animations
2. Create `src/styles/glass.css` — CSS custom properties + base styles
3. Create background gradient layer in `App.svelte`
4. Update all existing components to use new color tokens instead of hardcoded gray-900/etc

### Phase B: Top Pill + Dock (1-2h)
1. `WorkspacePill.svelte` — top-center floating pill with workspace dots
2. `Dock.svelte` — bottom-centered floating dock with app icons
3. Wire dock to existing `apps.launch` handler
4. Add hover/glow animations

### Phase C: Floating Panels (2-3h)
1. Refactor `Tile.svelte` — add glass panel styling, gaps, active states
2. Refactor `LeftPanel.svelte` — floating card with slide-out collapse
3. Refactor `RightPanel.svelte` — matching floating card
4. Update resize handles — thin accent lines instead of thick bars
5. Add `transition-all duration-200 ease-out` to all panel movements

### Phase D: Command Palette Redesign (1h)
1. Frosted glass overlay
2. Larger, centered popup
3. Keyboard hint badges
4. Smooth enter/exit animation

### Phase E: Agent Chat Polish (1-2h)
1. Remove message bubbles — flat text with color coding
2. Floating pill input at bottom
3. Voice button redesign (pulsing circle)
4. Typing indicator — subtle purple dots

### Phase F: Polish & Micro-interactions (2-3h)
1. Window open/close animations
2. Active window glow transitions
3. File tree hover effects with accent left-border
4. Terminal: transparent background, custom color scheme
5. Editor: Monaco theme matching palette
6. Scrollbar styling (thin, translucent)
7. Focus ring for keyboard navigation

---

## Keyboard Shortcut Map (Visual)

| Shortcut | Action | Visual Feedback |
|----------|--------|----------------|
| `Shift+1-9` | Switch workspace | Workspace pill dot fills with glow |
| `Shift+Arrow` | Focus adjacent tile | Border glow moves to new tile |
| `Ctrl+K` | Command palette | Frosted glass overlay fades in |
| `Shift+D` | Cycle layout | Brief layout name flash at top |
| `Shift+F` | Toggle fullscreen | Tile expands with smooth animation |
| `Shift+Q` | Close tile | Tile shrinks + fades out |
| `Shift+Space` | Float/toggle | Tile lifts with shadow increase |
| `Shift+Alt+Arrow` | Resize | Visible border stretches in direction |

---

## What We Skip (Browser Limitations)

- ❌ True blur-behind (CSS `backdrop-filter` only blurs elements behind, not desktop wallpaper)
- ❌ System tray integration
- ❌ Global hotkeys (only works within browser)
- ❌ Window snapping animations at OS level
- ❌ True transparency to desktop

## What We Gain (Browser Advantages)

- ✅ Smooth CSS transitions/animations
- ✅ WebSocket real-time updates
- ✅ Cross-platform (works on any device with a browser)
- ✅ No Wayland/X11 dependency
