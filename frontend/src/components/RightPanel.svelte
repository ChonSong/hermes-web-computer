<script lang="ts">
  import { onMount } from "svelte"
  import { chatSend, on, audioStart, audioStop, send as wsSend } from "../stores/ws"

  interface Message {
    id: string
    role: "user" | "agent"
    text: string
    timestamp: Date
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

  let chatArea: HTMLElement | undefined = $state()

  $effect(() => {
    if (chatArea) {
      chatArea.scrollTop = chatArea.scrollHeight
    }
  })

  onMount(() => {
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
  })
</script>

<div class="h-full flex flex-col bg-gray-900 border-l border-gray-800">
  <!-- Header -->
  <div class="flex-shrink-0 px-4 py-3 border-b border-gray-800 flex items-center">
    <h2 class="text-white font-semibold text-base">Agent</h2>
  </div>

  <!-- Messages -->
  <div bind:this={chatArea} class="flex-1 overflow-y-auto px-4 py-4 space-y-4">
    {#each messages as msg (msg.id)}
      <div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
        <div
          class="max-w-[80%] rounded-2xl px-4 py-2 text-sm leading-relaxed
          {msg.role === 'user'
            ? 'bg-blue-600 text-white rounded-br-md'
            : 'bg-gray-800 text-gray-200 rounded-bl-md'}"
        >
          {msg.text}
        </div>
      </div>
    {/each}
    {#if agentTyping}
      <div class="flex justify-start">
        <div class="bg-gray-800 text-gray-400 rounded-2xl px-4 py-2 text-sm rounded-bl-md">
          <div class="flex items-center gap-1">
            <div class="w-1.5 h-1.5 bg-gray-500 rounded-full animate-bounce" style="animation-delay: 0ms"></div>
            <div class="w-1.5 h-1.5 bg-gray-500 rounded-full animate-bounce" style="animation-delay: 150ms"></div>
            <div class="w-1.5 h-1.5 bg-gray-500 rounded-full animate-bounce" style="animation-delay: 300ms"></div>
          </div>
        </div>
      </div>
    {/if}
  </div>

  <!-- Input area -->
  <div class="flex-shrink-0 px-4 py-3 border-t border-gray-800">
    <div class="flex items-center gap-2">
      <!-- Voice record button -->
      {#if !isRecording}
        <button
          onclick={startRecording}
          class="flex-shrink-0 w-9 h-9 rounded-lg flex items-center justify-center transition-colors
                 bg-gray-800 text-gray-400 hover:bg-gray-700 hover:text-red-400"
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
          class="flex-shrink-0 w-9 h-9 rounded-lg flex items-center justify-center transition-colors
                 bg-red-900 text-red-400 hover:bg-red-800 animate-pulse"
          aria-label="Stop recording"
          title="Stop recording"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
            <rect x="6" y="6" width="12" height="12" rx="2" />
          </svg>
        </button>
      {/if}

      <!-- Audio status indicator -->
      {#if audioStatus === "listening"}
        <span class="text-xs text-red-400 font-mono">● REC</span>
      {:else if audioStatus === "error"}
        <span class="text-xs text-red-500 font-mono">⚠ mic error</span>
      {/if}

      <!-- Text input -->
      <input
        type="text"
        bind:value={input}
        onkeydown={handleKeydown}
        placeholder="Type a message..."
        class="flex-1 bg-gray-800 text-gray-200 rounded-lg px-3 py-2 text-sm
               placeholder-gray-500 border border-gray-700
               focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
      />

      <!-- Send button -->
      <button
        onclick={send}
        class="flex-shrink-0 w-9 h-9 rounded-lg bg-blue-600 hover:bg-blue-500
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
</div>
