<script lang="ts">
  /**
   * ResearchCard — Inline preview cards for URLs and JSON data in chat messages.
   * Phase 4.1: URL cards with favicon + JSON table cards.
   * Dark theme #191919.
   */
  export interface UrlInfo {
    url: string
    title: string
    description?: string
  }

  interface Props {
    urls?: UrlInfo[]
    jsonData?: unknown[]
    jsonHeaders?: string[]
    searchQuery?: string
  }

  let { urls = [], jsonData = undefined, jsonHeaders = undefined, searchQuery = "" }: Props = $props()

  const URL_REGEX = /https?:\/\/[^\s]+/g

  function getDomain(url: string): string {
    try {
      return new URL(url).hostname
    } catch {
      return url
    }
  }

  function getFaviconUrl(url: string): string {
    try {
      const domain = new URL(url).hostname
      return `https://www.google.com/s2/favicons?domain=${domain}&sz=32`
    } catch {
      return ""
    }
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

  function escapeHtml(s: string): string {
    return s
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
  }

  function highlightMatch(text: string, query: string): string {
    if (!query.trim()) return escapeHtml(text)
    const escaped = escapeHtml(text)
    const q = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
    return escaped.replace(new RegExp(q, "gi"), match => `<mark class="bg-yellow-500 text-black rounded px-0.5">${match}</mark>`)
  }

  function formatCellValue(val: unknown): string {
    if (val === null || val === undefined) return "—"
    if (typeof val === "object") return JSON.stringify(val)
    return String(val)
  }
</script>

{#if urls && urls.length > 0}
  <div class="mt-2 flex flex-col gap-2">
    {#each urls as urlInfo}
      {@const domain = getDomain(urlInfo.url)}
      {@const faviconUrl = getFaviconUrl(urlInfo.url)}
      {@const title = urlInfo.title || getTitle(urlInfo.url)}
      <a
        href={urlInfo.url}
        target="_blank"
        rel="noopener noreferrer"
        class="flex items-start gap-3 rounded-lg border border-white/10 bg-[#1a1a2e] p-3 hover:border-blue-500/40 hover:bg-[#1e1e38] transition-colors max-w-[85%] text-decoration-none group"
      >
        {#if faviconUrl}
          <img
            src={faviconUrl}
            alt=""
            class="w-5 h-5 flex-none mt-0.5 rounded"
            onerror={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
          />
        {/if}
        <div class="flex-1 min-w-0">
          <div class="text-white text-sm font-medium truncate group-hover:text-blue-300 transition-colors">
            {#if searchQuery}
              {@html highlightMatch(title, searchQuery)}
            {:else}
              {title}
            {/if}
          </div>
          <div class="text-blue-400/70 text-xs mt-0.5 truncate">{domain}</div>
          {#if urlInfo.description}
            <div class="text-gray-400 text-xs mt-1 truncate">{urlInfo.description}</div>
          {/if}
        </div>
        <svg class="w-4 h-4 text-gray-500 flex-none mt-0.5 group-hover:text-blue-400 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </a>
    {/each}
  </div>
{/if}

{#if jsonData && jsonHeaders && jsonData.length > 0}
  <div class="mt-2 rounded border border-green-500/30 bg-[#0d2017] overflow-hidden max-w-[85%]">
    <div class="px-3 py-1.5 border-b border-white/10 flex items-center gap-2">
      <svg class="w-3 h-3 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" stroke-linecap="round"/>
      </svg>
      <span class="text-green-400 text-xs font-mono font-semibold">JSON Data</span>
      <span class="text-gray-500 text-xs">{jsonData.length} rows</span>
    </div>
    <div class="overflow-x-auto">
      <table class="w-full text-xs">
        <thead>
          <tr class="border-b border-white/10">
            {#each jsonHeaders as header}
              <th class="px-3 py-2 text-left text-green-400 font-mono font-semibold whitespace-nowrap">{header}</th>
            {/each}
          </tr>
        </thead>
        <tbody>
          {#each jsonData as row, ri}
            <tr class="border-b border-white/5 {ri % 2 === 0 ? 'bg-white/[0.02]' : 'bg-transparent'}">
              {#each jsonHeaders as header}
                <td class="px-3 py-1.5 text-gray-300 font-mono whitespace-nowrap">
                  {#if searchQuery}
                    {@html highlightMatch(formatCellValue((row as Record<string, unknown>)[header]), searchQuery)}
                  {:else}
                    {formatCellValue((row as Record<string, unknown>)[header])}
                  {/if}
                </td>
              {/each}
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
{/if}