/**
 * theme.svelte.ts — Theme system with 7 themes
 * Persists to localStorage. Applies via data-theme on <html>.
 */
export interface Theme {
  id: string
  name: string
  description: string
  colors: {
    primary: string
    accent: string
    bg: string
    surface: string
    text: string
  }
}

export const themes: Theme[] = [
  {
    id: "illogical-impulse",
    name: "Illogical Impulse",
    description: "Deep purple/violet — original HWC default",
    colors: { primary: "#a78bfa", accent: "#34d399", bg: "#0a0a0f", surface: "#16161e", text: "#ffffff" },
  },
  {
    id: "catppuccin",
    name: "Catppuccin Mocha",
    description: "Warm dark with mauve/pink accents",
    colors: { primary: "#cba6f7", accent: "#a6e3a1", bg: "#11111b", surface: "#1e1e2e", text: "#cdd6f4" },
  },
  {
    id: "nord",
    name: "Nord",
    description: "Blue-grey arctic theme",
    colors: { primary: "#88c0d0", accent: "#a3be8c", bg: "#2e3440", surface: "#3b4252", text: "#eceff4" },
  },
  {
    id: "dracula",
    name: "Dracula",
    description: "Dark purple-pink themed",
    colors: { primary: "#bd93f9", accent: "#50fa7b", bg: "#1e1f29", surface: "#282a36", text: "#f8f8f2" },
  },
  {
    id: "tokyo-night",
    name: "Tokyo Night",
    description: "Deep blue with cyan/pink accents",
    colors: { primary: "#7aa2f7", accent: "#73daca", bg: "#0f111b", surface: "#1a1b2e", text: "#c0caf5" },
  },
  {
    id: "everforest",
    name: "Everforest",
    description: "Warm green with earthy tones",
    colors: { primary: "#a7c080", accent: "#e69875", bg: "#1e2326", surface: "#2d353b", text: "#d3c6aa" },
  },
  {
    id: "monokai",
    name: "Monokai",
    description: "Warm dark with yellow/orange accents",
    colors: { primary: "#a6e22e", accent: "#fd971f", bg: "#171812", surface: "#23241e", text: "#f8f8f2" },
  },
]

const STORAGE_KEY = "hwc-theme"

function loadTheme(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) || "illogical-impulse"
  } catch {
    return "illogical-impulse"
  }
}

function saveTheme(id: string) {
  try {
    localStorage.setItem(STORAGE_KEY, id)
  } catch { /* noop */ }
}

function applyTheme(id: string) {
  document.documentElement.setAttribute("data-theme", id)
}

class ThemeState {
  currentId = $state(loadTheme())

  constructor() {
    $effect(() => {
      applyTheme(this.currentId)
      saveTheme(this.currentId)
    })
  }

  get current(): Theme {
    return themes.find(t => t.id === this.currentId) ?? themes[0]
  }

  setTheme(id: string) {
    if (themes.some(t => t.id === id)) {
      this.currentId = id
    }
  }

  cycle() {
    const idx = themes.findIndex(t => t.id === this.currentId)
    this.currentId = themes[(idx + 1) % themes.length].id
  }
}

export const themeStore = new ThemeState()
