<script lang="ts">
  /**
   * FileUpload — Drag-drop file upload overlay for ChatPanel.
   * Detects files dropped on the chat area, reads as ArrayBuffer,
   * converts to base64, and writes via fsWrite() to the backend.
   */
  import { fsWrite } from "../stores/ws"

  interface UploadFile {
    id: string
    name: string
    size: number
    path: string | null
    progress: number
    error: string | null
  }

  interface Props {
    onUploadComplete?: (path: string, name: string) => void
    onUploadRemove?: (id: string) => void
  }

  let { onUploadComplete, onUploadRemove }: Props = $props()

  let isDragging = $state(false)
  let uploads = $state<UploadFile[]>([])

  // Counter for unique IDs
  let idCounter = 0

  function generateId(): string {
    return `upload_${++idCounter}_${Date.now()}`
  }

  // Format bytes to human-readable size
  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  // Read file as base64
  function readFileAsBase64(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => {
        const result = reader.result as string
        // Remove data URL prefix if present
        const base64 = result.includes(",") ? result.split(",")[1] : result
        resolve(base64)
      }
      reader.onerror = () => reject(new Error("Failed to read file"))
      reader.readAsDataURL(file)
    })
  }

  // Generate a target path for the uploaded file
  function generateUploadPath(fileName: string): string {
    const sanitized = fileName.replace(/[^a-zA-Z0-9._-]/g, "_")
    const timestamp = Date.now()
    return `/tmp/uploads/${timestamp}_${sanitized}`
  }

  // Upload a single file
  async function uploadFile(file: File): Promise<string> {
    const id = generateId()
    const uploadFile: UploadFile = {
      id,
      name: file.name,
      size: file.size,
      path: null,
      progress: 0,
      error: null,
    }

    uploads = [...uploads, uploadFile]

    try {
      // Read file as base64
      uploadFile.progress = 30
      uploads = [...uploads]

      const base64Content = await readFileAsBase64(file)

      // Generate target path
      uploadFile.progress = 50
      uploads = [...uploads]

      const targetPath = generateUploadPath(file.name)

      // Write to backend via fsWrite (it handles base64 decoding)
      fsWrite(targetPath, base64Content, "base64")

      // Simulate progress since fsWrite is fire-and-forget
      uploadFile.progress = 75
      uploads = [...uploads]

      // Give backend time to write
      await new Promise((r) => setTimeout(r, 200))

      uploadFile.progress = 100
      uploadFile.path = targetPath
      uploads = [...uploads]

      onUploadComplete?.(targetPath, file.name)

      return targetPath
    } catch (err) {
      uploadFile.error = String(err)
      uploads = [...uploads]
      throw err
    }
  }

  // Handle drag events
  export function handleDragOver(e: DragEvent) {
    e.preventDefault()
    e.stopPropagation()
    isDragging = true
  }

  export function handleDragLeave(e: DragEvent) {
    e.preventDefault()
    e.stopPropagation()
    // Only set false if leaving the drop zone entirely
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
    const x = e.clientX
    const y = e.clientY
    if (x < rect.left || x >= rect.right || y < rect.top || y >= rect.bottom) {
      isDragging = false
    }
  }

  export async function handleDrop(e: DragEvent) {
    e.preventDefault()
    e.stopPropagation()
    isDragging = false

    const files = e.dataTransfer?.files
    if (!files || files.length === 0) return

    // Upload all files in parallel
    const uploadPromises: Promise<void>[] = []
    for (let i = 0; i < files.length; i++) {
      uploadPromises.push(
        uploadFile(files[i]).catch(() => {
          // Error already stored in uploadFile.error
        })
      )
    }

    await Promise.all(uploadPromises)
  }

  // Remove an uploaded file from the list
  export function removeUpload(id: string) {
    uploads = uploads.filter((u) => u.id !== id)
    onUploadRemove?.(id)
  }

  // Get files that have been successfully uploaded
  export function getUploadedPaths(): Array<{ path: string; name: string }> {
    return uploads
      .filter((u) => u.path !== null && u.error === null)
      .map((u) => ({ path: u.path!, name: u.name }))
  }
</script>

<!-- Drop zone overlay -->
{#if isDragging}
  <div
    class="absolute inset-0 z-50 flex items-center justify-center bg-[#191919]/90 border-2 border-dashed border-blue-500 rounded-lg pointer-events-auto"
    role="region"
    aria-label="File drop zone"
  >
    <div class="text-center">
      <div class="text-4xl mb-2">📄</div>
      <div class="text-blue-400 text-lg font-medium">Drop files to upload</div>
      <div class="text-gray-400 text-sm mt-1">Files will be saved to /tmp/uploads/</div>
    </div>
  </div>
{/if}

<!-- Upload progress list (shown below messages when uploading) -->
{#if uploads.length > 0}
  <div class="flex flex-wrap gap-2 px-4 py-2 border-t border-white/10 bg-[#191919]">
    {#each uploads as upload (upload.id)}
      <div
        class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-800 border border-white/10 text-xs"
      >
        <span class="text-gray-300 truncate max-w-[120px]">{upload.name}</span>
        <span class="text-gray-500">{formatSize(upload.size)}</span>

        {#if upload.error}
          <span class="text-red-400">❌</span>
          <span class="text-red-400 text-[10px]">Error</span>
        {:else if upload.progress < 100}
          <span class="text-blue-400">{upload.progress}%</span>
          <!-- Progress bar -->
          <div class="w-12 h-1 bg-gray-700 rounded-full overflow-hidden">
            <div
              class="h-full bg-blue-500 transition-all duration-200"
              style="width: {upload.progress}%"
            ></div>
          </div>
        {:else}
          <span class="text-green-400">✓</span>
          <button
            type="button"
            onclick={() => removeUpload(upload.id)}
            class="ml-1 text-gray-500 hover:text-gray-300 transition-colors"
            aria-label="Remove {upload.name}"
          >
            ✕
          </button>
        {/if}
      </div>
    {/each}
  </div>
{/if}