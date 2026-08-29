import { reactive } from 'vue'
import { t } from './i18n'

export const confirmState = reactive({
  visible: false,
  title: '',
  message: '',
  danger: true,
  confirmText: '',
  size: 'sm', // sm | lg | xl
  _resolve: null,
})

export function useConfirm() {
  return (message, opts = {}) =>
    new Promise((resolve) => {
      confirmState.title = opts.title || t('common.confirm')
      confirmState.message = message
      confirmState.danger = opts.danger !== false
      confirmState.confirmText = opts.confirmText || t('common.ok')
      confirmState.size = opts.size || 'sm'
      confirmState._resolve = resolve
      confirmState.visible = true
    })
}

export function resolveConfirm(val) {
  confirmState.visible = false
  if (confirmState._resolve) confirmState._resolve(val)
  confirmState._resolve = null
}
