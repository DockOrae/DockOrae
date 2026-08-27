import { reactive } from 'vue'

export const toastState = reactive({
  items: [],
})

let id = 0

export function toast(message, type = 'success', duration = 3200) {
  const item = { id: ++id, message, type }
  toastState.items.push(item)
  setTimeout(() => {
    const i = toastState.items.findIndex((t) => t.id === item.id)
    if (i >= 0) toastState.items.splice(i, 1)
  }, duration)
}

export const toastOk = (m) => toast(m, 'success')
export const toastErr = (m) => toast(m, 'error', 5000)
export const toastInfo = (m) => toast(m, 'info')
