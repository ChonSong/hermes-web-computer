<script lang="ts">
  /**
   * ChatPanel — Session chat UI rendered in a tile.
   * Streams tokens, shows tool calls, auto-scrolls.
   */
  import { onMount, tick } from "svelte"
  import { on, send } from "../stores/ws"
  import { sessionStore } from "../stores/sessions.svelte"
  import type { SessionMessage } from "../stores/sessions.svelte"

  let messagesEl: HTMLDivElement
  let inputValue = $state("")
  let streaming = $state(false)
  let streamingContent = $state("")
  let errorMsg = $state<string | null>(null)

  // Active session from store
  let activeSession = $derived(sessionStore.activeSession)

  // Full message list including live streaming
  let allMessages = $derived([
    ...(activeSession?.messages ?? []),
    ...(streaming ? [{ role: "assistant" as const, content: streamingContent }] : [])
  ])

  // Auto-scroll when messages change
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
    if (!text || streaming) return
    inputValue = ""

    // Add user message optimistically
    if (activeSession) {
      sessionStore.activeSession = {
        ...activeSession,
        messages: [...(activeSession.messages ?? []), { role: "user", content: text }],
      }
    }

    errorMsg = null
    streaming = true
    streamingContent = ""

    let msgId = ""
    const streamCleanup = on("chat.streaming", (data: any) => {
      msgId = data.id ?? msgId
      streamingContent = (data.content ?? "") as string
    })

    const replyCleanup = on("chat.reply", (data: any) => {
      streamCleanup()
      replyCleanup()
      errorCleanup()
      streaming = false
      if (activeSession) {
        const finalMsg: SessionMessage = {
          role: "assistant",
          content: data.content ?? streamingContent,
        }
        sessionStore.activeSession = {
          ...activeSession,
          messages: [...(activeSession.messages ?? []), finalMsg],
        }
      }
    })

    const errorCleanup = on("chat.error", (data: any) => {
      streamCleanup()
      replyCleanup()
      errorCleanup()
      streaming = false
      errorMsg = (data as any)?.message ?? "Unknown error"
    })

    send({
      protocol: "agent",
      method: "chat.send",
      params: { message: text, session_id: sessionStore.activeId },
    })
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  // Render tool calls as formatted blocks
  function getToolCalls(msg: SessionMessage): Array<{ id: string; name: string; args: string }> {
    if (!msg.tool_calls) return []
    return msg.tool_calls.map(tc => ({
      id: tc.id,
      name: tc.function.name,
      args: tc.function.arguments,
    }))
  }

  function formatJson(str: string): string {
    try {
      return JSON.stringify(JSON.parse(str), null, 2)
    } catch {
      return str
    }
  }

  function safeippet(str: string, len = 200): string {
    return str.length > len ? str.slice(0, len) + "..." : str
  }
</script>

