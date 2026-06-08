import type { ChatResponse } from '~/types'
import { apiFetch } from '~/utils/api'

export type ChatStepEvent = {
  type: 'tool_call' | 'tool_result' | 'error' | 'done'
  text?: string
  toolName?: string
  toolInput?: string
  toolResult?: string
  error?: string
}

type SSEEvent = {
  event: string
  data: string
}

export type ChatItem = {
  role: 'user' | 'assistant'
  text: string
  steps: ChatStepEvent[]
  meta?: Pick<ChatResponse, 'intent' | 'provider' | 'model' | 'cost_usd'>
}

export type ChatSession = {
  id: string
  title: string
  persona: string
  messages: ChatItem[]
  sessionId: string
  createdAt: number
}

const STORAGE_KEY = 'vietclaw_chats'
const CONFIG_KEY = 'vietclaw_config'

const sessions = ref<ChatSession[]>([])
const currentSessionId = ref('')
const isGenerating = ref(false)

function normalizeSession(value: unknown): ChatSession | null {
  if (!value || typeof value !== 'object') return null
  const raw = value as Partial<ChatSession>
  const id = typeof raw.id === 'string' && raw.id ? raw.id : `session_${Date.now()}`
  return {
    id,
    title: typeof raw.title === 'string' && raw.title ? raw.title : 'Untitled Session',
    persona: typeof raw.persona === 'string' && raw.persona ? raw.persona : 'general',
    messages: Array.isArray(raw.messages)
      ? raw.messages.map(message => ({
          role: message.role === 'assistant' ? 'assistant' : 'user',
          text: typeof message.text === 'string' ? message.text : '',
          steps: Array.isArray(message.steps) ? message.steps : [],
          meta: message.meta
        }))
      : [],
    sessionId: typeof raw.sessionId === 'string' ? raw.sessionId : '',
    createdAt: typeof raw.createdAt === 'number' ? raw.createdAt : Date.now()
  }
}

function normalizeSessions(value: unknown): ChatSession[] {
  if (!Array.isArray(value)) return []
  return value.map(normalizeSession).filter((session): session is ChatSession => Boolean(session))
}

function loadSessions() {
  if (import.meta.client) {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (raw) sessions.value = normalizeSessions(JSON.parse(raw))
    } catch { sessions.value = [] }
  }
  if (sessions.value.length === 0) createSession()
  else currentSessionId.value = sessions.value[0]?.id ?? ''
}

function saveSessions() {
  if (!import.meta.client) return
  localStorage.setItem(STORAGE_KEY, JSON.stringify(sessions.value))
}

function createSession(persona = 'general') {
  const s: ChatSession = {
    id: `session_${Date.now()}`,
    title: 'Untitled Session',
    persona,
    messages: [],
    sessionId: '',
    createdAt: Date.now()
  }
  sessions.value.unshift(s)
  currentSessionId.value = s.id
  saveSessions()
  return s
}

function currentSession() {
  return sessions.value.find(s => s.id === currentSessionId.value)
}

function switchSession(id: string) {
  currentSessionId.value = id
}

function deleteSession(id: string) {
  sessions.value = sessions.value.filter(s => s.id !== id)
  if (currentSessionId.value === id && sessions.value.length > 0) {
    currentSessionId.value = sessions.value[0]?.id ?? ''
  }
  if (sessions.value.length === 0) createSession()
  saveSessions()
}

function clearSessionMessages() {
  const s = currentSession()
  if (s) {
    s.messages = []
    saveSessions()
  }
}

function loadConfig() {
  if (!import.meta.client) {
    return { apiKey: '', model: 'gemini-2.5-flash-preview-09-2025', temperature: 0.7, persona: 'general', voice: 'Zephyr' }
  }
  try {
    const raw = localStorage.getItem(CONFIG_KEY)
    if (raw) return JSON.parse(raw)
  } catch {}
  return { apiKey: '', model: 'gemini-2.5-flash-preview-09-2025', temperature: 0.7, persona: 'general', voice: 'Zephyr' }
}

