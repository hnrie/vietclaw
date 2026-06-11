<script setup lang="ts">
import { apiFetch } from '~/utils/api'
import type { ChannelStatus } from '~/types'
import type { ChannelEnvTest } from '~/types/config'
import { Radio, AlertCircle, Save, CheckCircle2 } from '@lucide/vue'

const toast = useToast()
const { config, loading, saving, dirty, load, save } = useSettings()
const runtime = ref<ChannelStatus[]>([])
const envTests = ref<Record<string, ChannelEnvTest>>({})

const channelIcons: Record<string, string> = {
  discord: 'M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.041-.106 13.107 13.107 0 01-1.872-.892.077.077 0 01-.008-.128 10.2 10.2 0 00.372-.292.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.01c.12.098.246.198.373.292a.077.077 0 01-.006.127 12.299 12.299 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.839 19.839 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03z',
  telegram: 'M11.944 0A12 12 0 000 12a12 12 0 0012 12 12 12 0 0012-12A12 12 0 0012 0a12 12 0 00-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 01.171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.479.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z'
}

function runtimeFor(name: string): ChannelStatus | undefined {
  return runtime.value.find(c => c.name === name)
}

async function fetchRuntime() {
  try {
    runtime.value = await apiFetch<ChannelStatus[]>('/api/channels')
  } catch {
    runtime.value = []
  }
}

async function testToken(channel: 'discord' | 'telegram') {
  try {
    const res = await apiFetch<ChannelEnvTest>(`/api/channels/${channel}/test`, { method: 'POST' })
    envTests.value[channel] = res
    if (!res.env_found) {
      toast.add(`Biến môi trường ${res.token_env} chưa được set`, 'error')
    } else {
      toast.add('Token env OK', 'success')
    }
  } catch (err) {
    toast.add(err instanceof Error ? err.message : 'Test thất bại', 'error')
  }
}

