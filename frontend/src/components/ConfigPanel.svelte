<script lang="ts">
  /**
   * ConfigPanel — Right panel tab for Hermes config editor.
   * Features: model picker from /v1/models, env var management, restart button.
   */
  import { onMount } from "svelte"
  import { configStore, type ModelInfo } from "../stores/config.svelte"

  let activeSection = $state<string | null>("model")
  let newEnvKey = $state("")
  let newEnvValue = $state("")
  let isRestarting = $state(false)
  let restartMessage = $state("")

  let loading = $derived(configStore.loading)
  let error = $derived(configStore.error)
  let config = $derived(configStore.config)
  let models = $derived(configStore.models)
  let envVars = $derived(configStore.envVars)

  onMount(() => {
    configStore.refresh()
    configStore.loadModels()
    configStore.listEnv()
  })

  async function handleModelChange(e: Event) {
    const target = e.target as HTMLSelectElement
    await configStore.setConfig("model.default", target.value)
  }

  async function handleReasoningChange(e: Event) {
    const target = e.target as HTMLSelectElement
    await configStore.setConfig("agent.reasoning_effort", target.value)
  }

  async function handleMaxTurnsChange(e: Event) {
    const target = e.target as HTMLInputElement
    await configStore.setConfig("agent.max_turns", parseInt(target.value))
  }

  async function handleAddEnv() {
    if (!newEnvKey.trim()) return
    await configStore.setEnv(newEnvKey.trim(), newEnvValue)
    newEnvKey = ""
    newEnvValue = ""
  }

  async function handleDeleteEnv(key: string) {
    if (confirm(`Delete environment variable "${key}"?`)) {
      await configStore.deleteEnv(key)
    }
  }

  async function handleRestart() {
    if (!confirm("Are you sure you want to restart Hermes?")) return
    isRestarting = true
    restartMessage = ""
    const success = await configStore.restart()
    isRestarting = false
    restartMessage = success ? "Restart signal sent" : `Failed: ${configStore.error}`
  }

  function getReasoningOptions() {
    return ["low", "medium", "high", "xhigh", "xxhigh"]
  }
</script>

