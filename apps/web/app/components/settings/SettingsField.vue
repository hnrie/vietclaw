<script setup lang="ts">
defineProps<{
  label: string
  hint?: string
  mono?: boolean
}>()
</script>

<template>
  <div class="space-y-1">
    <label class="text-[10px] font-medium uppercase tracking-wider text-zinc-500">{{ label }}</label>
    <slot />
    <p v-if="hint" class="text-[10px] leading-relaxed text-zinc-600">{{ hint }}</p>
  </div>
</template>
