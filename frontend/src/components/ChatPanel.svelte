<script lang="ts">
  /**
   * ChatPanel — Session chat UI rendered in a tile.
   * Reads streaming state from sessionStore, sends via sessionStore.send().
   */
  import { tick } from "svelte"
  import { sessionStore } from "../stores/sessions.svelte"
  import type { SessionMessage } from "../stores/sessions.svelte"

  let messagesEl: HTMLDivElement
  let inputValue = $state("")
  let errorMsg = $state<string | null>(null)

  // Active session
  let activeId = $derived(sessionStore.activeId)
  let activeSession = $derived(sessionStore.activeSession)

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
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
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
</script>

<div class="flex flex-col h-full bg-gray-900 text-gray-100 text-sm">
  <!-- Header -->
  <div class="flex-none px-4 py-3 border-b border-white/10">
    <div class="text-xs text-gray-400">
      {activeSession?.title ?? "New Chat"} &middot; {activeSession?.model ?? "agent"}
    </div>
  </div>

  <!-- Messages -->
  <div bind:this={messagesEl} class="flex-1 overflow-y-auto px-4 py-3 space-y-4">
    {#if allMessages.length === 0}
      <div class="text-gray-500 text-center mt-8 text-xs">
        Send a message to start the conversation.
      </div>
    {/if}

    {#each allMessages as msg, i}
      {#if msg.role === "user"}
        <div class="flex justify-end">
          <div class="max-w-[75%] rounded-lg px-3 py-2 bg-blue-600 text-white text-sm">
            {renderContent(msg)}
          </div>
        </div>
      {:else if msg.role === "assistant"}
        <div class="flex flex-col">
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
            <div class="text-gray-100 text-sm whitespace-pre-wrap">{msg.content}</div>
          {/if}
        </div>
      {:else if msg.role === "tool"}
        <div class="rounded border border-green-500/30 bg-green-950/20 px-3 py-2 text-xs font-mono text-green-300">
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
    <div class="flex gap-2">
      <textarea
        bind:value={inputValue}
        onkeydown={handleKeydown}
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
</div>
