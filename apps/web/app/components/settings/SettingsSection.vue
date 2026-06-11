<script setup lang="ts">
const props = defineProps<{
  title: string
  description?: string
  defaultOpen?: boolean
}>()

const open = ref(props.defaultOpen ?? false)
</script>

<template>
  <section class="rounded-lg border border-zinc-800 bg-zinc-900/20">
    <button
      type="button"
      class="flex w-full items-center justify-between gap-3 px-4 py-3 text-left transition-colors hover:bg-zinc-900/40"
      @click="open = !open"
    >
      <div>
        <h3 class="text-xs font-semibold text-zinc-200">{{ title }}</h3>
        <p v-if="description" class="mt-0.5 text-[10px] text-zinc-500">{{ description }}</p>
      </div>
      <span class="text-[10px] text-zinc-600 shrink-0">{{ open ? '−' : '+' }}</span>
    </button>
    <div v-show="open" class="border-t border-zinc-800/80 px-4 py-4 space-y-4">
      <slot />
    </div>
  </section>
</template>
