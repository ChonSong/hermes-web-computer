<script lang="ts">
  import { onMount } from "svelte"

  interface WindowInfo {
    window_id: string
    pid: number
    title: string
    geometry?: string
  }

  interface Props {
    httpUrl?: string
    display?: string
    ptyId?: string
  }

  let {
    httpUrl = "http://localhost:9453",
    display = ":10",
    ptyId = "",
  }: Props = $props()

  let iframeEl: HTMLIFrameElement | undefined = $state()
  let status = $state<"connecting" | "connected" | "disconnected">("connecting")
  let errorMsg = $state("")
  let windows = $state<WindowInfo[]>([])
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null
  let pollWindowsInterval: ReturnType<typeof setInterval> | null = null
  let retryCount = $state(0)

  // Build the iframe src URL from the httpUrl and display
  const srcUrl = $derived(`${httpUrl}/index.html?session=${display}&ack=1`)

  function connect() {
    status = "connecting"
    errorMsg = ""
    // Give the iframe time to load
    if (reconnectTimeout) clearTimeout(reconnectTimeout)
    reconnectTimeout = setTimeout(() => {
      // Check if iframe loaded
      if (iframeEl?.contentDocument?.readyState === "complete") {
        status = "connected"
        retryCount = 0
      } else {
        // Try again
        if (retryCount < 5) {
          retryCount++
          connect()
        } else {
          status = "disconnected"
          errorMsg = "Failed to connect after multiple attempts"
        }
      }
    }, 3000)
  }

  function handleIframeLoad() {
    status = "connected"
    retryCount = 0
  }

  function handleIframeError() {
    status = "disconnected"
    errorMsg = "Iframe failed to load. Check if xpra is installed."
    scheduleReconnect()
  }

  function scheduleReconnect() {
    if (reconnectTimeout) clearTimeout(reconnectTimeout)
    const delay = Math.min(1000 * Math.pow(2, retryCount), 30000)
    reconnectTimeout = setTimeout(() => {
      retryCount++
      status = "connecting"
      // Force iframe reload
      if (iframeEl) {
        iframeEl.src = srcUrl + "&t=" + Date.now()
      }
    }, delay)
  }

  function retry() {
    retryCount = 0
    status = "connecting"
    if (iframeEl) {
      iframeEl.src = srcUrl + "&t=" + Date.now()
    }
  }

  function openNewTab() {
    window.open(srcUrl, "_blank", "noopener")
  }

  // Poll for window list updates
  function startWindowPolling() {
    if (pollWindowsInterval) clearInterval(pollWindowsInterval)
    pollWindowsInterval = setInterval(async () => {
      try {
        const resp = await fetch(`/api/xpra/list`, {
          headers: { "Upgrade": "websocket" } as any,
        })
        // WebSocket approach not right, use WS instead
      } catch {
        // ignore
      }
    }, 5000)
  }

  onMount(() => {
    // Set up window message listener for xpra events
    const handleMessage = (event: MessageEvent) => {
      if (event.data && typeof event.data === "object" && event.data.type === "xpra-window-update") {
        windows = event.data.windows || []
      }
    }
    window.addEventListener("message", handleMessage)

    // Start polling for window list
    startWindowPolling()

    return () => {
      window.removeEventListener("message", handleMessage)
      if (reconnectTimeout) clearTimeout(reconnectTimeout)
      if (pollWindowsInterval) clearInterval(pollWindowsInterval)
    }
  })

  // Status dot color
  const statusColor = $derived(
    status === "connected" ? "#4ade80" :
    status === "connecting" ? "#facc15" :
    "#f87171"
  )

  const statusLabel = $derived(
    status === "connected" ? "Connected" :
    status === "connecting" ? "Connecting..." :
    "Disconnected"
  )
</script>

