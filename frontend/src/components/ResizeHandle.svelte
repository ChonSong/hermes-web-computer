<script lang="ts">
  let {
    side = 'left' as 'left' | 'right',
    width,
    minWidth = 200,
    maxWidth = 600,
    onWidthChange,
  }: {
    side?: 'left' | 'right';
    width: number;
    minWidth?: number;
    maxWidth?: number;
    onWidthChange?: () => void;
  } = $props();

  let isDragging = $state(false);
  let startX = $state(0);
  let startWidth = $state(0);

  function onMousedown(e: MouseEvent) {
    e.preventDefault();
    isDragging = true;
    startX = e.clientX;
    startWidth = width;

    const onMousemove = (ev: MouseEvent) => {
      if (!isDragging) return;
      const dx = ev.clientX - startX;
      const delta = side === 'left' ? dx : -dx;
      const newWidth = Math.min(maxWidth, Math.max(minWidth, startWidth + delta));
      width = newWidth;
    };

    const onMouseup = () => {
      isDragging = false;
      onWidthChange?.();
      document.removeEventListener('mousemove', onMousemove);
      document.removeEventListener('mouseup', onMouseup);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    document.addEventListener('mousemove', onMousemove);
    document.addEventListener('mouseup', onMouseup);
    document.body.style.cursor = 'ew-resize';
    document.body.style.userSelect = 'none';
  }
</script>

<div
  class="bg-gray-800 hover:bg-blue-500 transition-colors cursor-ew-resize select-none"
  class:opacity-100={isDragging}
  on:mousedown={onMousedown}
></div>
