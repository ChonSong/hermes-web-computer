/**
 * commands.svelte.ts — Slash command registry (/ prefix) for HWC.
 *
 * Built-in commands: /session, /model, /workspace, /skills, /clear, /help, /abort
 *
 * Usage:
 *   import { commandStore, parseCommand } from "../stores/commands"
 *   commandStore.autocomplete(filter) → Command[]
 *   commandStore.execute(cmd, args) → void
 */

import { send, on } from "./ws"
import { sessionStore } from "./sessions.svelte"
import { configStore } from "./config.svelte"
import { workspaceStore, setActiveWorkspace } from "./workspace"
import { skillsStore } from "./skills.svelte"
import { get as getStore } from "svelte/store"

// ─── Types ─────────────────────────────────────────────────────────────────────

export interface Command {
  id: string
  name: string
  description: string
  aliases?: string[]
  execute(args: string[]): void | Promise<void>
}

export interface ParsedCommand {
  cmd: string       // e.g. "session"
  args: string[]    // e.g. ["list"] from "/session list"
  raw: string       // original input
}

// ─── Command Registry ────────────────────────────────────────────────────────

class CommandStore {
  // Autocomplete state
  showAutocomplete = $state(false)
  selectedIndex = $state(0)
  filter = $state("")

  // Registered commands
  commands = $state<Command[]>([])

  // Derived: filtered commands for autocomplete
  filtered = $derived(this.commands.filter(c => {
    const f = this.filter.toLowerCase()
    if (!f) return true
    return c.name.toLowerCase().includes(f) ||
      c.aliases?.some(a => a.toLowerCase().includes(f)) ||
      c.id.toLowerCase().includes(f)
  }))

  constructor() {
    this._registerBuiltin()
  }

  // ── Parse raw input → {cmd, args, raw} ──────────────────────────────────

  parse(raw: string): ParsedCommand {
    const trimmed = raw.trim()
    if (!trimmed.startsWith("/")) return { cmd: "", args: [], raw }
    const parts = trimmed.slice(1).split(/\s+/)
    return {
      cmd: parts[0]?.toLowerCase() ?? "",
      args: parts.slice(1),
      raw,
    }
  }

  // ── Find command by name/alias ──────────────────────────────────────────

  find(name: string): Command | undefined {
    const n = name.toLowerCase()
    return this.commands.find(c =>
      c.name.toLowerCase() === n ||
      c.aliases?.some(a => a.toLowerCase() === n) ||
      c.id.toLowerCase() === n
    )
  }

  // ── Execute a command by name with args ─────────────────────────────────

  async execute(name: string, args: string[]): Promise<void> {
    const cmd = this.find(name)
    if (!cmd) {
      console.warn("[commands] Unknown command:", name)
      return
    }
    this.showAutocomplete = false
    this.filter = ""
    this.selectedIndex = 0
    await cmd.execute(args)
  }

  // ── Autocomplete helpers ─────────────────────────────────────────────────

  autocomplete(filter: string) {
    this.filter = filter
    this.selectedIndex = 0
    this.showAutocomplete = filter.startsWith("/")
  }

  selectNext() {
    if (this.filtered.length === 0) return
    this.selectedIndex = (this.selectedIndex + 1) % this.filtered.length
  }

  selectPrev() {
    if (this.filtered.length === 0) return
    this.selectedIndex = (this.selectedIndex - 1 + this.filtered.length) % this.filtered.length
  }

  dismiss() {
    this.showAutocomplete = false
    this.filter = ""
    this.selectedIndex = 0
  }

  // ── Register a command ──────────────────────────────────────────────────

  register(cmd: Command) {
    this.commands = [...this.commands, cmd]
  }

  // ── Built-in commands ───────────────────────────────────────────────────

