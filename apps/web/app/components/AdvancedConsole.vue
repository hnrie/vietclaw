<script setup lang="ts">
import { X, Database, Server, DollarSign, Radio, FileText, History, Sliders } from '@lucide/vue'

const props = defineProps<{ open: boolean }>()
defineEmits<{ close: [] }>()

const tabs = [
  { id: 'sessions', label: 'Phiên', icon: History },
  { id: 'memory', label: 'Memory', icon: Database },
  { id: 'settings', label: 'Cài đặt', icon: Sliders },
  { id: 'providers', label: 'Providers', icon: Server },
  { id: 'budget', label: 'Budget', icon: DollarSign },
  { id: 'channels', label: 'Kênh', icon: Radio },
  { id: 'logs', label: 'Logs', icon: FileText },
] as const

const activeTab = ref<string>('memory')
const { status, framework, online, refresh } = useDaemon()
const { load: loadSettings, dirty, saving, save } = useSettings()

watch(() => props.open, (v) => {
  if (v) {
    refresh()
    void loadSettings()
  }
})
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="open" class="fixed inset-0 z-50 flex justify-end">
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="$emit('close')" />
        <aside class="relative flex h-full w-full max-w-2xl flex-col border-l border-zinc-800 bg-zinc-950 shadow-2xl">
          <header class="flex items-center justify-between border-b border-zinc-800 px-5 py-4">
            <div>
              <h2 class="text-sm font-semibold text-zinc-100">Công cụ nâng cao</h2>
              <p class="text-[11px] text-zinc-500">
                <span
                  class="inline-flex items-center gap-1.5"
                  :class="online ? 'text-emerald-400/80' : 'text-zinc-500'"
                >
                  <span class="h-1.5 w-1.5 rounded-full" :class="online ? 'bg-emerald-500' : 'bg-zinc-600'" />
                  {{ online ? 'Daemon online' : 'Chưa kết nối' }}
                </span>
                <span v-if="status?.version" class="text-zinc-600"> · {{ status.version }}</span>
              </p>
            </div>
            <div class="flex items-center gap-2">
              <span v-if="dirty" class="text-[10px] text-amber-400/90">Chưa lưu</span>
              <button
                v-if="dirty"
                type="button"
                class="rounded-md bg-zinc-100 px-2.5 py-1 text-[11px] font-semibold text-zinc-950 hover:bg-white disabled:opacity-40"
                :disabled="saving"
                @click="save()"
              >
                Lưu
              </button>
              <button class="rounded p-1.5 text-zinc-500 hover:bg-zinc-900 hover:text-zinc-300" @click="$emit('close')">
                <X :size="18" />
              </button>
            </div>
          </header>

          <div class="flex gap-1 overflow-x-auto border-b border-zinc-800/80 px-4 py-2 vc-scrollbar">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              class="flex items-center gap-1.5 rounded-md px-3 py-2 text-[11px] font-medium whitespace-nowrap transition-colors"
              :class="activeTab === tab.id ? 'bg-zinc-900 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'"
              @click="activeTab = tab.id"
            >
              <component :is="tab.icon" :size="13" />
              {{ tab.label }}
            </button>
          </div>

          <div class="flex-1 overflow-y-auto p-4 md:p-6 vc-scrollbar">
            <div v-if="activeTab === 'sessions'" class="h-full min-h-[320px]">
              <SessionsView />
            </div>
            <div v-else-if="activeTab === 'memory'">
              <MemoryView />
            </div>
            <div v-else-if="activeTab === 'settings'">
              <SettingsView />
            </div>
            <div v-else-if="activeTab === 'providers'">
              <ProvidersView />
            </div>
            <div v-else-if="activeTab === 'budget'">
              <BudgetView />
            </div>
            <div v-else-if="activeTab === 'channels'">
              <ChannelsView />
            </div>
            <div v-else-if="activeTab === 'logs'">
              <LogsView />
            </div>
          </div>

          <footer v-if="framework && activeTab === 'settings'" class="border-t border-zinc-800/80 px-5 py-2 text-[10px] text-zinc-600 font-mono">
            framework: delegate {{ framework.delegate_enabled ? 'on' : 'off' }} · hooks {{ framework.hooks_enabled ? 'on' : 'off' }} ({{ framework.hooks_registered ?? 0 }})
          </footer>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer-enter-active,
.drawer-leave-active {
  transition: opacity 0.2s ease;
}
.drawer-enter-active aside,
.drawer-leave-active aside {
  transition: transform 0.25s ease;
}
.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}
.drawer-enter-from aside,
.drawer-leave-to aside {
  transform: translateX(100%);
}
</style>