function saveConfig(cfg: Record<string, unknown>) {
  if (!import.meta.client) return
  localStorage.setItem(CONFIG_KEY, JSON.stringify(cfg))
}

function parseSSEBlock(block: string): SSEEvent | null {
  const lines = block.split(/\r?\n/)
  let event = 'message'
  const data: string[] = []
  for (const line of lines) {
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
    } else if (line.startsWith('data:')) {
      data.push(line.slice(5).trimStart())
    }
  }
  if (data.length === 0) return null
  return { event, data: data.join('\n') }
}

function applySSEEvent(event: SSEEvent, assistantMsg: ChatItem): boolean {
  if (event.event === 'done') return true
  if (event.event === 'error') {
    const parsed = JSON.parse(event.data)
    assistantMsg.text += `\n\nError: ${parsed.error}`
    assistantMsg.steps.push({ type: 'error', error: parsed.error })
    return true
  }

  const parsed = JSON.parse(event.data)
  if (event.event === 'tool_call') {
    assistantMsg.steps.push({
      type: 'tool_call',
      toolName: parsed.name,
      toolInput: parsed.input
    })
  } else if (event.event === 'tool_result') {
    assistantMsg.steps.push({
      type: 'tool_result',
      toolName: parsed.name,
      toolResult: parsed.result
    })
  } else if (parsed.text) {
    assistantMsg.text += parsed.text
  }
  return false
}

function yieldToUI() {
  return new Promise<void>(resolve => {
    if (typeof window !== 'undefined' && window.requestAnimationFrame) {
      window.requestAnimationFrame(() => resolve())
      return
    }
    setTimeout(resolve, 0)
  })
}

async function sendMessage(text: string) {
  const s = currentSession()
  if (!s || isGenerating.value) return

  s.messages.push({ role: 'user', text, steps: [] })
  if (s.title === 'Untitled Session' && s.messages.length >= 1) {
    s.title = text.slice(0, 30) + (text.length > 30 ? '...' : '')
  }
  saveSessions()

  isGenerating.value = true

  const assistantMsg: ChatItem = { role: 'assistant', text: '', steps: [] }
  s.messages.push(assistantMsg)

  try {
    const res = await fetch('/api/chat/stream', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        session_id: s.sessionId || undefined,
        user_id: 'local',
        channel: 'web',
        message: text,
        mode: 'eco'
      })
    })

    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`)
    }

    const reader = res.body?.getReader()
    if (!reader) throw new Error('No response body')

    const decoder = new TextDecoder()
    let buffer = ''
    let stop = false

    while (!stop) {
      const { done, value } = await reader.read()
      if (done) {
        buffer += decoder.decode()
      } else {
        buffer += decoder.decode(value, { stream: true })
      }

      const blocks = buffer.split(/\r?\n\r?\n/)
      buffer = blocks.pop() || ''
      for (const block of blocks) {
        const event = parseSSEBlock(block)
        if (!event) continue
        try {
          stop = applySSEEvent(event, assistantMsg)
          await yieldToUI()
        } catch {
          assistantMsg.steps.push({ type: 'error', error: 'Invalid stream event' })
          stop = true
        }
        if (stop) break
      }

      if (done) break
    }

    if (s.sessionId === '' && assistantMsg.text) {
      // Session ID should have been set by the backend
    }
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Connection failed.'
    assistantMsg.text = `⚠️ ${msg}`
    assistantMsg.steps.push({ type: 'error', error: msg })
  } finally {
    isGenerating.value = false
    saveSessions()
  }
}

export function useChat() {
  if (sessions.value.length === 0) loadSessions()

  return {
    sessions,
    currentSessionId,
    isGenerating,
    currentSession,
    createSession,
    switchSession,
    deleteSession,
    clearSessionMessages,
    sendMessage,
    loadSessions,
    saveSessions,
    loadConfig,
    saveConfig
  }
}
