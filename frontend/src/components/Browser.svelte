<script lang="ts">
  import { onMount, onDestroy } from "svelte"
  import { send, on } from "../stores/ws"

  let { id = "", browserId: bId = "" }: { id?: string; browserId?: string } = $props()

  let url = $state("about:blank")
  let urlInput = $state("about:blank")
  let screenshot = $state<string | null>(null)
  let loading = $state(false)
  let error = $state<string | null>(null)

  function navigate(targetUrl: string) {
    if (!targetUrl || targetUrl === url) return
    loading = true
    error = null

    if (!targetUrl.startsWith("http://") && !targetUrl.startsWith("https://")) {
      targetUrl = "https://" + targetUrl
    }

    send({
      protocol: "agent",
      method: "browser.navigate",
      params: {
        session_id: bId,
        url: targetUrl,
      },
    })
    urlInput = targetUrl
  }

  function handleUrlSubmit() {
    navigate(urlInput)
  }

  function goBack() {
    if (!bId) return
    send({
      protocol: "agent",
      method: "browser.back",
      params: { session_id: bId },
    })
  }

  function goForward() {
    if (!bId) return
    send({
      protocol: "agent",
      method: "browser.forward",
      params: { session_id: bId },
    })
  }

  function refresh() {
    navigate(url)
  }

  // Click handler for interacting with the screenshot
  let imgContainer: HTMLDivElement

  function handleImageClick(event: MouseEvent) {
    if (!bId || !screenshot) return
    const rect = imgContainer.getBoundingClientRect()
    const img = imgContainer.querySelector("img")
    if (!img) return

    const viewportW = 1280
    const viewportH = 900
    const x = ((event.clientX - rect.left) / rect.width) * viewportW
    const y = ((event.clientY - rect.top) / rect.height) * viewportH

    send({
      protocol: "agent",
      method: "browser.click",
      params: {
        session_id: bId,
        x,
        y,
      },
    })
  }

  let cleanupFns: (() => void)[] = []

  onMount(() => {
    cleanupFns.push(
      on("browser.navigated", (data: unknown) => {
        const resp = data as { url?: string; screenshot?: string }
        if (resp.url) { url = resp.url; urlInput = resp.url }
        if (resp.screenshot) screenshot = resp.screenshot
        loading = false
      }),
      on("browser.screenshot.response", (data: unknown) => {
        const resp = data as { screenshot?: string }
        if (resp.screenshot) screenshot = resp.screenshot
        loading = false
      }),
      on("browser.clicked", () => {
        if (bId) send({ protocol: "agent", method: "browser.screenshot", params: { session_id: bId } })
      }),
      on("browser.input.done", () => {
        if (bId) send({ protocol: "agent", method: "browser.screenshot", params: { session_id: bId } })
      }),
      on("browser.error", (data: unknown) => {
        const resp = data as { message?: string }
        error = resp.message || "Unknown error"
        loading = false
      }),
    )

    if (bId && url && url !== "about:blank") {
      navigate(url)
    }
  })

  onDestroy(() => {
    cleanupFns.forEach(fn => fn())
  })
</script>

<div class="flex flex-col h-full bg-gray-950">
  <!-- URL bar -->
  <div class="flex items-center gap-1 p-1.5 bg-gray-900 border-b border-gray-700">
    <button
      class="px-2 py-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded transition-colors text-sm"
      onclick={goBack}
      aria-label="Back"
      title="Back"
    >
      ◀
    </button>
    <button
      class="px-2 py-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded transition-colors text-sm"
      onclick={goForward}
      aria-label="Forward"
      title="Forward"
    >
      ▶
    </button>
    <button
      class="px-2 py-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded transition-colors text-sm"
      onclick={refresh}
      aria-label="Refresh"
      title="Refresh"
    >
      ↻
    </button>
    <input
      type="text"
      bind:value={urlInput}
      class="flex-1 px-3 py-1 bg-gray-800 border border-gray-600 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
      placeholder="Enter URL..."
      onkeydown={(e) => { if (e.key === "Enter") handleUrlSubmit() }}
    />
    <button
      class="px-3 py-1 text-sm bg-blue-600 hover:bg-blue-500 text-white rounded transition-colors"
      onclick={handleUrlSubmit}
    >
      Go
    </button>
  </div>

  <!-- Browser viewport -->
  <div class="flex-1 relative overflow-hidden" bind:this={imgContainer}>
    {#if loading}
      <div class="absolute inset-0 flex items-center justify-center bg-gray-900">
        <div class="text-gray-400">Loading...</div>
      </div>
    {/if}

    {#if error}
      <div class="absolute inset-0 flex items-center justify-center bg-gray-900">
        <div class="text-red-400 text-center p-4">
          <p class="font-bold">Error</p>
          <p class="text-sm mt-1">{error}</p>
        </div>
      </div>
    {/if}

    {#if screenshot}
      <img
        src="data:image/png;base64,{screenshot}"
        alt="Browser screenshot"
        class="w-full h-full object-contain cursor-pointer"
        onclick={handleImageClick}
      />
    {:else if !loading}
      <div class="flex items-center justify-center h-full text-gray-500">
        <div class="text-center">
          <p class="text-lg">🌐</p>
          <p class="text-sm mt-2">Enter a URL to browse</p>
        </div>
      </div>
    {/if}
  </div>
</div>
