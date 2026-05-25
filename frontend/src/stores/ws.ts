import { writable, type Writable, get } from "svelte/store"
import { layoutState } from "./layout.svelte"

export interface Envelope {
  protocol: "ui" | "agent" | "audio"
  method: string
  params?: Record<string, unknown>
  id: string
  ts: number
}

export interface Event {
  protocol: string
  event: string
  data?: Record<string, unknown>
  ts: number
}

export interface LayoutTree {
  id: string
  type: string
  content?: string
  direction?: string
  children?: LayoutTree[]
  path?: string
  pty_id?: string
  browser_id?: string
  size?: number
  http_url?: string
  display?: string
}

export interface LayoutOp {
  op: string
  target_id?: string
  direction?: string
  content?: string
  pty_id?: string
  browser_id?: string
  size?: number
}

export const ws = writable<{ connected: boolean; lastError: string | null }>({
  connected: false,
  lastError: null,
})

// Keep the writable store for backward compatibility, but sync to reactive state on every write
const layoutWritable = writable<{ tree: LayoutTree | null; version: number }>({
  tree: null,
  version: 0,
})

export const layout: Writable<{ tree: LayoutTree | null; version: number }> = {
  subscribe: layoutWritable.subscribe,
  set(value) {
    layoutWritable.set(value)
    // Dispatch custom event for Svelte 5 reactivity
    try { window.dispatchEvent(new CustomEvent('hwc-layout-update', { detail: value })) } catch(e) {}
  },
  update(fn) {
    layoutWritable.update(fn)
    const current = get(layoutWritable)
    try { window.dispatchEvent(new CustomEvent('hwc-layout-update', { detail: current })) } catch(e) {}
  },
}

export { layoutWritable }

export const focus = writable<string>("root")
export const ptyOutputs = writable<Map<string, string>>(new Map())

let layoutVersion = 0

// Apply a single layout operation to the tree
function applyLayoutOp(tree: LayoutTree, op: {op: string; target_id?: string; direction?: string; content?: string; pty_id?: string; browser_id?: string}): LayoutTree {
  if (op.op === 'split' && op.target_id && op.content) {
    // Find target node and split it
    const target = findNode(tree, op.target_id)
    if (target && target.type === 'leaf') {
      const direction = op.direction || 'h'
      const newChildren: LayoutTree[] = [
        { ...target },
        {
          id: target.id + '_right',
          type: 'leaf',
          content: op.content,
          pty_id: op.pty_id,
          browser_id: op.browser_id,
          size: 0.5,
        },
      ]
      // Update the target node to be a split
      return updateNode(tree, op.target_id, {
        type: 'split',
        direction,
        children: newChildren,
        size: undefined,
      })
    }
  }
  return tree
}

// Find a node in the tree by ID
function findNode(tree: LayoutTree, id: string): LayoutTree | null {
  if (tree.id === id) return tree
  if (tree.children) {
    for (const child of tree.children) {
      const found = findNode(child, id)
      if (found) return found
    }
  }
  return null
}

// Update a node in the tree (immutable)
function updateNode(tree: LayoutTree, id: string, updates: Partial<LayoutTree>): LayoutTree {
  if (tree.id === id) {
    return { ...tree, ...updates }
  }
  if (tree.children) {
    return {
      ...tree,
      children: tree.children.map(c => updateNode(c, id, updates)),
    }
  }
  return tree
}

let socket: WebSocket | null = null
const handlers: Map<string, Set<(data: unknown) => void>> = new Map()

let reqId = 0
function nextId(): string {
  return `req_${++reqId}`
}

export function connect(url: string = "ws://localhost:3005/ws") {
  if (socket?.readyState === WebSocket.OPEN) return

  socket = new WebSocket(url)

  socket.onopen = () => {
    ws.set({ connected: true, lastError: null })
  }

  socket.onmessage = (ev) => {
    const event: Event = JSON.parse(ev.data)
    // Capture ALL events for debugging - add to window for cross-module access
    const win = globalThis as typeof globalThis & { __wsEvents?: Event[] }
    if (!win.__wsEvents) win.__wsEvents = []
    win.__wsEvents.push(event)
    console.log('[WS] RECV event:', event.event, 'data:', JSON.stringify(event.data)?.substring(0, 200), '| total events:', win.__wsEvents.length)
    if (event.event === "layout.initial") {
      const newTree = event.data?.tree as LayoutTree
      const newVersion = (event.data?.layout_version as number) || 1
      layout.set({ tree: newTree, version: newVersion })
      layoutState.setLayout(newTree, newVersion)
      layoutVersion = newVersion
    } else if (event.event === "layout.delta") {
      const d = event.data as {layout_version?: number, tree?: LayoutTree} | null
      const newTree = d?.tree
      const newVersion = d?.layout_version || layoutVersion + 1
      if (newTree) {
        layout.set({ tree: newTree, version: newVersion })
        layoutState.setLayout(newTree, newVersion)
        layoutVersion = newVersion
      }
    }
    if (event.protocol === "agent" && event.event === "pty.output") {
      const data = event.data as { pty_id: string; data: string }
      ptyOutputs.update(map => {
        const prev = map.get(data.pty_id) || ""
        map.set(data.pty_id, prev + data.data)
        return map
      })
    }
    const eventHandlers = handlers.get(event.event)
    console.log('[WS] Looking for handler:', event.event, 'found:', eventHandlers?.size || 0)
    if (eventHandlers) {
      const handlersArr = Array.from(eventHandlers)
      for (const handler of handlersArr) {
        handler(event.data)
      }
    }
  }

  socket.onclose = () => {
    ws.set({ connected: false, lastError: "Connection closed" })
    setTimeout(() => connect(url), 2000)
  }

  socket.onerror = () => {
    ws.set({ connected: false, lastError: "Connection error" })
  }
}

