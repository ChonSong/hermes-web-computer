<script lang="ts">
  /**
   * ChatPanel — Session chat UI rendered in a tile.
   * Reads streaming state from sessionStore, sends via sessionStore.send().
   * Phase 4: Research cards (URL + JSON), connection status, message search.
   */
  import { tick } from "svelte"
  import { sessionStore } from "../stores/sessions.svelte"
  import { commandStore, parseCommand } from "../stores/commands.svelte"
  import type { SessionMessage } from "../stores/sessions.svelte"
  import FileUpload from "./FileUpload.svelte"
  import ResearchCard from "./ResearchCard.svelte"

  let messagesEl: HTMLDivElement
  let inputValue = $state("")
  let errorMsg = $state<string | null>(null)

  // File upload state
  let fileUpload: ReturnType<typeof FileUpload> | null = $state(null)
  let uploadedFiles = $state<Array<{ path: string; name: string }>>([])

  // Autocomplete state
  let showAutocomplete = $state(false)
  let selectedIndex = $state(0)
  let inputEl: HTMLTextAreaElement | null = $state(null)

  // Active session
  let activeId = $derived(sessionStore.activeId)
  let activeSession = $derived(sessionStore.activeSession)

  // Context meter — token estimation from message content lengths
  const CONTEXT_WINDOW = 200_000

  let charCount = $derived.by(() => {
    return (activeSession?.messages ?? []).reduce((sum, msg) => {
      const content = typeof msg.content === "string" ? msg.content : JSON.stringify(msg.content)
      return sum + content.length
    }, 0)
  })

  let tokenEstimate = $derived(Math.floor(charCount / 4))
  let tokenPercent = $derived(Math.min((tokenEstimate / CONTEXT_WINDOW) * 100, 100))
  let tokenPercentDisplay = $derived(tokenPercent.toFixed(1))

  function contextColor(pct: number): string {
    if (pct >= 85) return "#ef4444"
    if (pct >= 60) return "#eab308"
    return "#22c55e"
  }

  function formatTokens(t: number): string {
    if (t >= 1000) return (t / 1000).toFixed(1) + "K"
    return String(t)
  }

  // Full message list (persisted messages + current streaming buffer)
  let allMessages = $derived.by(() => {
    const msgs = [...(activeSession?.messages ?? [])]
    const { text, toolCalls } = sessionStore.getBuf(activeId ?? "")
    if (text || toolCalls.length > 0) {
      msgs.push({
        role: "assistant",
        content: text,
        tool_calls: toolCalls.map((tc: any) => ({
          id: tc.id ?? "",
          type: "function" as const,
          function: {
            name: tc.name ?? "",
            arguments: tc.arguments ?? {},
          },
        })),
      } as SessionMessage)
    }
    return msgs
  })

  // Auto-scroll on new messages or buffer changes
  $effect(() => {
    allMessages.length
    tick().then(() => {
      messagesEl?.scrollTo({ top: messagesEl.scrollHeight, behavior: "smooth" })
    })
  })

  function renderContent(msg: SessionMessage): string {
    if (typeof msg.content === "string") return msg.content
    return JSON.stringify(msg.content)
  }

  async function handleSend() {
    const text = inputValue.trim()
    if (!text || !activeId) return
    inputValue = ""
    errorMsg = null
    try {
      await sessionStore.send(text)
    } catch (e) {
      errorMsg = String(e)
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    // Arrow navigation for autocomplete
    if (showAutocomplete && commandStore.filtered.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault()
        commandStore.selectNext()
        return
      }
      if (e.key === "ArrowUp") {
        e.preventDefault()
        commandStore.selectPrev()
        return
      }
      if (e.key === "Escape") {
        e.preventDefault()
        commandStore.dismiss()
        showAutocomplete = false
        return
      }
    }

    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()

      // If autocomplete is open, execute selected command
      if (showAutocomplete && commandStore.filtered.length > 0) {
        const cmd = commandStore.filtered[selectedIndex]
        if (cmd) {
          const parsed = parseCommand("/" + cmd.name)
          commandStore.execute(cmd.name, parsed.args)
          inputValue = ""
          showAutocomplete = false
          return
        }
      }

      handleSend()
    }
  }

  // Detect / prefix for autocomplete
  function handleInput() {
    const val = inputValue
    if (val.startsWith("/")) {
      commandStore.autocomplete(val)
      showAutocomplete = true
    } else {
      commandStore.dismiss()
      showAutocomplete = false
    }
  }

  function getToolCalls(msg: SessionMessage) {
    if (!msg.tool_calls) return []
    return msg.tool_calls.map(tc => ({
      id: tc.id,
      name: tc.function?.name ?? "",
      args: typeof tc.function?.arguments === "string"
        ? tc.function.arguments
        : JSON.stringify(tc.function?.arguments ?? {}, null, 2),
    }))
  }

  function formatTime(ts: number): string {
    return new Date(ts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
  }

  function handleUploadComplete(path: string, name: string) {
    uploadedFiles = [...uploadedFiles, { path, name }]
  }

  function handleUploadRemove(id: string) {
    uploadedFiles = uploadedFiles.filter((_, i) => i !== Number(id.split("_")[1]))
  }

  // ============================================================
  // 4.1 Research Cards — URL detection + JSON data tables
  // ============================================================

  const URL_REGEX = /https?:\/\/[^\s]+/g

  interface UrlCard {
    url: string
    title: string
    description: string
  }

  function extractUrls(text: string): UrlCard[] {
    const matches = text.match(URL_REGEX) ?? []
    return matches.map(url => ({
      url,
      title: url.length > 60 ? url.substring(0, 60) + "…" : url,
      description: "",
    }))
  }

  function tryParseJson(text: string): unknown | null {
    const trimmed = text.trim()
    if ((trimmed.startsWith("{") && trimmed.endsWith("}")) ||
        (trimmed.startsWith("[") && trimmed.endsWith("]"))) {
      try {
        return JSON.parse(trimmed)
      } catch {
        return null
      }
    }
    return null
  }

  function isJsonTable(data: unknown): boolean {
    if (!Array.isArray(data)) return false
    return data.length > 0 && typeof data[0] === "object" && data[0] !== null
  }

  function getJsonHeaders(data: unknown[]): string[] {
    if (!data.length) return []
    return Object.keys(data[0] as Record<string, unknown>)
  }

  type MessageSegment =
    | { type: "text"; content: string }
    | { type: "url"; url: string; title: string }
    | { type: "json-table"; data: unknown[]; headers: string[] }

  function renderMessageWithCards(text: string): MessageSegment[] {
    if (!text) return [{ type: "text", content: "" }]

    // Full-text JSON table?
    const fullJson = tryParseJson(text)
    if (fullJson !== null && Array.isArray(fullJson) && isJsonTable(fullJson)) {
      return [{ type: "json-table", data: fullJson as unknown[], headers: getJsonHeaders(fullJson as unknown[]) }]
    }

    // Split text by URLs and collect segments
    const parts = text.split(URL_REGEX)
    if (parts.length === 1) {
      // No URLs — return as plain text
      return [{ type: "text", content: text }]
    }

    const segments: MessageSegment[] = []
    let match: RegExpExecArray | null
    // Reset lastIndex before iterating
    URL_REGEX.lastIndex = 0
    while ((match = URL_REGEX.exec(text)) !== null) {
      const before = text.slice(URL_REGEX.lastIndex - match[0].length, URL_REGEX.lastIndex - match[0].length)
      if (before) segments.push({ type: "text", content: before })
      const url = match[0].trim()
      if (url) segments.push({ type: "url", url, title: getTitle(url) })
    }
    // trailing text after last URL
    const lastIdx = URL_REGEX.lastIndex
    if (lastIdx < text.length) {
      segments.push({ type: "text", content: text.slice(lastIdx) })
    }

    return segments
  }

  function getTitle(url: string): string {
    try {
      const u = new URL(url)
      const path = u.pathname.replace(/\/$/, "")
      const parts = path.split("/").filter(Boolean)
      if (parts.length > 0) {
        return decodeURIComponent(parts[parts.length - 1].replace(/[-_]/g, " ")) || u.hostname
      }
      return u.hostname
    } catch {
      return url.length > 50 ? url.substring(0, 50) + "…" : url
    }
  }

  // ============================================================
  // 4.3 Message Search
  // ============================================================

  let showSearch = $state(false)
  let searchQuery = $state("")
  let searchInputEl: HTMLInputElement | null = $state(null)
  let currentMatchIndex = $state(0)

  // Collect all message indexes that match the query
  let matchingMessageIndexes = $derived.by(() => {
    if (!searchQuery.trim()) return []
    const q = searchQuery.toLowerCase()
    return allMessages
      .map((msg, i) => ({ msg, i }))
      .filter(({ msg }) => {
        const content = typeof msg.content === "string" ? msg.content : JSON.stringify(msg.content)
        return content.toLowerCase().includes(q)
      })
      .map(({ i }) => i)
  })

  let matchCount = $derived(matchingMessageIndexes.length)
  let matchLabel = $derived(
    matchCount === 0
      ? "No matches"
      : matchCount === 1
        ? "1 match"
        : `${currentMatchIndex + 1} of ${matchCount}`
  )

  // Navigate to a specific match (scroll into view)
  function navigateToMatch(idx: number) {
    const targetIdx = matchingMessageIndexes[idx]
    if (targetIdx === undefined) return
    currentMatchIndex = idx
    tick().then(() => {
      const container = messagesEl
      if (!container) return
      const items = container.querySelectorAll<HTMLElement>("[data-message-index]")
      const el = items[targetIdx]
      if (el) el.scrollIntoView({ behavior: "smooth", block: "center" })
    })
  }

  function handleSearchKeydown(e: KeyboardEvent) {
    if (e.key === "ArrowDown") {
      e.preventDefault()
      if (matchCount > 0) {
        navigateToMatch((currentMatchIndex + 1) % matchCount)
      }
      return
    }
    if (e.key === "ArrowUp") {
      e.preventDefault()
      if (matchCount > 0) {
        navigateToMatch((currentMatchIndex - 1 + matchCount) % matchCount)
      }
      return
    }
    if (e.key === "Enter") {
      e.preventDefault()
      if (matchCount > 0) {
        navigateToMatch((currentMatchIndex + 1) % matchCount)
      }
      return
    }
    if (e.key === "Escape") {
      showSearch = false
      searchQuery = ""
      return
    }
  }

  function openSearch() {
    showSearch = true
    tick().then(() => searchInputEl?.focus())
  }

  function closeSearch() {
    showSearch = false
    searchQuery = ""
    currentMatchIndex = 0
  }

  // Highlight matched text in a string
  function highlightMatch(text: string, query: string): string {
    if (!query.trim()) return escapeHtml(text)
    const escaped = escapeHtml(text)
    const q = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
    return escaped.replace(new RegExp(q, "gi"), match => `<mark class="bg-yellow-500 text-black rounded px-0.5">${match}</mark>`)
  }

  function escapeHtml(s: string): string {
    return s
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
  }
</script>

<div
  class="flex flex-col h-full bg-gray-900 text-gray-100 text-sm relative"
  ondragover={(e) => fileUpload?.handleDragOver(e)}
  ondragleave={(e) => fileUpload?.handleDragLeave(e)}
  ondrop={(e) => fileUpload?.handleDrop(e)}
  role="application"
  aria-label="Chat panel with file drop support"
>
  <!-- Drop zone overlay -->
  <FileUpload
    bind:this={fileUpload}
    onUploadComplete={handleUploadComplete}
    onUploadRemove={handleUploadRemove}
  />

  <!-- Header -->
  <div class="flex-none px-4 py-3 border-b border-white/10">
    <div class="flex items-center justify-between gap-3">
      <div class="text-xs text-gray-400">
        {activeSession?.title ?? "New Chat"} &middot; {activeSession?.model ?? "agent"}
      </div>

      <!-- Search icon -->
      <button
        type="button"
        onclick={openSearch}
        class="p-1.5 rounded hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
        title="Search messages (Ctrl+F)"
        aria-label="Search messages"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <circle cx="11" cy="11" r="7" stroke-width="2"/>
          <path stroke-width="2" d="M16 16l4 4" stroke-linecap="round"/>
        </svg>
      </button>
    </div>

    <!-- Context meter -->
    <div
      class="flex items-center gap-2 px-2 py-1 rounded text-xs font-mono mt-2"
      style="border: 1px solid {contextColor(tokenPercent)}40; background: #191919;"
      title={tokenPercent >= 85 ? "Context usage >85%. Consider compressing or starting a new session." : `Context: ${tokenPercentDisplay}% used`}
    >
      <span class="text-gray-400">{formatTokens(tokenEstimate)}</span>
      <div class="w-24 h-2 rounded bg-gray-800 overflow-hidden">
        <div
          class="h-full rounded transition-all duration-300"
          style="width: {tokenPercent}%; background-color: {contextColor(tokenPercent)};"
        ></div>
      </div>
      <span class="text-gray-500">{CONTEXT_WINDOW / 1000}K</span>
    </div>
  </div>

  <!-- Search overlay -->
  {#if showSearch}
    <div class="flex-none px-4 py-2 bg-[#191919] border-b border-white/10 flex items-center gap-3">
      <svg class="w-4 h-4 text-gray-400 flex-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <circle cx="11" cy="11" r="7" stroke-width="2"/>
        <path stroke-width="2" d="M16 16l4 4" stroke-linecap="round"/>
      </svg>
      <input
        bind:this={searchInputEl}
        bind:value={searchQuery}
        onkeydown={handleSearchKeydown}
        placeholder="Search messages…"
        class="flex-1 bg-transparent text-sm text-white placeholder-gray-500 focus:outline-none"
        type="text"
      />
      <span class="text-xs text-gray-400 font-mono flex-none">{matchLabel}</span>
      <button
        type="button"
        onclick={closeSearch}
        class="p-1 rounded hover:bg-white/10 text-gray-400 hover:text-white flex-none"
        aria-label="Close search"
      >
        ✕
      </button>
    </div>
  {/if}

  <!-- Messages -->
  <div bind:this={messagesEl} class="flex-1 overflow-y-auto px-4 py-3 space-y-4">
    {#if allMessages.length === 0}
      <div class="text-gray-500 text-center mt-8 text-xs">
        Send a message to start the conversation.
      </div>
    {/if}

    {#each allMessages as msg, i}
      {#if msg.role === "user"}
        <div class="flex justify-end" data-message-index={i}>
          <div class="max-w-[75%] rounded-lg px-3 py-2 bg-blue-600 text-white text-sm">
            {#if showSearch && searchQuery}
              {@html highlightMatch(renderContent(msg), searchQuery)}
            {:else}
              {renderContent(msg)}
            {/if}
          </div>
        </div>
      {:else if msg.role === "assistant"}
        <div class="flex flex-col" data-message-index={i}>
          {#if msg.tool_calls && msg.tool_calls.length > 0}
            <div class="space-y-2 mb-1">
              {#each getToolCalls(msg) as tc}
                <div class="rounded border border-amber-500/40 bg-amber-950/30 px-3 py-2 text-xs font-mono">
                  <div class="text-amber-400 font-semibold mb-1">◆ {tc.name}</div>
                  <pre class="text-gray-300 whitespace-pre-wrap break-all">{tc.args}</pre>
                </div>
              {/each}
            </div>
          {/if}
          {#if msg.content}
            {@const segments = renderMessageWithCards(msg.content as string)}
            {#each segments as seg}
              {#if seg.type === "text"}
                <div class="text-gray-100 text-sm whitespace-pre-wrap">
                  {#if showSearch && searchQuery}
                    {@html highlightMatch(seg.content, searchQuery)}
                  {:else}
                    {seg.content}
                  {/if}
                </div>
              {:else if seg.type === "url"}
                <ResearchCard urls={[{ url: seg.url, title: seg.title }]} searchQuery={showSearch ? searchQuery : ""} />
              {:else if seg.type === "json-table"}
                <ResearchCard jsonData={seg.data} jsonHeaders={seg.headers} searchQuery={showSearch ? searchQuery : ""} />
              {/if}
            {/each}
          {/if}
        </div>
      {:else if msg.role === "tool"}
        <div class="rounded border border-green-500/30 bg-green-950/20 px-3 py-2 text-xs font-mono text-green-300" data-message-index={i}>
          <div class="text-green-400 text-[10px] mb-1">tool result</div>
          <pre class="whitespace-pre-wrap break-all">{msg.content}</pre>
        </div>
      {/if}
    {/each}
  </div>

  <!-- Error -->
  {#if errorMsg || sessionStore.error}
    <div class="flex-none px-4 py-1 text-red-400 text-xs bg-red-950/50">
      {errorMsg ?? sessionStore.error}
    </div>
  {/if}

  <!-- Input -->
  <div class="flex-none border-t border-white/10 p-3">
    <!-- Uploaded file chips -->
    {#if uploadedFiles.length > 0}
      <div class="flex flex-wrap gap-2 mb-2">
        {#each uploadedFiles as file, i}
          <div class="flex items-center gap-1.5 px-2 py-1 rounded-full bg-gray-800 border border-white/10 text-xs">
            <span class="text-gray-300 truncate max-w-[100px]">{file.name}</span>
            <span class="text-gray-500 text-[10px]">📎</span>
            <button
              type="button"
              onclick={() => {
                uploadedFiles = uploadedFiles.filter((_, idx) => idx !== i)
              }}
              class="text-gray-500 hover:text-gray-300 transition-colors"
              aria-label="Remove {file.name}"
            >
              ✕
            </button>
          </div>
        {/each}
      </div>
    {/if}

    <div class="flex gap-2">
      <textarea
        bind:value={inputValue}
        bind:this={inputEl}
        onkeydown={handleKeydown}
        oninput={handleInput}
        placeholder="Message… (Enter to send, Shift+Enter for newline)"
        rows="1"
        class="flex-1 resize-none rounded bg-gray-800 border border-white/20 px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
      />
      <button
        onclick={handleSend}
        disabled={!inputValue.trim() || !activeId}
        class="px-4 py-2 rounded bg-blue-600 hover:bg-blue-500 disabled:bg-gray-700 disabled:text-gray-500 text-white text-sm transition-colors"
      >
        Send
      </button>
    </div>
  </div>

  <!-- Autocomplete dropdown -->
  {#if showAutocomplete && commandStore.filtered.length > 0}
    <div class="flex-none border-t border-white/10 bg-gray-900 max-h-64 overflow-y-auto">
      {#each commandStore.filtered as cmd, i}
        <button
          type="button"
          onclick={() => {
            const parsed = parseCommand("/" + cmd.name)
            commandStore.execute(cmd.name, parsed.args)
            inputValue = ""
            showAutocomplete = false
          }}
          class="w-full text-left px-4 py-2 text-sm hover:bg-gray-700 flex items-center gap-2 {i === selectedIndex ? 'bg-gray-700 text-white' : 'text-gray-300'}"
        >
          <span class="text-blue-400 font-mono">/{cmd.name}</span>
          <span class="text-gray-500 text-xs">{cmd.description}</span>
        </button>
      {/each}
    </div>
  {/if}
</div>
