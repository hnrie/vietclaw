<script setup lang="ts">
import { AlertCircle, ArrowUp, CheckCircle2, Copy, RefreshCw, Terminal, Wrench } from '@lucide/vue'
import { marked } from 'marked'
import hljs from 'highlight.js'

const { currentSession, isGenerating, sendMessage, clearSessionMessages } = useChat()
const toast = useToast()

const chatInput = ref('')
const chatBox = ref<HTMLElement | null>(null)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

marked.setOptions({ breaks: true, gfm: true })

function renderMarkdown(text: string): string {
  try {
    return marked.parse(text) as string
  } catch {
    return text
  }
}

function autoResize(el: HTMLTextAreaElement) {
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 192) + 'px'
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

async function handleSend() {
  const text = chatInput.value.trim()
  if (!text || isGenerating.value) return
  chatInput.value = ''
  if (textareaRef.value) textareaRef.value.style.height = 'auto'
  void sendMessage(text)
  await nextTick()
  scrollToBottom()
}

function scrollToBottom() {
  if (chatBox.value) {
    chatBox.value.scrollTop = chatBox.value.scrollHeight
  }
}

function highlightCode(el: Element) {
  el.querySelectorAll('pre code').forEach((block) => {
    hljs.highlightElement(block as HTMLElement)
  })
}

async function copyMessage(text: string) {
  await window.navigator.clipboard.writeText(text)
  toast.add('Copied', 'success')
}

const session = computed(() => currentSession())
const messages = computed(() => session.value?.messages || [])

watch(
  () => messages.value.map(msg => `${msg.role}:${msg.text.length}:${msg.steps.length}`).join('|'),
  async () => {
    await nextTick()
    scrollToBottom()
  },
  { flush: 'post' }
)
</script>

