<script lang="ts">

  interface Message {
    id: string;
    role: "user" | "agent";
    text: string;
    timestamp: Date;
  }

  let messages: Message[] = $state([
    {
      id: "welcome",
      role: "agent",
      text: "Hello! I'm your agent. How can I help you today?",
      timestamp: new Date(),
    },
  ]);

  let input = $state("");
  let isRecording = $state(false);
  let nextId = $state(2);

  function send() {
    const text = input.trim();
    if (!text) return;

    messages.push({
      id: String(nextId++),
      role: "user",
      text,
      timestamp: new Date(),
    });
    input = "";
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  }

  function toggleVoice() {
    isRecording = !isRecording;
  }

  let chatArea: HTMLElement | undefined = $state();

  $effect(() => {
    if (chatArea) {
      chatArea.scrollTop = chatArea.scrollHeight;
    }
  });
</script>

<div class="h-full flex flex-col bg-gray-900 border-l border-gray-800">
  <!-- Header -->
  <div
    class="flex-shrink-0 px-4 py-3 border-b border-gray-800 flex items-center"
  >
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
  </div>

  <!-- Input area -->
  <div class="flex-shrink-0 px-4 py-3 border-t border-gray-800">
    <div class="flex items-center gap-2">
      <!-- Voice toggle -->
      <button
        on:click={toggleVoice}
        class="flex-shrink-0 w-9 h-9 rounded-lg flex items-center justify-center transition-colors"
        class:bg-gray-800={!isRecording}
        class:bg-red-900={isRecording}
        class:text-gray-400={!isRecording}
        class:text-red-400={isRecording}
        title={isRecording ? "Stop recording" : "Start recording"}
      >
        {#if isRecording}
          <!-- Stop icon -->
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="w-4 h-4"
            viewBox="0 0 24 24"
            fill="currentColor"
          >
            <rect x="6" y="6" width="12" height="12" rx="2" />
          </svg>
        {:else}
          <!-- Microphone icon -->
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="w-4 h-4"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z" />
            <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
            <line x1="12" y1="19" x2="12" y2="22" />
          </svg>
        {/if}
      </button>

      <!-- Text input -->
      <input
        type="text"
        bind:value={input}
        on:keydown={handleKeydown}
        placeholder="Type a message..."
        class="flex-1 bg-gray-800 text-gray-200 rounded-lg px-3 py-2 text-sm
               placeholder-gray-500 border border-gray-700
               focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
      />

      <!-- Send button -->
      <button
        on:click={send}
        class="flex-shrink-0 w-9 h-9 rounded-lg bg-blue-600 hover:bg-blue-500
               text-white flex items-center justify-center transition-colors
               disabled:opacity-40 disabled:cursor-not-allowed"
        disabled={!input.trim()}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="w-4 h-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <line x1="22" y1="2" x2="11" y2="13" />
          <polygon points="22 2 15 22 11 13 2 9 22 2" />
        </svg>
      </button>
    </div>
  </div>
</div>