  private _registerBuiltin() {
    const store = this

    // /help — list all commands
    this.register({
      id: "help",
      name: "help",
      description: "Show all available commands",
      aliases: ["?"],
      execute() {
        const lines = store.commands.map(c =>
          `  /${c.name}${c.aliases?.length ? ` (${c.aliases.map(a => "/"+a).join(", ")})` : ""} — ${c.description}`
        )
        const msg = `Available commands:\n${lines.join("\n")}`
        // Show as a system message in chat
        window.dispatchEvent(new CustomEvent("hwc-show-help", { detail: msg }))
      },
    })

    // /clear — clear chat messages
    this.register({
      id: "clear",
      name: "clear",
      description: "Clear current chat messages",
      aliases: ["cls"],
      execute() {
        window.dispatchEvent(new CustomEvent("hwc-clear-chat"))
      },
    })

    // /abort — interrupt the agent
    this.register({
      id: "abort",
      name: "abort",
      description: "Interrupt the current agent operation",
      aliases: ["stop", "cancel"],
      execute() {
        send({ protocol: "agent", method: "chat.abort" })
        window.dispatchEvent(new CustomEvent("hwc-agent-abort"))
      },
    })

    // /session — session management (list/create/switch)
    this.register({
      id: "session",
      name: "session",
      description: "Manage sessions: list, new, switch, delete",
      aliases: ["sess"],
      execute(args) {
        const sub = args[0]?.toLowerCase()
        if (sub === "list" || sub === "ls") {
          sessionStore.refresh()
          window.dispatchEvent(new CustomEvent("hwc-show-panel", { detail: "sessions" }))
        } else if (sub === "new" || sub === "create") {
          const workspace = args[1]
          const model = args[2]
          sessionStore.create(workspace, model)
        } else if (sub === "switch" || sub === "use") {
          const id = args[1]
          if (id) sessionStore.select(id)
        } else if (sub === "delete" || sub === "rm") {
          const id = args[1]
          if (id) sessionStore.delete(id)
        } else {
          // Default: list sessions
          sessionStore.refresh()
          window.dispatchEvent(new CustomEvent("hwc-show-panel", { detail: "sessions" }))
        }
      },
    })

    // /model — show/switch models
    this.register({
      id: "model",
      name: "model",
      description: "Show available models or switch model",
      aliases: ["m"],
      execute(args) {
        const sub = args[0]?.toLowerCase()
        if (sub === "list" || sub === "ls" || !sub) {
          configStore.loadModels()
          configStore.refresh()
          window.dispatchEvent(new CustomEvent("hwc-show-panel", { detail: "config" }))
        } else if (sub === "set" || sub === "switch") {
          const modelId = args[1]
          if (modelId) {
            configStore.setConfig("model.default", modelId)
          }
        } else {
          configStore.loadModels()
          window.dispatchEvent(new CustomEvent("hwc-show-panel", { detail: "config" }))
        }
      },
    })

    // /workspace — show/switch workspace
    this.register({
      id: "workspace",
      name: "workspace",
      description: "Show current workspace or switch (1-9)",
      aliases: ["ws"],
      execute(args) {
        const sub = args[0]?.toLowerCase()
        if (sub === "list" || sub === "ls") {
          // Just show the current workspace
          const ws = getStore(workspaceStore)
          window.dispatchEvent(new CustomEvent("hwc-show-message", {
            detail: `Current workspace: ${ws.activeWorkspace}`
          }))
        } else {
          const n = parseInt(args[0])
          if (!isNaN(n) && n >= 1 && n <= 9) {
            setActiveWorkspace(n)
          } else {
            const ws = getStore(workspaceStore)
            window.dispatchEvent(new CustomEvent("hwc-show-message", {
              detail: `Current workspace: ${ws.activeWorkspace}. Use /workspace 1-9 to switch.`
            }))
          }
        }
      },
    })

    // /skills — open skills panel
    this.register({
      id: "skills",
      name: "skills",
      description: "Open the skills panel",
      aliases: ["skill"],
      execute() {
        skillsStore.refresh()
        window.dispatchEvent(new CustomEvent("hwc-show-panel", { detail: "skills" }))
      },
    })
  }
}

export const commandStore = new CommandStore()

// ─── Helpers ────────────────────────────────────────────────────────────────

/**
 * Parse a raw input string and return structured command + args.
 * Returns {cmd: "", args: [], raw} if input doesn't start with /.
 */
export function parseCommand(input: string): ParsedCommand {
  return commandStore.parse(input)
}