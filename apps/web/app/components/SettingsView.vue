<script setup lang="ts">
import { RefreshCw, Save } from '@lucide/vue'
import SettingsField from '~/components/settings/SettingsField.vue'
import SettingsSection from '~/components/settings/SettingsSection.vue'

const { config, loading, saving, dirty, load, save, reload, discard } = useSettings()

const inputClass = 'w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-200 focus:border-zinc-600 focus:outline-none'
const monoClass = `${inputClass} font-mono`
const selectClass = `${inputClass} cursor-pointer`

function listToText(items: string[] | undefined): string {
  return (items || []).join(', ')
}

function textToList(text: string): string[] {
  return text.split(',').map(s => s.trim()).filter(Boolean)
}

onMounted(() => {
  if (!config.value) void load()
})

async function handleSave() {
  await save()
}

async function handleReload() {
  await reload()
}
</script>

<template>
  <div class="mx-auto max-w-2xl space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 class="text-sm font-semibold text-zinc-200">Cài đặt</h2>
        <p class="text-[11px] text-zinc-500">Lưu vào config.json — daemon áp dụng ngay (trừ port/host).</p>
      </div>
      <div class="flex items-center gap-2">
        <span v-if="dirty" class="text-[10px] text-amber-400/90">Chưa lưu</span>
        <button
          type="button"
          class="flex items-center gap-1.5 rounded-md border border-zinc-800 px-2.5 py-1.5 text-[11px] text-zinc-400 hover:border-zinc-700 hover:text-zinc-200 disabled:opacity-40"
          :disabled="saving"
          @click="discard"
        >
          Hủy
        </button>
        <button
          type="button"
          class="flex items-center gap-1.5 rounded-md border border-zinc-800 px-2.5 py-1.5 text-[11px] text-zinc-400 hover:border-zinc-700 hover:text-zinc-200 disabled:opacity-40"
          :disabled="saving"
          @click="handleReload"
        >
          <RefreshCw :size="12" />
          Tải lại
        </button>
        <button
          type="button"
          class="flex items-center gap-1.5 rounded-md bg-zinc-100 px-3 py-1.5 text-[11px] font-semibold text-zinc-950 hover:bg-white disabled:opacity-40"
          :disabled="saving || !dirty"
          @click="handleSave"
        >
          <Save :size="12" />
          Lưu
        </button>
      </div>
    </div>

    <div v-if="loading" class="space-y-3">
      <div v-for="i in 4" :key="i" class="h-16 rounded-lg bg-zinc-900/50 animate-pulse" />
    </div>

    <template v-else-if="config">
      <SettingsSection title="Agent" description="Hành vi và giới hạn của agent" :default-open="true">
        <div class="grid gap-4 sm:grid-cols-2">
          <SettingsField label="Tên">
            <input v-model="config.agent.name" type="text" :class="inputClass" />
          </SettingsField>
          <SettingsField label="Ngôn ngữ">
            <select v-model="config.agent.language" :class="selectClass">
              <option value="vi">Tiếng Việt</option>
              <option value="en">English</option>
            </select>
          </SettingsField>
          <SettingsField label="Experience">
            <select v-model="config.agent.experience" :class="selectClass">
              <option value="prompt">prompt</option>
              <option value="pro">pro</option>
            </select>
          </SettingsField>
          <SettingsField label="Style">
            <input v-model="config.agent.style" type="text" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Max steps" hint="0 = không giới hạn">
            <input v-model.number="config.agent.max_steps" type="number" min="0" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Max output tokens" hint="0 = không giới hạn">
            <input v-model.number="config.agent.max_output_tokens" type="number" min="0" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Max context chars">
            <input v-model.number="config.agent.max_context_chars" type="number" min="0" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Max history messages">
            <input v-model.number="config.agent.max_history_messages" type="number" min="0" :class="monoClass" />
          </SettingsField>
        </div>
        <div class="flex flex-wrap gap-4 pt-2">
          <label class="flex items-center gap-2 text-xs text-zinc-300">
            <input type="checkbox" v-model="config.agent.reflexion.enabled" class="rounded border-zinc-700" />
            Reflexion
          </label>
          <label class="flex items-center gap-2 text-xs text-zinc-300">
            <input type="checkbox" v-model="config.agent.memory_tools.enabled" class="rounded border-zinc-700" />
            Memory tools
          </label>
          <label class="flex items-center gap-2 text-xs text-zinc-300">
            <input type="checkbox" v-model="config.agent.heartbeat.enabled" class="rounded border-zinc-700" />
            Heartbeat
          </label>
        </div>
        <div v-if="config.agent.heartbeat.enabled" class="grid gap-4 sm:grid-cols-2 border-t border-zinc-800/60 pt-4">
          <SettingsField label="Interval (giây)">
            <input v-model.number="config.agent.heartbeat.interval_seconds" type="number" min="60" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Session ID">
            <input v-model="config.agent.heartbeat.session_id" type="text" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Prompt" class="sm:col-span-2">
            <textarea v-model="config.agent.heartbeat.prompt" rows="3" :class="inputClass" />
          </SettingsField>
        </div>
      </SettingsSection>

      <SettingsSection title="Router" description="Provider và model mặc định">
        <div class="grid gap-4 sm:grid-cols-2">
          <SettingsField label="Default provider">
            <select v-model="config.router.default_provider" :class="selectClass">
              <option v-for="p in config.providers" :key="p.id" :value="p.id">{{ p.id }}</option>
            </select>
          </SettingsField>
          <SettingsField label="Default model">
            <input v-model="config.router.default_model" type="text" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Intent mode">
            <input v-model="config.router.intent_mode" type="text" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Agent routing">
            <input v-model="config.router.agent_routing" type="text" :class="monoClass" />
          </SettingsField>
        </div>
        <div class="flex flex-wrap gap-4">
          <label class="flex items-center gap-2 text-xs text-zinc-300">
            <input type="checkbox" v-model="config.router.cheap_first" class="rounded border-zinc-700" />
            Cheap first
          </label>
          <label class="flex items-center gap-2 text-xs text-zinc-300">
            <input type="checkbox" v-model="config.router.allow_escalation" class="rounded border-zinc-700" />
            Allow escalation
          </label>
        </div>
      </SettingsSection>

      <SettingsSection title="Providers" description="Bật/tắt và cấu hình LLM">
        <div class="space-y-3">
          <div
            v-for="p in config.providers"
            :key="p.id"
            class="rounded-md border border-zinc-800/80 bg-zinc-950/50 p-3 space-y-3"
          >
            <div class="flex items-center justify-between gap-2">
              <div>
                <span class="text-xs font-semibold text-zinc-200">{{ p.id }}</span>
                <span class="ml-2 text-[10px] font-mono text-zinc-500">{{ p.type }}</span>
              </div>
              <label class="flex items-center gap-2 text-[11px] text-zinc-400">
                <input type="checkbox" v-model="p.enabled" class="rounded border-zinc-700" />
                Bật
              </label>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <SettingsField label="Model">
                <input v-model="p.default_model" type="text" :class="monoClass" />
              </SettingsField>
              <SettingsField label="API key env">
                <input v-model="p.api_key_env" type="text" :class="monoClass" placeholder="GEMINI_API_KEY" />
              </SettingsField>
              <SettingsField label="Base URL" class="sm:col-span-2">
                <input v-model="p.base_url" type="text" :class="monoClass" placeholder="https://..." />
              </SettingsField>
            </div>
          </div>
        </div>
      </SettingsSection>

      <SettingsSection title="Budget">
        <div class="grid gap-4 sm:grid-cols-2">
          <SettingsField label="Daily USD limit">
            <input v-model.number="config.budget.daily_usd_limit" type="number" min="0" step="0.01" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Require approval above (USD)">
            <input v-model.number="config.budget.require_approval_above_usd" type="number" min="0" step="0.01" :class="monoClass" />
          </SettingsField>
        </div>
      </SettingsSection>

      <SettingsSection title="Kênh chat" description="Discord, Telegram, file đính kèm">
        <div class="space-y-4">
          <div class="rounded-md border border-zinc-800/80 p-3 space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold text-zinc-200">Discord</span>
              <label class="flex items-center gap-2 text-[11px] text-zinc-400">
                <input type="checkbox" v-model="config.channels.discord.enabled" class="rounded border-zinc-700" />
                Bật
              </label>
            </div>
            <SettingsField label="Token env">
              <input v-model="config.channels.discord.token_env" type="text" :class="monoClass" />
            </SettingsField>
            <SettingsField label="Allowed guilds (id, phẩy)">
              <input
                :value="listToText(config.channels.discord.allowed_guilds)"
                type="text"
                :class="monoClass"
                @input="config.channels.discord.allowed_guilds = textToList(($event.target as HTMLInputElement).value)"
              />
            </SettingsField>
            <label class="flex items-center gap-2 text-xs text-zinc-300">
              <input type="checkbox" v-model="config.channels.discord.respond_in_dm" class="rounded border-zinc-700" />
              Respond in DM
            </label>
          </div>

          <div class="rounded-md border border-zinc-800/80 p-3 space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold text-zinc-200">Telegram</span>
              <label class="flex items-center gap-2 text-[11px] text-zinc-400">
                <input type="checkbox" v-model="config.channels.telegram.enabled" class="rounded border-zinc-700" />
                Bật
              </label>
            </div>
            <SettingsField label="Token env">
              <input v-model="config.channels.telegram.token_env" type="text" :class="monoClass" />
            </SettingsField>
            <SettingsField label="Allowed chats (id, phẩy)">
              <input
                :value="listToText(config.channels.telegram.allowed_chats)"
                type="text"
                :class="monoClass"
                @input="config.channels.telegram.allowed_chats = textToList(($event.target as HTMLInputElement).value)"
              />
            </SettingsField>
            <label class="flex items-center gap-2 text-xs text-zinc-300">
              <input type="checkbox" v-model="config.channels.telegram.respond_in_private" class="rounded border-zinc-700" />
              Respond in private
            </label>
          </div>

          <div class="rounded-md border border-zinc-800/80 p-3 space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold text-zinc-200">Attachments</span>
              <label class="flex items-center gap-2 text-[11px] text-zinc-400">
                <input type="checkbox" v-model="config.channels.attachments.enabled" class="rounded border-zinc-700" />
                Bật
              </label>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <SettingsField label="Max files">
                <input v-model.number="config.channels.attachments.max_files" type="number" min="0" :class="monoClass" />
              </SettingsField>
              <SettingsField label="Max bytes">
                <input v-model.number="config.channels.attachments.max_bytes" type="number" min="0" :class="monoClass" />
              </SettingsField>
            </div>
          </div>
        </div>
      </SettingsSection>

      <SettingsSection title="Tools">
        <div class="space-y-4">
          <div class="flex flex-wrap gap-4">
            <label class="flex items-center gap-2 text-xs text-zinc-300">
              <input type="checkbox" v-model="config.tools.shell.enabled" class="rounded border-zinc-700" />
              Shell
            </label>
            <label class="flex items-center gap-2 text-xs text-zinc-300">
              <input type="checkbox" v-model="config.tools.files.enabled" class="rounded border-zinc-700" />
              Files
            </label>
            <label class="flex items-center gap-2 text-xs text-zinc-300">
              <input type="checkbox" v-model="config.tools.files.workspace_only" class="rounded border-zinc-700" />
              Files workspace only
            </label>
          </div>
          <div v-if="config.tools.shell.enabled" class="grid gap-4 sm:grid-cols-2 border-t border-zinc-800/60 pt-4">
            <SettingsField label="Sandbox">
              <select v-model="config.tools.shell.sandbox" :class="selectClass">
                <option value="none">none</option>
                <option value="docker">docker</option>
              </select>
            </SettingsField>
            <SettingsField label="Workspace mode">
              <select v-model="config.tools.shell.workspace_mode" :class="selectClass">
                <option value="ro">ro</option>
                <option value="rw">rw</option>
              </select>
            </SettingsField>
            <SettingsField label="Docker image">
              <input v-model="config.tools.shell.docker_image" type="text" :class="monoClass" />
            </SettingsField>
            <SettingsField label="Timeout (s)">
              <input v-model.number="config.tools.shell.timeout_seconds" type="number" min="0" :class="monoClass" />
            </SettingsField>
          </div>
        </div>
      </SettingsSection>

      <SettingsSection title="Framework">
        <div class="flex flex-wrap gap-4">
          <label class="flex items-center gap-2 text-xs text-zinc-300">
            <input type="checkbox" v-model="config.framework.enabled" class="rounded border-zinc-700" />
            Enabled
          </label>
          <label class="flex items-center gap-2 text-xs text-zinc-300">
            <input type="checkbox" v-model="config.framework.delegate_enabled" class="rounded border-zinc-700" />
            Delegate
          </label>
          <label class="flex items-center gap-2 text-xs text-zinc-300">
            <input type="checkbox" v-model="config.framework.hooks_enabled" class="rounded border-zinc-700" />
            Hooks
          </label>
        </div>
      </SettingsSection>

      <SettingsSection title="Runtime & Server" description="Thay port/host cần restart daemon">
        <div class="grid gap-4 sm:grid-cols-2">
          <SettingsField label="Mode">
            <input v-model="config.runtime.mode" type="text" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Max concurrent tasks">
            <input v-model.number="config.runtime.max_concurrent_tasks" type="number" min="1" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Host">
            <input v-model="config.server.host" type="text" :class="monoClass" />
          </SettingsField>
          <SettingsField label="Port">
            <input v-model.number="config.server.port" type="number" min="1" max="65535" :class="monoClass" />
          </SettingsField>
        </div>
      </SettingsSection>
    </template>

    <p v-else class="text-center text-xs text-zinc-500 py-8">Không tải được cấu hình. Kiểm tra daemon.</p>
  </div>
</template>
