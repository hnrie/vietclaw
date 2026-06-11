<script setup lang="ts">
import { apiFetch } from '~/utils/api'
import { Server, Code, Save } from '@lucide/vue'

const toast = useToast()
const { config, loading, saving, dirty, load, save } = useSettings()
const modelLists = ref<Record<string, string[]>>({})
const loadingModels = ref<Record<string, boolean>>({})

async function fetchModels(providerId: string) {
  if (modelLists.value[providerId] || loadingModels.value[providerId]) return
  loadingModels.value[providerId] = true
  try {
    const res = await apiFetch<{ models: string[] }>(`/api/providers/${providerId}/models`)
    modelLists.value[providerId] = res.models || []
  } catch {
    toast.add('Không lấy được danh sách model', 'error')
  } finally {
    loadingModels.value[providerId] = false
  }
}

onMounted(async () => {
  if (!config.value) await load()
})
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <Server :size="16" class="text-zinc-400" />
        <h2 class="text-sm font-semibold text-zinc-200">Providers</h2>
        <span v-if="config" class="text-[10px] text-zinc-500 font-mono">{{ config.providers.length }} configured</span>
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

    <div v-if="loading" class="space-y-3">
      <div v-for="i in 3" :key="i" class="h-24 rounded-lg bg-zinc-900/40 animate-pulse" />
    </div>

    <div v-else-if="config && config.providers.length === 0" class="text-center py-16">
      <Server :size="32" class="mx-auto text-zinc-700 mb-3" />
      <p class="text-xs text-zinc-500">Chưa có provider.</p>
    </div>

    <div v-else-if="config" class="space-y-2">
      <div
        v-for="p in config.providers"
        :key="p.id"
        class="rounded-lg border border-zinc-900 bg-zinc-950/40 p-4 hover:border-zinc-800 transition-colors"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-zinc-900 border border-zinc-800 flex items-center justify-center">
              <Code :size="16" class="text-zinc-400" />
            </div>
            <div>
              <h3 class="text-xs font-semibold text-zinc-200">{{ p.id }}</h3>
              <p class="text-[10px] text-zinc-500 font-mono">{{ p.type }}</p>
            </div>
          </div>
          <label class="flex items-center gap-2 text-[11px] text-zinc-400 shrink-0">
            <input type="checkbox" v-model="p.enabled" class="rounded border-zinc-700" />
            {{ p.enabled ? 'active' : 'off' }}
          </label>
        </div>
        <div class="mt-3 pt-3 border-t border-zinc-900/60 grid gap-3 sm:grid-cols-2">
          <div>
            <span class="text-[10px] text-zinc-600 block mb-1">model</span>
            <div class="flex gap-2">
              <select
                v-if="modelLists[p.id]?.length"
                v-model="p.default_model"
                class="flex-1 rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-[11px] font-mono text-zinc-300"
              >
                <option v-for="m in modelLists[p.id]" :key="m" :value="m">{{ m }}</option>
              </select>
              <input
                v-else
                v-model="p.default_model"
                type="text"
                class="flex-1 rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-[11px] font-mono text-zinc-300"
              />
              <button
                type="button"
                class="rounded border border-zinc-800 px-2 text-[10px] text-zinc-500 hover:text-zinc-300"
                :disabled="loadingModels[p.id]"
                @click="fetchModels(p.id)"
              >
                {{ loadingModels[p.id] ? '…' : 'models' }}
              </button>
            </div>
          </div>
          <div>
            <span class="text-[10px] text-zinc-600 block mb-1">api key env</span>
            <input
              v-model="p.api_key_env"
              type="text"
              placeholder="not needed"
              class="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-[11px] font-mono text-zinc-400"
            />
          </div>
          <div class="sm:col-span-2">
            <span class="text-[10px] text-zinc-600 block mb-1">base url</span>
            <input
              v-model="p.base_url"
              type="text"
              class="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-[11px] font-mono text-zinc-400"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
