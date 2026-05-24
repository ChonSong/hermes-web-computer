<script lang="ts">
  import { onMount } from "svelte"

  interface Props {
    httpUrl?: string
    display?: string
  }

  let { httpUrl = "http://localhost:9453", display = ":10" }: Props = $props()

  let iframeEl: HTMLIFrameElement | undefined = $state()
  let loading = $state(true)
  let error = $state(false)

  const srcUrl = $derived(`${httpUrl}/index.html?session=${display}`)

  onMount(() => {
    if (!iframeEl) return
    const el = iframeEl

    el.addEventListener("load", () => { loading = false })
    el.addEventListener("error", () => { loading = false; error = true })

    const poll = setInterval(() => {
      if (el.contentDocument?.readyState === "complete") {
        clearInterval(poll)
        loading = false
      }
    }, 500)

    return () => {
      el.removeEventListener("load", () => {})
      el.removeEventListener("error", () => {})
      clearInterval(poll)
    }
  })
</script>

<div class="xpra-tile">
  <div class="xpra-header">
    <span class="xpra-title">Xpra — {display}</span>
    <div class="xpra-controls">
      <a href={srcUrl} target="_blank" rel="noopener" class="xpra-btn" title="Open in new tab">⧉</a>
    </div>
  </div>

  {#if loading}
    <div class="xpra-loading">
      <div class="xpra-spinner"></div>
      <span>Connecting to Xpra session {display}...</span>
    </div>
  {/if}

  {#if error}
    <div class="xpra-error">
      <p>Failed to connect to Xpra at {httpUrl}</p>
      <p class="xpra-hint">Make sure xpra is installed and the server is running.</p>
      <button onclick={() => { error = false; loading = true }}>
        Retry
      </button>
    </div>
  {/if}

  <iframe
    bind:this={iframeEl}
    src={srcUrl}
    title="Xpra session {display}"
    allow="clipboard-read; clipboard-write"
    sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
    class:hidden={loading || error}
  ></iframe>
</div>

<style>
  .xpra-tile {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    background: #1a1a1a;
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
  }
  .xpra-title {
    font-family: monospace;
    color: #aaa;
  }
  .xpra-controls {
    display: flex;
    gap: 4px;
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
  iframe {
    flex: 1;
    border: none;
    width: 100%;
  }
  iframe.hidden {
    display: none;
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
</style>