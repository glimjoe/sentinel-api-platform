<template>
  <div class="diff-viewer">
    <div class="panel">
      <div class="panel-header">Expected</div>
      <pre class="panel-body">{{ formatJSON(expected) }}</pre>
    </div>
    <div class="panel">
      <div class="panel-header">Actual</div>
      <pre class="panel-body">{{ formatJSON(actual) }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ expected?: unknown; actual?: unknown }>()

function formatJSON(v: unknown): string {
  if (v === undefined || v === null) return '(empty)'
  if (typeof v === 'string') {
    try { return JSON.stringify(JSON.parse(v), null, 2) } catch { return v }
  }
  return JSON.stringify(v, null, 2)
}
</script>

<style scoped>
.diff-viewer { display: flex; gap: 8px; }
.panel { flex: 1; border: 1px solid #ddd; border-radius: 4px; overflow: hidden; }
.panel-header { background: #f5f5f5; padding: 4px 8px; font-size: 12px; font-weight: bold; }
.panel-body { padding: 8px; margin: 0; font-size: 12px; max-height: 300px; overflow: auto; white-space: pre-wrap; word-break: break-all; background: #fafafa; }
</style>