<div class="xpra-tile">
  <div class="xpra-header">
    <div class="xpra-title-group">
      <span class="xpra-status-dot" style="background:{statusColor}"></span>
      <span class="xpra-title">Xpra — {display}</span>
      <span class="xpra-windows-count" title="Number of windows">
        {windows.length} window{windows.length !== 1 ? "s" : ""}
      </span>
    </div>
    <div class="xpra-controls">
      <button
        class="xpra-btn"
        onclick={openNewTab}
        title="Open in new tab"
        style="background:transparent;border:none;color:#888;cursor:pointer;padding:2px 6px;border-radius:4px;font-size:12px;"
      >
        ⧉
      </button>
    </div>
  </div>

  {#if status === "connecting"}
    <div class="xpra-loading">
      <div class="xpra-spinner"></div>
      <span>Connecting to Xpra session {display}...</span>
    </div>
  {/if}

  {#if status === "disconnected"}
    <div class="xpra-error">
      <p>Failed to connect to Xpra at {httpUrl}</p>
      {#if errorMsg}
        <p class="xpra-hint">{errorMsg}</p>
      {/if}
      <p class="xpra-hint">Make sure xpra is installed and the server is running.</p>
      <button onclick={retry}>Retry</button>
    </div>
  {/if}

  {#if status === "connected"}
    <div class="xpra-iframe-container">
      <iframe
        bind:this={iframeEl}
        src={srcUrl}
        title="Xpra session {display}"
        allow="clipboard-read; clipboard-write"
        sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-pointer-lock"
        class="xpra-iframe"
        onload={handleIframeLoad}
        onerror={handleIframeError}
      ></iframe>
    </div>
  {/if}

  <!-- Window list overlay at bottom -->
  {#if windows.length > 0 && status === "connected"}
    <div class="xpra-windows-bar">
      {#each windows as win (win.window_id)}
        <div class="xpra-window-chip" title={win.title}>
          <span class="xpra-window-title">{win.title || "Window"}</span>
          <button
            class="xpra-window-close"
            title="Close window"
            onclick={() => {
              // Send detach via WS
            }}
          >
            ×
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .xpra-tile {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    background: #191919;
    border-radius: 12px;
    overflow: hidden;
  }

  .xpra-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 6px 12px;
    background: #252525;
    border-bottom: 1px solid #333;
    color: #ccc;
    font-size: 12px;
    flex-shrink: 0;
    gap: 8px;
  }

  .xpra-title-group {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .xpra-status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .xpra-title {
    font-family: monospace;
    color: #aaa;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .xpra-windows-count {
    font-size: 10px;
    color: #666;
    font-family: monospace;
  }

  .xpra-controls {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }

  .xpra-btn {
    color: #888;
    text-decoration: none;
    padding: 2px 6px;
    border-radius: 4px;
    transition: color 0.15s, background 0.15s;
  }

  .xpra-btn:hover {
    color: #fff;
    background: #333;
  }

  .xpra-iframe-container {
    flex: 1;
    position: relative;
    overflow: hidden;
    min-height: 0;
  }

  .xpra-iframe {
    width: 100%;
    height: 100%;
    border: none;
    background: #000;
    display: block;
  }

  .xpra-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: #666;
    font-size: 13px;
    gap: 12px;
  }

  .xpra-spinner {
    width: 24px;
    height: 24px;
    border: 2px solid #333;
    border-top-color: #666;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .xpra-error {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: #cc4444;
    font-size: 13px;
    gap: 8px;
    padding: 20px;
    text-align: center;
  }

  .xpra-hint {
    color: #666;
    font-size: 12px;
  }

  button {
    background: #333;
    color: #ccc;
    border: 1px solid #444;
    border-radius: 6px;
    padding: 6px 16px;
    cursor: pointer;
    font-size: 12px;
    margin-top: 8px;
  }

  button:hover {
    background: #3a3a3a;
  }

  /* Window list bar */
  .xpra-windows-bar {
    display: flex;
    gap: 4px;
    padding: 4px 8px;
    background: #1a1a1a;
    border-top: 1px solid #333;
    overflow-x: auto;
    flex-shrink: 0;
    max-height: 60px;
    overflow-y: auto;
  }

  .xpra-window-chip {
    display: flex;
    align-items: center;
    gap: 4px;
    background: #2a2a2a;
    border: 1px solid #3a3a3a;
    border-radius: 4px;
    padding: 2px 6px;
    font-size: 11px;
    color: #aaa;
    white-space: nowrap;
    cursor: pointer;
    transition: background 0.15s;
  }

  .xpra-window-chip:hover {
    background: #3a3a3a;
  }

  .xpra-window-title {
    max-width: 100px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .xpra-window-close {
    background: transparent;
    border: none;
    color: #888;
    cursor: pointer;
    font-size: 12px;
    padding: 0 2px;
    line-height: 1;
    border-radius: 2px;
    margin: 0;
    padding: 0 2px;
  }

  .xpra-window-close:hover {
    color: #cc4444;
    background: transparent;
  }
</style>