export function send(env: Omit<Envelope, "id" | "ts">): string {
  const id = nextId()
  const full: Envelope = { ...env, id, ts: Date.now() }
  socket?.send(JSON.stringify(full))
  return id
}

export function sendOp(op: LayoutOp): string {
  return send({ protocol: "ui", method: "layout.update", params: op as any })
}

export function on(event: string, handler: (data: unknown) => void): () => void {
  if (!handlers.has(event)) {
    handlers.set(event, new Set())
  }
  handlers.get(event)!.add(handler)
  console.log(`[WS] Handler registered for event: ${event}, total handlers for this event: ${handlers.get(event)!.size}`)
  return () => {
    const set = handlers.get(event)
    if (set) {
      set.delete(handler)
      if (set.size === 0) {
        handlers.delete(event)
      }
    }
  }
}

// FS helpers
export function fsList(path: string): string {
  return send({ protocol: "ui", method: "fs.list", params: { path } })
}

export function fsRead(path: string): string {
  return send({ protocol: "ui", method: "fs.read", params: { path } })
}

export function fsWrite(path: string, content: string, encoding = "utf8"): string {
  return send({ protocol: "ui", method: "fs.write", params: { path, content, encoding } })
}

export function fsStat(path: string): string {
  return send({ protocol: "ui", method: "fs.stat", params: { path } })
}

export function fsRename(oldPath: string, newPath: string): string {
  return send({ protocol: "ui", method: "fs.rename", params: { old_path: oldPath, new_path: newPath } })
}

// App helpers
export function appsList(): string {
  return send({ protocol: "ui", method: "apps.list" })
}

export function appsLaunch(type: string, path?: string): string {
  const params: Record<string, string> = { type }
  if (path) params.path = path
  return send({ protocol: "ui", method: "apps.launch", params })
}

// Session helpers
export function sessionNew(workspace?: string, model?: string): string {
  return send({ protocol: "ui", method: "session.new", params: { workspace, model } })
}

export function sessionList(): string {
  return send({ protocol: "ui", method: "session.list" })
}

export function sessionGet(id: string): string {
  return send({ protocol: "ui", method: "session.get", params: { id } })
}

export function sessionDelete(id: string): string {
  return send({ protocol: "ui", method: "session.delete", params: { id } })
}

export function sessionUpdate(id: string, pinned?: boolean, title?: string): string {
  return send({ protocol: "ui", method: "session.update", params: { id, pinned, title } })
}

// Chat helpers
export function chatSend(message: string, sessionId?: string): string {
  const params: Record<string, string> = { message }
  if (sessionId) params.session_id = sessionId
  return send({ protocol: "agent", method: "chat.send", params })
}

// Audio helpers
export function audioStart(sessionId?: string): string {
  return send({ protocol: "audio", method: "audio.start", params: { session_id: sessionId } })
}

export function audioStop(): string {
  return send({ protocol: "audio", method: "audio.stop" })
}

export function audioStream(opusChunk: Uint8Array): string {
  // Convert Uint8Array to number[] for JSON serialization
  return send({ protocol: "audio", method: "audio.stream", params: { opus_chunk: Array.from(opusChunk) } })
}

// Dashboard helpers
export function dashStats(): string {
  return send({ protocol: "ui", method: "dashboard.stats" })
}

export function analyticsGet(days: number = 7): string {
  return send({ protocol: "ui", method: "analytics.get", params: { days } })
}

export function systemInfo(): string {
  return send({ protocol: "ui", method: "system.info" })
}

export function systemResources(): string {
  return send({ protocol: "ui", method: "system.resources" })
}

export function systemServices(): string {
  return send({ protocol: "ui", method: "system.services" })
}

export function observabilityStatus(): string {
  return send({ protocol: "ui", method: "observability.status" })
}

export function fsDelete(path: string): string {
  return send({ protocol: "ui", method: "fs.delete", params: { path } })
}