onMounted(async () => {
  if (!config.value) await load()
  await fetchRuntime()
})
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <Radio :size="16" class="text-zinc-400" />
        <h2 class="text-sm font-semibold text-zinc-200">Kênh</h2>
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
      <div v-for="i in 2" :key="i" class="h-20 rounded-lg bg-zinc-900/40 animate-pulse" />
    </div>

    <div v-else-if="config" class="space-y-2">
      <!-- Discord -->
      <div class="rounded-lg border border-zinc-900 bg-zinc-950/40 p-4">
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-zinc-900 border border-zinc-800 flex items-center justify-center">
              <svg class="w-4 h-4 text-zinc-400" viewBox="0 0 24 24" fill="currentColor">
                <path :d="channelIcons.discord" />
              </svg>
            </div>
            <div>
              <h3 class="text-xs font-semibold text-zinc-200">Discord</h3>
              <p v-if="runtimeFor('discord')" class="text-[10px] text-zinc-500">
                {{ runtimeFor('discord')?.running ? 'running' : runtimeFor('discord')?.enabled ? 'enabled' : 'disabled' }}
              </p>
            </div>
          </div>
          <label class="flex items-center gap-2 text-[11px] text-zinc-400">
            <input type="checkbox" v-model="config.channels.discord.enabled" class="rounded border-zinc-700" />
            Bật
          </label>
        </div>
        <div class="mt-3 space-y-2 border-t border-zinc-900/60 pt-3">
          <input
            v-model="config.channels.discord.token_env"
            type="text"
            class="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-[11px] font-mono text-zinc-400"
            placeholder="VIETCLAW_DISCORD_TOKEN"
          />
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="rounded border border-zinc-800 px-2 py-1 text-[10px] text-zinc-500 hover:text-zinc-300"
              @click="testToken('discord')"
            >
              Kiểm tra token
            </button>
            <span v-if="envTests.discord" class="flex items-center gap-1 text-[10px]" :class="envTests.discord.env_found ? 'text-emerald-400' : 'text-rose-400'">
              <CheckCircle2 v-if="envTests.discord.env_found" :size="12" />
              {{ envTests.discord.env_found ? 'env OK' : 'env missing' }}
            </span>
          </div>
          <div v-if="runtimeFor('discord')?.error" class="flex items-start gap-2 p-2.5 rounded bg-rose-950/20 border border-rose-900/20">
            <AlertCircle :size="12" class="text-rose-400 mt-0.5 shrink-0" />
            <span class="text-[11px] text-rose-400">{{ runtimeFor('discord')?.error }}</span>
          </div>
        </div>
      </div>

      <!-- Telegram -->
      <div class="rounded-lg border border-zinc-900 bg-zinc-950/40 p-4">
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-zinc-900 border border-zinc-800 flex items-center justify-center">
              <svg class="w-4 h-4 text-zinc-400" viewBox="0 0 24 24" fill="currentColor">
                <path :d="channelIcons.telegram" />
              </svg>
            </div>
            <div>
              <h3 class="text-xs font-semibold text-zinc-200">Telegram</h3>
              <p v-if="runtimeFor('telegram')" class="text-[10px] text-zinc-500">
                {{ runtimeFor('telegram')?.running ? 'running' : runtimeFor('telegram')?.enabled ? 'enabled' : 'disabled' }}
              </p>
            </div>
          </div>
          <label class="flex items-center gap-2 text-[11px] text-zinc-400">
            <input type="checkbox" v-model="config.channels.telegram.enabled" class="rounded border-zinc-700" />
            Bật
          </label>
        </div>
        <div class="mt-3 space-y-2 border-t border-zinc-900/60 pt-3">
          <input
            v-model="config.channels.telegram.token_env"
            type="text"
            class="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-[11px] font-mono text-zinc-400"
            placeholder="VIETCLAW_TELEGRAM_TOKEN"
          />
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="rounded border border-zinc-800 px-2 py-1 text-[10px] text-zinc-500 hover:text-zinc-300"
              @click="testToken('telegram')"
            >
              Kiểm tra token
            </button>
            <span v-if="envTests.telegram" class="flex items-center gap-1 text-[10px]" :class="envTests.telegram.env_found ? 'text-emerald-400' : 'text-rose-400'">
              <CheckCircle2 v-if="envTests.telegram.env_found" :size="12" />
              {{ envTests.telegram.env_found ? 'env OK' : 'env missing' }}
            </span>
          </div>
          <div v-if="runtimeFor('telegram')?.error" class="flex items-start gap-2 p-2.5 rounded bg-rose-950/20 border border-rose-900/20">
            <AlertCircle :size="12" class="text-rose-400 mt-0.5 shrink-0" />
            <span class="text-[11px] text-rose-400">{{ runtimeFor('telegram')?.error }}</span>
          </div>
        </div>
      </div>

      <!-- Attachments -->
      <div class="rounded-lg border border-zinc-900 bg-zinc-950/40 p-4">
        <div class="flex items-center justify-between gap-3">
          <span class="text-xs font-semibold text-zinc-200">File đính kèm</span>
          <label class="flex items-center gap-2 text-[11px] text-zinc-400">
            <input type="checkbox" v-model="config.channels.attachments.enabled" class="rounded border-zinc-700" />
            Bật
          </label>
        </div>
        <div class="mt-3 grid grid-cols-2 gap-3 border-t border-zinc-900/60 pt-3">
          <div>
            <span class="text-[10px] text-zinc-600 block mb-1">max files</span>
            <input v-model.number="config.channels.attachments.max_files" type="number" min="0" class="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-[11px] font-mono text-zinc-300" />
          </div>
          <div>
            <span class="text-[10px] text-zinc-600 block mb-1">max bytes</span>
            <input v-model.number="config.channels.attachments.max_bytes" type="number" min="0" class="w-full rounded border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-[11px] font-mono text-zinc-300" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