<template>
  <div class="flex h-full flex-1 flex-col">
    <div
      ref="chatBox"
      class="flex-1 overflow-y-auto p-4 md:p-6 vc-scrollbar"
      @vue:mounted="(el: any) => { if (el?.$el) highlightCode(el.$el) }"
    >
      <div
        v-if="messages.length === 0"
        class="mx-auto flex h-full max-w-xl flex-col justify-center px-4 py-20 text-center"
      >
        <div class="mx-auto mb-4 flex h-10 w-10 items-center justify-center rounded border border-zinc-800 bg-zinc-900/40">
          <Terminal :size="20" class="text-zinc-400" />
        </div>
        <h3 class="mb-1 text-sm font-semibold tracking-tight text-zinc-200">VietClaw Workspace</h3>
        <p class="mx-auto mb-6 max-w-sm text-xs text-zinc-500">Lightweight agent workspace. Ask anything or use tools.</p>

        <div class="mx-auto grid max-w-md grid-cols-1 gap-2 text-left md:grid-cols-2">
          <button
            class="cursor-pointer rounded border border-zinc-900 bg-zinc-950/40 p-3 text-left transition-all hover:border-zinc-700"
            @click="chatInput = 'Giải thích về kiến trúc microservices'; autoResize(textareaRef as HTMLTextAreaElement)"
          >
            <h4 class="flex items-center gap-1.5 text-xs font-mono text-zinc-300">
              <Terminal :size="14" class="text-zinc-500" /> microservices
            </h4>
            <p class="mt-1 text-[10px] text-zinc-500">Giải thích kiến trúc microservices...</p>
          </button>
          <button
            class="cursor-pointer rounded border border-zinc-900 bg-zinc-950/40 p-3 text-left transition-all hover:border-zinc-700"
            @click="chatInput = 'Tìm thông tin về Go release mới nhất'; autoResize(textareaRef as HTMLTextAreaElement)"
          >
            <h4 class="flex items-center gap-1.5 text-xs font-mono text-zinc-300">
              <Terminal :size="14" class="text-zinc-500" /> go_release
            </h4>
            <p class="mt-1 text-[10px] text-zinc-500">Tìm thông tin Go mới nhất...</p>
          </button>
          <button
            class="cursor-pointer rounded border border-zinc-900 bg-zinc-950/40 p-3 text-left transition-all hover:border-zinc-700"
            @click="chatInput = 'Đọc file config.json trong workspace'; autoResize(textareaRef as HTMLTextAreaElement)"
          >
            <h4 class="flex items-center gap-1.5 text-xs font-mono text-zinc-300">
              <Terminal :size="14" class="text-zinc-500" /> read_file
            </h4>
            <p class="mt-1 text-[10px] text-zinc-500">Đọc file từ workspace...</p>
          </button>
          <button
            class="cursor-pointer rounded border border-zinc-900 bg-zinc-950/40 p-3 text-left transition-all hover:border-zinc-700"
            @click="chatInput = 'Tìm file có chứa từ khóa error trong workspace'; autoResize(textareaRef as HTMLTextAreaElement)"
          >
            <h4 class="flex items-center gap-1.5 text-xs font-mono text-zinc-300">
              <Terminal :size="14" class="text-zinc-500" /> grep_files
            </h4>
            <p class="mt-1 text-[10px] text-zinc-500">Tìm kiếm trong workspace...</p>
          </button>
        </div>
      </div>

      <div class="mx-auto max-w-3xl space-y-6">
        <div
          v-for="(msg, idx) in messages"
          :key="idx"
          class="flex gap-4"
          :class="msg.role === 'user' ? 'justify-end' : 'justify-start'"
        >
          <div
            v-if="msg.role === 'assistant'"
            class="flex h-7 w-7 shrink-0 items-center justify-center rounded bg-zinc-100 text-zinc-950"
          >
            <span class="text-[9px] font-bold">AI</span>
          </div>

          <div
            v-if="msg.role === 'user'"
            class="max-w-[85%] rounded border border-zinc-800 bg-zinc-900 px-3.5 py-2.5 text-sm leading-relaxed text-zinc-200"
          >
            <p class="whitespace-pre-wrap">{{ msg.text }}</p>
          </div>

          <div v-else class="max-w-[85%] space-y-2">
            <div v-if="msg.steps.length > 0" class="space-y-1.5">
              <template v-for="(step, si) in msg.steps" :key="si">
                <div
                  v-if="step.type === 'tool_call'"
                  class="flex items-center gap-2 rounded border border-amber-900/20 bg-amber-950/20 px-3 py-1.5 text-[11px]"
                >
                  <Wrench :size="12" class="shrink-0 text-amber-400" />
                  <span class="font-mono font-medium text-amber-300">Bước {{ si + 1 }}</span>
                  <span class="text-zinc-400">đang dùng</span>
                  <span class="font-mono text-zinc-200">{{ step.toolName }}</span>
                </div>
                <div
                  v-else-if="step.type === 'tool_result'"
                  class="flex items-center gap-2 rounded border border-emerald-900/20 bg-emerald-950/20 px-3 py-1.5 text-[11px]"
                >
                  <CheckCircle2 :size="12" class="shrink-0 text-emerald-400" />
                  <span class="text-emerald-300">xong</span>
                  <span class="font-mono text-zinc-300">{{ step.toolName }}</span>
                </div>
                <div
                  v-else-if="step.type === 'error'"
                  class="flex items-center gap-2 rounded border border-rose-900/20 bg-rose-950/20 px-3 py-1.5 text-[11px]"
                >
                  <AlertCircle :size="12" class="shrink-0 text-rose-400" />
                  <span class="text-rose-300">{{ step.error }}</span>
                </div>
              </template>
            </div>

            <div
              v-if="msg.text"
              class="rounded border border-zinc-900 bg-zinc-950/40 px-4 py-3 text-sm prose prose-invert"
              v-html="renderMarkdown(msg.text)"
              v-html-hook="highlightCode"
            />

            <div
              v-if="msg.text || msg.steps.length > 0"
              class="flex items-center gap-3.5 pt-1 text-[10px] text-zinc-500"
            >
              <button
                class="flex items-center gap-1 transition-colors hover:text-zinc-300"
                @click="copyMessage(msg.text)"
              >
                <Copy :size="12" /> [COPY]
              </button>
            </div>
          </div>

          <div
            v-if="msg.role === 'user'"
            class="flex h-7 w-7 shrink-0 items-center justify-center rounded border border-zinc-800 bg-zinc-900"
          >
            <span class="text-[9px] font-semibold text-zinc-500">USR</span>
          </div>
        </div>

        <div v-if="isGenerating" class="flex items-center justify-start gap-4">
          <div class="flex h-7 w-7 shrink-0 items-center justify-center rounded border border-zinc-800 bg-zinc-950">
            <span class="text-[9px] text-zinc-500">SYS</span>
          </div>
          <div class="flex items-center gap-2 rounded border border-zinc-900 bg-zinc-950/20 px-3.5 py-2 text-xs text-zinc-500">
            <div class="flex space-x-1">
              <span class="h-1.5 w-1.5 animate-bounce rounded-full bg-zinc-500" style="animation-delay: 0.1s" />
              <span class="h-1.5 w-1.5 animate-bounce rounded-full bg-zinc-500" style="animation-delay: 0.2s" />
              <span class="h-1.5 w-1.5 animate-bounce rounded-full bg-zinc-500" style="animation-delay: 0.3s" />
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="z-20 border-t border-zinc-800/10 bg-gradient-to-t from-zinc-950/80 to-transparent p-4 md:p-6">
      <div class="relative mx-auto max-w-3xl">
        <div class="flex flex-col rounded-lg border border-zinc-800 bg-zinc-900/30 p-2 transition-colors focus-within:border-zinc-700">
          <textarea
            ref="textareaRef"
            v-model="chatInput"
            rows="1"
            placeholder="Type a message..."
            class="max-h-48 min-h-[32px] w-full resize-none bg-transparent px-2 py-1 text-sm leading-relaxed text-zinc-200 placeholder-zinc-600 focus:outline-none"
            @input="autoResize($event.target as HTMLTextAreaElement)"
            @keydown="onKeydown"
          />

          <div class="mt-1 flex items-center justify-between border-t border-zinc-800/40 px-1 pt-2">
            <div />
            <div class="flex items-center gap-2">
              <button
                class="rounded p-1.5 text-zinc-500 transition-colors hover:bg-zinc-800 hover:text-zinc-300"
                title="Reset Session"
                @click="clearSessionMessages()"
              >
                <RefreshCw :size="14" />
              </button>
              <button
                class="flex h-7 w-7 items-center justify-center rounded bg-zinc-100 text-zinc-950 transition-colors hover:bg-zinc-200 disabled:opacity-30"
                :disabled="isGenerating || !chatInput.trim()"
                @click="handleSend"
              >
                <ArrowUp :size="16" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
export default {
  directives: {
    htmlHook: {
      mounted(el: HTMLElement, binding: any) {
        if (binding.value) binding.value(el)
      },
      updated(el: HTMLElement, binding: any) {
        if (binding.value) binding.value(el)
      }
    }
  }
}
</script>