// App launch helpers for dashboard tile types
export function launchDashTile(type: string): string {
  return send({ protocol: "ui", method: "apps.launch", params: { type } })
}

// Profile helpers
export function profilesList(): string {
  return send({ protocol: "agent", method: "profiles.list" })
}

export function profilesGet(id: string): string {
  return send({ protocol: "agent", method: "profiles.get", params: { id } })
}

// Skills helpers
export function skillsList(): string {
  return send({ protocol: "agent", method: "skills.list" })
}

export function skillsContent(id: string): string {
  return send({ protocol: "agent", method: "skills.content", params: { id } })
}

// Crons helpers
export function cronsList(): string {
  return send({ protocol: "agent", method: "crons.list" })
}

export function cronsCreate(name: string, schedule: string, action: string, enabled = true): string {
  return send({ protocol: "agent", method: "crons.create", params: { name, schedule, action, enabled } })
}

export function cronsUpdate(id: string, name?: string, schedule?: string, action?: string, enabled?: boolean): string {
  return send({ protocol: "agent", method: "crons.update", params: { id, name, schedule, action, enabled } })
}

export function cronsDelete(id: string): string {
  return send({ protocol: "agent", method: "crons.delete", params: { id } })
}

export function cronsPause(id: string): string {
  return send({ protocol: "agent", method: "crons.pause", params: { id } })
}

export function cronsResume(id: string): string {
  return send({ protocol: "agent", method: "crons.resume", params: { id } })
}

export function cronsRun(id: string): string {
  return send({ protocol: "agent", method: "crons.run", params: { id } })
}

// Memory helpers
export function memoryRead(key: string): string {
  return send({ protocol: "agent", method: "memory.read", params: { key } })
}

export function memoryWrite(key: string, value: string): string {
  return send({ protocol: "agent", method: "memory.write", params: { key, value } })
}

// Config helpers
export function configGet(): string {
  return send({ protocol: "ui", method: "config.get" })
}

export function configSet(key: string, value: unknown): string {
  return send({ protocol: "ui", method: "config.set", params: { key, value } })
}

export function configDelete(key: string): string {
  return send({ protocol: "ui", method: "config.delete", params: { key } })
}

export function configList(): string {
  return send({ protocol: "ui", method: "config.list" })
}

// Env helpers
export function envList(): string {
  return send({ protocol: "ui", method: "env.list" })
}

export function envSet(key: string, value: string): string {
  return send({ protocol: "ui", method: "env.set", params: { key, value } })
}

export function envDelete(key: string): string {
  return send({ protocol: "ui", method: "env.delete", params: { key } })
}

// System helpers
export function systemRestart(): string {
  return send({ protocol: "ui", method: "system.restart" })
}

// Docker helpers
export function dockerList(): string {
  return send({ protocol: "ui", method: "docker.list" })
}

export function dockerContainerInfo(id: string): string {
  return send({ protocol: "ui", method: "docker.container_info", params: { id } })
}

export function dockerContainerLogs(id: string, tail?: number): string {
  return send({ protocol: "ui", method: "docker.container_logs", params: { id, tail: tail ?? 100 } })
}

export function dockerContainerStats(id: string): string {
  return send({ protocol: "ui", method: "docker.container_stats", params: { id } })
}

// MCP helpers
export function mcpList(): string {
  return send({ protocol: "agent", method: "mcp.list" })
}

export function mcpConnect(name: string, command: string, args?: string[]): string {
  return send({ protocol: "agent", method: "mcp.connect", params: { name, command, args: args ?? [] } })
}

export function mcpDisconnect(name: string): string {
  return send({ protocol: "agent", method: "mcp.disconnect", params: { name } })
}

export function mcpToolsList(serverName: string): string {
  return send({ protocol: "agent", method: "mcp.tools.list", params: { server_name: serverName } })
}

export function mcpToolsCall(serverName: string, toolName: string, toolArgs?: Record<string, unknown>): string {
  return send({ protocol: "agent", method: "mcp.tools.call", params: { server_name: serverName, tool_name: toolName, arguments: toolArgs ?? {} } })
}

export function mcpResourcesList(serverName: string): string {
  return send({ protocol: "agent", method: "mcp.resources.list", params: { server_name: serverName } })
}

export function mcpResourcesRead(serverName: string, uri: string): string {
  return send({ protocol: "agent", method: "mcp.resources.read", params: { server_name: serverName, uri } })
}

export function mcpPromptsList(serverName: string): string {
  return send({ protocol: "agent", method: "mcp.prompts.list", params: { server_name: serverName } })
}

export function mcpPromptsGet(serverName: string, name: string, promptArgs?: Record<string, unknown>): string {
  return send({ protocol: "agent", method: "mcp.prompts.get", params: { server_name: serverName, name, arguments: promptArgs ?? {} } })
}

// Don't auto-connect - let main.ts call connect()