<div class="flex flex-col h-full">
  <div class="flex-none px-4 py-3 border-b border-white/10 flex items-center justify-between">
    <h2 class="text-white font-semibold text-base">Config</h2>
    <button
      class="px-3 py-1 text-xs rounded bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors"
      onclick={handleRestart}
      disabled={isRestarting}
    >
      {isRestarting ? "Restarting..." : "Restart"}
    </button>
  </div>

  {#if restartMessage}
    <div class="px-4 py-2 text-xs {restartMessage.includes('Failed') ? 'text-red-400' : 'text-green-400'} bg-white/5">
      {restartMessage}
    </div>
  {/if}

  {#if error}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-red-400 text-sm">{error}</div>
  {:else if loading}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-gray-500 text-sm">Loading config...</div>
  {:else}
    <div class="flex-1 overflow-y-auto px-2 py-2 space-y-1">
      <!-- Model Section -->
      <div>
        <button
          class="w-full flex items-center gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors text-left"
          onclick={() => activeSection = activeSection === "model" ? null : "model"}
        >
          <span class="text-base">🤖</span>
          <span class="text-gray-200 text-sm">Model</span>
          {#if config?.model?.default}
            <span class="ml-auto text-xs text-gray-500 truncate max-w-32">{config.model.default}</span>
          {/if}
        </button>

        {#if activeSection === "model"}
          <div class="mt-2 px-4 py-3 bg-white/5 rounded-lg space-y-3">
            <!-- Model Picker -->
            <div>
              <label class="block text-xs text-gray-400 mb-1">Default Model</label>
              <select
                class="w-full bg-white/10 border border-white/20 rounded px-3 py-2 text-sm text-white"
                value={config?.model?.default ?? ""}
                onchange={handleModelChange}
              >
                {#each models as model (model.id)}
                  <option value={model.id}>{model.name || model.id}</option>
                {/each}
                {#if !models.find(m => m.id === config?.model?.default)}
                  <option value={config?.model?.default}>{config?.model?.default}</option>
                {/if}
              </select>
            </div>

            <!-- Provider -->
            <div>
              <label class="block text-xs text-gray-400 mb-1">Provider</label>
              <input
                type="text"
                class="w-full bg-white/10 border border-white/20 rounded px-3 py-2 text-sm text-white"
                value={config?.model?.provider ?? ""}
                readonly
              />
            </div>

            <!-- Base URL -->
            <div>
              <label class="block text-xs text-gray-400 mb-1">Base URL</label>
              <input
                type="text"
                class="w-full bg-white/10 border border-white/20 rounded px-3 py-2 text-sm text-white"
                value={config?.model?.base_url ?? ""}
                readonly
              />
            </div>
          </div>
        {/if}
      </div>

      <!-- Agent Section -->
      <div>
        <button
          class="w-full flex items-center gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors text-left"
          onclick={() => activeSection = activeSection === "agent" ? null : "agent"}
        >
          <span class="text-base">⚡</span>
          <span class="text-gray-200 text-sm">Agent</span>
        </button>

        {#if activeSection === "agent"}
          <div class="mt-2 px-4 py-3 bg-white/5 rounded-lg space-y-3">
            <!-- Reasoning Effort -->
            <div>
              <label class="block text-xs text-gray-400 mb-1">Reasoning Effort</label>
              <select
                class="w-full bg-white/10 border border-white/20 rounded px-3 py-2 text-sm text-white"
                value={config?.agent?.reasoning_effort ?? "xhigh"}
                onchange={handleReasoningChange}
              >
                {#each getReasoningOptions() as opt}
                  <option value={opt}>{opt}</option>
                {/each}
              </select>
            </div>

            <!-- Max Turns -->
            <div>
              <label class="block text-xs text-gray-400 mb-1">Max Turns</label>
              <input
                type="number"
                class="w-full bg-white/10 border border-white/20 rounded px-3 py-2 text-sm text-white"
                value={config?.agent?.max_turns ?? 200}
                onchange={handleMaxTurnsChange}
              />
            </div>

            <!-- Verbose -->
            <div class="flex items-center gap-2">
              <input
                type="checkbox"
                id="verbose-mode"
                class="w-4 h-4 rounded"
                checked={config?.agent?.verbose ?? false}
                onchange={async (e) => {
                  const target = e.target as HTMLInputElement
                  await configStore.setConfig("agent.verbose", target.checked)
                }}
              />
              <label for="verbose-mode" class="text-xs text-gray-400">Verbose Mode</label>
            </div>
          </div>
        {/if}
      </div>

      <!-- Environment Variables Section -->
      <div>
        <button
          class="w-full flex items-center gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors text-left"
          onclick={() => activeSection = activeSection === "env" ? null : "env"}
        >
          <span class="text-base">🔧</span>
          <span class="text-gray-200 text-sm">Environment</span>
          <span class="ml-auto text-xs text-gray-500">{Object.keys(envVars).length} vars</span>
        </button>

        {#if activeSection === "env"}
          <div class="mt-2 px-4 py-3 bg-white/5 rounded-lg space-y-3">
            <!-- Add new env var -->
            <div class="flex gap-2">
              <input
                type="text"
                placeholder="KEY"
                class="flex-1 bg-white/10 border border-white/20 rounded px-2 py-1.5 text-sm text-white placeholder-gray-500"
                bind:value={newEnvKey}
              />
              <input
                type="text"
                placeholder="value"
                class="flex-1 bg-white/10 border border-white/20 rounded px-2 py-1.5 text-sm text-white placeholder-gray-500"
                bind:value={newEnvValue}
              />
              <button
                class="px-3 py-1.5 text-xs rounded bg-green-500/20 text-green-400 hover:bg-green-500/30"
                onclick={handleAddEnv}
              >
                Add
              </button>
            </div>

            <!-- Env vars list -->
            <div class="space-y-1 max-h-64 overflow-y-auto">
              {#each Object.entries(envVars) as [key, value] (key)}
                <div class="group flex items-center gap-2 py-1 px-2 rounded hover:bg-white/5">
                  <span class="text-xs text-cyan-400 font-mono truncate flex-1">{key}</span>
                  <span class="text-xs text-gray-500 truncate max-w-24">{value}</span>
                  <button
                    class="opacity-0 group-hover:opacity-100 text-xs text-red-400 hover:text-red-300"
                    onclick={() => handleDeleteEnv(key)}
                  >
                    ✕
                  </button>
                </div>
              {/each}
              {#if Object.keys(envVars).length === 0}
                <div class="text-xs text-gray-500 text-center py-2">No environment variables</div>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>