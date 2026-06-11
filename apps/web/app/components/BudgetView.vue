<script setup lang="ts">
import { apiFetch, formatMoney } from '~/utils/api'
import { DollarSign, Save } from '@lucide/vue'

const { config, loading, saving, dirty, load, save } = useSettings()
const todayCost = ref<number | null>(null)

async function fetchTodayCost() {
  try {
    const res = await apiFetch<{ total_cost_usd: number }>('/api/budget')
    todayCost.value = res.total_cost_usd
  } catch {
    todayCost.value = null
  }
}

onMounted(async () => {
  if (!config.value) await load()
  await fetchTodayCost()
})
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <DollarSign :size="16" class="text-zinc-400" />
        <h2 class="text-sm font-semibold text-zinc-200">Budget</h2>
      </div>
      <button
        type="button"
        class="flex items-center gap-1.5 rounded-md bg-zinc-100 px-3 py-1.5 text-[11px] font-semibold text-zinc-950 hover:bg-white disabled:opacity-40"
        :disabled="saving || !dirty"
        @click="save()"
      >
        <Save :size="12" />
        Lưu
      </button>
    </div>

    <div v-if="loading" class="h-32 rounded-lg bg-zinc-900/40 animate-pulse" />

    <div v-else-if="config" class="rounded-lg border border-zinc-900 bg-zinc-950/40 overflow-hidden">
      <div class="px-5 py-4 border-b border-zinc-900/60 flex flex-wrap items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <span class="text-xs text-zinc-400">Router</span>
          <label class="flex items-center gap-1.5 text-[10px] text-zinc-500">
            <input type="checkbox" v-model="config.router.cheap_first" class="rounded border-zinc-700" />
            cheap first
          </label>
          <label class="flex items-center gap-1.5 text-[10px] text-zinc-500">
            <input type="checkbox" v-model="config.router.allow_escalation" class="rounded border-zinc-700" />
            escalation
          </label>
        </div>
        <span v-if="todayCost !== null" class="text-[10px] font-mono text-zinc-500">
          hôm nay {{ formatMoney(todayCost) }}
        </span>
      </div>
      <div class="grid grid-cols-2 divide-x divide-zinc-900/60">
        <div class="px-4 py-5">
          <div class="text-[10px] font-medium text-zinc-500 uppercase tracking-wider mb-1.5">daily cap (USD)</div>
          <input
            v-model.number="config.budget.daily_usd_limit"
            type="number"
            min="0"
            step="0.01"
            class="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-lg font-bold font-mono text-zinc-100"
          />
        </div>
        <div class="px-4 py-5">
          <div class="text-[10px] font-medium text-zinc-500 uppercase tracking-wider mb-1.5">approval above (USD)</div>
          <input
            v-model.number="config.budget.require_approval_above_usd"
            type="number"
            min="0"
            step="0.01"
            class="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-lg font-bold font-mono text-zinc-100"
          />
        </div>
      </div>
    </div>
  </div>
</template>