<div class="flex flex-col h-full bg-[#0e0e16] rounded-2xl">
  <!-- Header -->
  <div class="shrink-0 px-4 py-3 border-b border-white/5 flex items-center gap-3">
    <div class="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
    <span class="text-sm font-medium text-gray-300 truncate">
      {activeSession?.title ?? "No session"}
    </span>
    {#if streaming}
      <span class="ml-auto text-xs text-purple-400 animate-pulse">thinking...</span>
    {/if}
  </div>

  <!-- Messages -->
  <div bind:this={messagesEl} class="flex-1 overflow-y-auto px-4 py-3 space-y-4">
    {#if !activeSession}
      <div class="flex items-center justify-center h-full text-gray-500 text-sm">
        Select or create a session to start chatting
      </div>
    {:else if allMessages.length === 0}
      <div class="flex flex-col items-center justify-center h-full gap-3 text-gray-500">
        <span class="text-3xl">💬</span>
        <p class="text-sm">Send a message to start the conversation</p>
      </div>
    {:else}
      {#each allMessages as msg, i (i)}
        <div class="flex gap-3 {msg.role === 'user' ? 'flex-row-reverse' : ''}">
          <!-- Avatar -->
          <div class="shrink-0 w-7 h-7 rounded-full flex items-center justify-center text-xs
                      {msg.role === 'user' ? 'bg-purple-600' : msg.role === 'tool' ? 'bg-orange-600' : 'bg-gray-700'}">
            {#if msg.role === 'user'}👤
            {:else if msg.role === 'tool'}🔧
            {:else}🤖
            {/if}
          </div>

          <div class="flex-1 min-w-0 {msg.role === 'user' ? 'text-right' : ''}">
            <div class="inline-block max-w-[85%] px-3 py-2 rounded-2xl text-sm text-gray-200
                        {msg.role === 'user' ? 'bg-purple-600/30 rounded-tr-sm' : msg.role === 'tool' ? 'bg-orange-500/20 rounded-tl-sm' : 'bg-white/5 rounded-tl-sm'}">
              {#if msg.role === 'tool'}
                <span class="text-orange-400 text-xs font-mono">{msg.name ?? 'tool'}</span>
                <pre class="text-xs text-gray-400 mt-1 whitespace-pre-wrap break-all max-h-32 overflow-y-auto">{safeippet(formatJson(msg.content as string))}</pre>
              {:else}
                <pre class="whitespace-pre-wrap break-words text-gray-200">{msg.content as string}</pre>
              {/if}
            </div>

            <!-- Tool calls for assistant messages -->
            {#if msg.role === 'assistant' && msg.tool_calls}
              <div class="mt-1 space-y-1">
                {#each msg.tool_calls as tc}
                  <div class="inline-flex items-center gap-1 px-2 py-1 rounded bg-purple-500/20 text-purple-300 text-xs font-mono">
                    <span>🔧</span>
                    <span>{tc.function.name}</span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      {/each}

      <!-- Streaming indicator -->
      {#if streaming && streamingContent}
        <div class="flex gap-3">
          <div class="shrink-0 w-7 h-7 rounded-full bg-gray-700 flex items-center justify-center text-xs">🤖</div>
          <div class="flex-1">
            <div class="inline-block max-w-[85%] px-3 py-2 rounded-2xl rounded-tl-sm bg-white/5 text-sm">
              <span class="text-gray-200">{streamingContent}</span>
              <span class="inline-block w-2 h-4 bg-purple-400 animate-pulse ml-1" />
            </div>
          </div>
        </div>
      {/if}
    {/if}

    <!-- Error -->
    {#if errorMsg}
      <div class="flex gap-3">
        <div class="shrink-0 w-7 h-7 rounded-full bg-red-900 flex items-center justify-center text-xs">⚠</div>
        <div class="flex-1">
          <div class="inline-block px-3 py-2 rounded-2xl bg-red-900/30 text-red-400 text-sm">
            {errorMsg}
          </div>
        </div>
      </div>
    {/if}
  </div>

  <!-- Input -->
  <div class="shrink-0 px-4 py-3 border-t border-white/5">
    <div class="flex gap-2 items-end">
      <textarea
        bind:value={inputValue}
        onkeydown={handleKeydown}
        placeholder={activeSession ? "Type a message..." : "No session selected"}
        disabled={!activeSession || streaming}
        rows="1"
        class="flex-1 px-3 py-2 text-sm rounded-xl bg-white/5 border border-white/10
               text-gray-200 placeholder-gray-600 resize-none
               focus:outline-none focus:border-purple-500/50 disabled:opacity-50"
      />
      <button
        onclick={handleSend}
        disabled={!activeSession || !inputValue.trim() || streaming}
        class="px-4 py-2 rounded-xl text-sm font-medium
               bg-purple-600 hover:bg-purple-500 disabled:bg-gray-700 disabled:text-gray-600
               text-white transition-colors"
      >
        {streaming ? "..." : "Send"}
      </button>
    </div>
  </div>
</div>
