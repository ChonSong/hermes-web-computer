<script lang="ts">
  import { onMount } from "svelte"
  import { chatSend, on, audioStart, audioStop, send as wsSend } from "../stores/ws"
  import ProfilePanel from "./ProfilePanel.svelte"
  import SkillsPanel from "./SkillsPanel.svelte"
  import CronPanel from "./CronPanel.svelte"
  import MemoryPanel from "./MemoryPanel.svelte"
  import SettingsPanel from "./SettingsPanel.svelte"

  // Tab state: Chat | Profiles | Skills | Crons | Memory | Settings
  type TabId = "chat" | "profiles" | "skills" | "crons" | "memory" | "settings"
  let activeTab = $state<TabId>("chat")

  // Chat state (kept for chat tab)
  interface Message {
    id: string
    role: "user" | "agent"
    text: string
    timestamp: Date
    attachment?: string
  }

  let messages: Message[] = $state([
    {
      id: "welcome",
      role: "agent",
      text: "Hello! I'm your agent. How can I help you today?",
      timestamp: new Date(),
    },
  ])

  let input = $state("")
  let isRecording = $state(false)
  let nextId = $state(2)
  let agentTyping = $state(false)
  let audioStatus = $state<"idle" | "listening" | "error">("idle")
  let collapsed = $state(false)
  let dropTargetActive = $state(false)

  // MediaRecorder state for voice
  let mediaRecorder: MediaRecorder | null = $state(null)
  let audioContext: AudioContext | null = $state(null)

  function send() {
    const text = input.trim()
    if (!text) return

    messages.push({
      id: String(nextId++),
      role: "user",
      text,
      timestamp: new Date(),
    })
    input = ""
    agentTyping = true

    // Send to backend via WebSocket
    chatSend(text)
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      send()
    }
  }

  async function startRecording() {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })

      // Signal backend to start audio session
      audioStart("session-1")
      audioStatus = "listening"
      isRecording = true

      // Set up MediaRecorder to capture audio
      mediaRecorder = new MediaRecorder(stream, {
        mimeType: "audio/webm;codecs=opus",
      })

      mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          // Send audio data to backend
          event.data.arrayBuffer().then((buffer) => {
            const chunk = new Uint8Array(buffer)
            wsSend({
              protocol: "audio",
              method: "audio.stream",
              params: {
                opus_chunk: Array.from(chunk),
                session_id: "session-1",
              },
            })
          })
        }
      }

      // Collect data every 100ms for low latency
      mediaRecorder.start(100)

      // Stop all tracks when recording ends
      stream.getTracks().forEach((track) => {
        track.onended = () => {
          stopRecording()
        }
      })
    } catch (err) {
      console.error("Failed to start recording:", err)
      audioStatus = "error"
      isRecording = false
    }
  }

  function stopRecording() {
    if (mediaRecorder && mediaRecorder.state !== "inactive") {
      mediaRecorder.stop()
      mediaRecorder = null
    }

    // Signal backend to stop audio session
    audioStop()
    audioStatus = "idle"
    isRecording = false
  }

  function toggleVoice() {
    if (isRecording) {
      stopRecording()
    } else {
      startRecording()
    }
  }

  // Drag-and-drop handlers for file drops into agent chat
  function handleDragOver(e: DragEvent) {
    e.preventDefault()
    e.dataTransfer!.dropEffect = "copy"
    dropTargetActive = true
  }

  function handleDragLeave(e: DragEvent) {
    // Only deactivate if we actually left the panel (not just entered a child)
    if (!e.currentTarget.contains(e.relatedTarget as Node)) {
      dropTargetActive = false
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault()
    dropTargetActive = false
    const filePath = e.dataTransfer?.getData("text/plain")
    if (!filePath) return

    // Send file content request via WS
    wsSend({ protocol: "ui", method: "fs.read", params: { path: filePath } })

    // Add user message with file attachment indicator
    messages.push({
      id: String(nextId++),
      role: "user",
      text: `📎 Attached file: \`${filePath}\``,
      timestamp: new Date(),
      attachment: filePath,
    })
    agentTyping = true
  }

  let chatArea: HTMLElement | undefined = $state()

  $effect(() => {
    if (chatArea) {
      chatArea.scrollTop = chatArea.scrollHeight
    }
  })

  onMount(() => {
    // Listen for dock panel switch events from Dock component
    const handleDockPanel = (e: Event) => {
      const detail = (e as CustomEvent).detail as { panel: string }
      if (detail.panel === "profiles") activeTab = "profiles"
      else if (detail.panel === "skills") activeTab = "skills"
      else if (detail.panel === "crons") activeTab = "crons"
      else if (detail.panel === "memory") activeTab = "memory"
      else if (detail.panel === "settings") activeTab = "settings"
    }
    window.addEventListener("hwc-dock-panel", handleDockPanel)

    // Listen for agent chat replies
    on("chat.reply", (data: unknown) => {
      agentTyping = false
      const resp = data as { message?: string; complete?: boolean }
      if (resp.message) {
        messages.push({
          id: String(nextId++),
          role: "agent",
          text: resp.message,
          timestamp: new Date(),
        })
      }
    })

    // Handle streaming replies
    on("chat.streaming", (data: unknown) => {
      const resp = data as { text?: string; done?: boolean }
      if (resp.text) {
        // Update the last agent message or create a new one
        const lastAgent = messages.findLast(m => m.role === "agent" && m.id !== "welcome")
        if (lastAgent) {
          lastAgent.text += resp.text
        }
      }
      if (resp.done) {
        agentTyping = false
      }
    })

    // Handle chat errors
    on("chat.error", (data: unknown) => {
      agentTyping = false
      const resp = data as { message?: string }
      messages.push({
        id: String(nextId++),
        role: "agent",
        text: `Error: ${resp.message || "Unknown error"}`,
        timestamp: new Date(),
      })
    })

    // Handle audio responses
    on("audio.response", (data: unknown) => {
      const resp = data as { data?: string; text?: string }
      if (resp.text) {
        // Audio transcribed to text
        messages.push({
          id: String(nextId++),
          role: "user",
          text: resp.text,
          timestamp: new Date(),
        })
        agentTyping = true
        chatSend(resp.text)
      }
    })

    // Handle audio errors
    on("audio.error", (data: unknown) => {
      const resp = data as { message?: string }
      console.error("Audio error:", resp.message)
      audioStatus = "error"
      isRecording = false
    })

    // Return cleanup
    return () => {
      window.removeEventListener("hwc-dock-panel", handleDockPanel)
    }
  })
</script>

<div
  class="h-full mt-12 mr-1 mb-1 flex flex-col overflow-hidden transition-all duration-200 ease-out
    backdrop-blur-2xl bg-[#12121a]/80 border border-white/10 rounded-2xl shadow-panel"
  class:w-0={collapsed}
  class:w-[360px]={!collapsed}
  class:opacity-0={collapsed}
  ondragover={handleDragOver}
  ondragleave={handleDragLeave}
  ondrop={handleDrop}
>
  <!-- Tab bar -->
  <div class="flex-shrink-0 px-2 pt-2 gap-1 flex items-center border-b border-white/10 pb-1">
    <div class="flex gap-0.5 flex-1">
      <button
        class="px-3 py-1.5 text-xs font-medium transition-colors rounded-lg
          {activeTab === 'chat' ? 'text-white bg-white/10' : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
        onclick={() => activeTab = "chat"}
      >
        💬 Chat
      </button>
      <button
        class="px-3 py-1.5 text-xs font-medium transition-colors rounded-lg
          {activeTab === 'profiles' ? 'text-white bg-white/10' : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
        onclick={() => activeTab = "profiles"}
      >
        👤 Profiles
      </button>
      <button
        class="px-3 py-1.5 text-xs font-medium transition-colors rounded-lg
          {activeTab === 'skills' ? 'text-white bg-white/10' : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
        onclick={() => activeTab = "skills"}
      >
        ◆ Skills
      </button>
      <button
        class="px-3 py-1.5 text-xs font-medium transition-colors rounded-lg
          {activeTab === 'crons' ? 'text-white bg-white/10' : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
        onclick={() => activeTab = "crons"}
      >
        ⏰ Crons
      </button>
      <button
        class="px-3 py-1.5 text-xs font-medium transition-colors rounded-lg
          {activeTab === 'memory' ? 'text-white bg-white/10' : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
        onclick={() => activeTab = "memory"}
      >
        🧠 Memory
      </button>
      <button
        class="px-3 py-1.5 text-xs font-medium transition-colors rounded-lg
          {activeTab === 'settings' ? 'text-white bg-white/10' : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
        onclick={() => activeTab = "settings"}
      >
        ⚙️
      </button>
    </div>
    <button
      class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition-colors rounded-lg hover:bg-white/5 shrink-0"
      onclick={() => collapsed = !collapsed}
      aria-label="Toggle panel"
    >
      ▶
    </button>
  </div>

  <!-- Tab content -->
  <div class="flex-1 overflow-hidden">
    {#if activeTab === "chat"}
      <!-- Chat content -->
      <div bind:this={chatArea} class="h-full overflow-y-auto px-4 py-4 space-y-3">
        {#each messages as msg (msg.id)}
          <div class="{msg.role === 'user' ? 'text-right' : 'text-left'}">
            {#if msg.attachment}
              <div class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-purple-500/10 text-purple-300 mb-1 {msg.role === 'user' ? 'ml-auto' : ''}">
                <span>📎</span>
                <span class="truncate max-w-[200px]">{msg.attachment.split('/').pop()}</span>
              </div>
            {/if}
            <p class="text-sm leading-relaxed
              {msg.role === 'user' ? 'text-white' : 'text-purple-300'}">
              {msg.text}
            </p>
          </div>
        {/each}
        {#if agentTyping}
          <div class="text-left">
            <div class="flex items-center gap-1">
              <div class="w-1.5 h-1.5 bg-purple-400/50 rounded-full animate-bounce" style="animation-delay: 0ms"></div>
              <div class="w-1.5 h-1.5 bg-purple-400/50 rounded-full animate-bounce" style="animation-delay: 150ms"></div>
              <div class="w-1.5 h-1.5 bg-purple-400/50 rounded-full animate-bounce" style="animation-delay: 300ms"></div>
            </div>
          </div>
        {/if}
      </div>
      <!-- Input area -->
      <div class="flex-shrink-0 px-4 py-3 border-t border-white/5">
        <div class="flex items-center gap-2">
          {#if !isRecording}
            <button
              onclick={startRecording}
              class="flex-shrink-0 w-9 h-9 rounded-full flex items-center justify-center transition-colors
                     bg-white/5 text-gray-400 hover:bg-white/10 hover:text-red-400 border border-white/10"
              aria-label="Start recording"
              title="Start recording"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z" />
                <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
                <line x1="12" y1="19" x2="12" y2="22" />
              </svg>
            </button>
          {:else}
            <button
              onclick={stopRecording}
              class="flex-shrink-0 w-9 h-9 rounded-full flex items-center justify-center transition-colors
                     bg-red-500/20 text-red-400 border border-red-500/30 animate-pulse"
              aria-label="Stop recording"
              title="Stop recording"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
                <rect x="6" y="6" width="12" height="12" rx="2" />
              </svg>
            </button>
          {/if}

          {#if audioStatus === "listening"}
            <span class="text-xs text-red-400 font-mono">● REC</span>
          {:else if audioStatus === "error"}
            <span class="text-xs text-red-500 font-mono">⚠ mic error</span>
          {/if}

          <input
            type="text"
            bind:value={input}
            onkeydown={handleKeydown}
            placeholder="Type a message..."
            class="flex-1 bg-white/5 text-white rounded-full px-4 py-2 text-sm
                   placeholder-gray-500 border border-white/10
                   focus:outline-none focus:border-purple-500/50 focus:ring-1 focus:ring-purple-500/20"
          />

          <button
            onclick={send}
            class="flex-shrink-0 w-9 h-9 rounded-full bg-purple-600 hover:bg-purple-500
                   text-white flex items-center justify-center transition-colors
                   disabled:opacity-40 disabled:cursor-not-allowed"
            aria-label="Send message"
            disabled={!input.trim()}
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="22" y1="2" x2="11" y2="13" />
              <polygon points="22 2 15 22 11 13 2 9 22 2" />
            </svg>
          </button>
        </div>
      </div>
    {:else if activeTab === "profiles"}
      <ProfilePanel />
    {:else if activeTab === "skills"}
      <SkillsPanel />
    {:else if activeTab === "crons"}
      <CronPanel />
    {:else if activeTab === "memory"}
      <MemoryPanel />
    {:else if activeTab === "settings"}
      <SettingsPanel />
    {/if}
  </div>
</div>

<!-- Collapse trigger edge -->
{#if collapsed}
  <div
    class="absolute right-0 top-12 bottom-1 w-[3px] bg-purple-500/40 cursor-pointer hover:bg-purple-500 transition-colors"
    onclick={() => collapsed = false}
    aria-label="Expand panel"
  />
{/if}
