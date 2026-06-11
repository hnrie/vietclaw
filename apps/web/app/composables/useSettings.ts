import type { VietClawConfig, SettingsPutResponse } from '~/types/config'
import { apiFetch } from '~/utils/api'

export function useSettings() {
  const config = useState<VietClawConfig | null>('vietclawServerConfig', () => null)
  const snapshot = useState('vietclawServerConfigSnapshot', () => '')
  const loading = useState('vietclawSettingsLoading', () => false)
  const saving = useState('vietclawSettingsSaving', () => false)

  const dirty = computed(() => {
    if (!config.value) return false
    return JSON.stringify(config.value) !== snapshot.value
  })

  function applyLoaded(cfg: VietClawConfig) {
    config.value = cfg
    snapshot.value = JSON.stringify(cfg)
  }

  async function load() {
    loading.value = true
    try {
      const cfg = await apiFetch<VietClawConfig>('/api/settings')
      applyLoaded(cfg)
    } finally {
      loading.value = false
    }
  }

  async function save(): Promise<boolean> {
    if (!config.value) return false
    saving.value = true
    const toast = useToast()
    try {
      const res = await apiFetch<SettingsPutResponse>('/api/settings', {
        method: 'PUT',
        body: JSON.stringify(config.value)
      })
      applyLoaded(res.config)
      toast.add('Đã lưu cấu hình', 'success')
      return true
    } catch (err) {
      toast.add(err instanceof Error ? err.message : 'Lưu cấu hình thất bại', 'error')
      return false
    } finally {
      saving.value = false
    }
  }

  async function reload(): Promise<boolean> {
    saving.value = true
    const toast = useToast()
    try {
      const res = await apiFetch<SettingsPutResponse>('/api/settings/reload', { method: 'POST' })
      applyLoaded(res.config)
      toast.add('Đã tải lại từ config.json', 'success')
      return true
    } catch (err) {
      toast.add(err instanceof Error ? err.message : 'Tải lại thất bại', 'error')
      return false
    } finally {
      saving.value = false
    }
  }

  function discard() {
    if (!snapshot.value) return
    config.value = JSON.parse(snapshot.value) as VietClawConfig
  }

  return { config, loading, saving, dirty, load, save, reload, discard }
}
