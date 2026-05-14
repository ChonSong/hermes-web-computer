/**
 * config.svelte.ts — Svelte 5 reactive config store
 * Handles Hermes config and environment variable management.
 */
import { send, on } from "./ws"

export interface HermesConfig {
  model: {
    base_url: string
    default: string
    provider: string
    api_key?: string
  }
  providers: Record<string, unknown>
  fallback_providers: string[]
  toolsets: string[]
  agent: {
    max_turns: number
    gateway_timeout: number
    restart_drain_timeout: number
    api_max_retries: number
    service_tier: string
    tool_use_enforcement: string
    gateway_timeout_warning: number
    gateway_notify_interval: number
    gateway_auto_continue_freshness: number
    image_input_mode: string
    disabled_toolsets: string[]
    personalities: Record<string, string>
    reasoning_effort: string
    verbose: boolean
  }
  terminal: Record<string, unknown>
  web: Record<string, unknown>
  browser: Record<string, unknown>
  env: {
    vars: Record<string, string>
  }
}

export interface ModelInfo {
  id: string
  name: string
  provider?: string
}

class ConfigState {
  config = $state<HermesConfig | null>(null)
  models = $state<ModelInfo[]>([])
  envVars = $state<Record<string, string>>({})
  loading = $state(false)
  error = $state<string | null>(null)
  restartStatus = $state<string | null>(null)

  async refresh(): Promise<void> {
    this.loading = true
    this.error = null
    return new Promise<void>((resolve) => {
      const cleanup = on("config.get", (data: any) => {
        cleanup()
        this.config = data as HermesConfig
        this.loading = false
        resolve()
      })
      const errCleanup = on("config.get.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        this.loading = false
        resolve()
      })
      send({ protocol: "ui", method: "config.get" })
      setTimeout(() => { this.loading = false; resolve() }, 5000)
    })
  }

  async loadModels(): Promise<void> {
    try {
      const res = await fetch("/v1/models")
      if (res.ok) {
        const data = await res.json()
        if (data.data && Array.isArray(data.data)) {
          this.models = data.data.map((m: any) => ({
            id: m.id || m.name,
            name: m.name || m.id,
            provider: m.provider
          }))
        }
      }
    } catch (e) {
      console.error("Failed to load models:", e)
    }
  }

  async setConfig(key: string, value: unknown): Promise<boolean> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("config.set.ok", (_data: any) => {
        cleanup()
        this.refresh()
        resolve(true)
      })
      const errCleanup = on("config.set.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "config.set", params: { key, value } })
    })
  }

  async deleteConfig(key: string): Promise<boolean> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("config.delete.ok", (_data: any) => {
        cleanup()
        this.refresh()
        resolve(true)
      })
      const errCleanup = on("config.delete.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "config.delete", params: { key } })
    })
  }

  async listEnv(): Promise<void> {
    this.error = null
    return new Promise<void>((resolve) => {
      const cleanup = on("env.list", (data: any) => {
        cleanup()
        this.envVars = (data as any)?.env ?? {}
        resolve()
      })
      const errCleanup = on("env.list.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve()
      })
      send({ protocol: "ui", method: "env.list" })
      setTimeout(() => resolve(), 5000)
    })
  }

  async setEnv(key: string, value: string): Promise<boolean> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("env.set.ok", (_data: any) => {
        cleanup()
        this.listEnv()
        resolve(true)
      })
      const errCleanup = on("env.set.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "env.set", params: { key, value } })
    })
  }

  async deleteEnv(key: string): Promise<boolean> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("env.delete.ok", (_data: any) => {
        cleanup()
        this.listEnv()
        resolve(true)
      })
      const errCleanup = on("env.delete.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "env.delete", params: { key } })
    })
  }

  async restart(): Promise<boolean> {
    this.restartStatus = null
    return new Promise((resolve) => {
      const cleanup = on("system.restart.ok", (_data: any) => {
        cleanup()
        this.restartStatus = "Restart signal sent"
        resolve(true)
      })
      const errCleanup = on("system.restart.error", (data: any) => {
        errCleanup()
        this.restartStatus = (data as any)?.message ?? "Restart failed"
        this.error = this.restartStatus
        resolve(false)
      })
      send({ protocol: "ui", method: "system.restart" })
    })
  }
}

export const configStore = new ConfigState()