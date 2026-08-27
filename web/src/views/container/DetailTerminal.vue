<template>
  <div class="flex flex-col h-[520px]">
    <div class="flex items-center gap-2 mb-2 flex-wrap">
      <select v-model="shell" class="input !w-32 !py-1.5 !text-xs" :disabled="connected">
        <option value="/bin/sh">/bin/sh</option>
        <option value="/bin/bash">/bin/bash</option>
        <option value="/bin/ash">/bin/ash</option>
        <option value="/bin/zsh">/bin/zsh</option>
        <option value="custom">{{ t('terminal.custom') }}</option>
      </select>
      <input v-if="shell === 'custom'" v-model="customShell" class="input !w-40 !py-1.5 !text-xs" placeholder="/bin/busybox sh" />
      <button class="btn btn-sm" :class="connected ? 'btn-danger' : 'btn-brand'" @click="toggle">
        {{ connected ? t('terminal.disconnect') : t('terminal.connect') }}
      </button>
      <button class="btn btn-ghost btn-sm" @click="clearScreen" :disabled="!term">
        <Icon name="x" size="13" /> {{ t('terminal.clear') }}
      </button>
      <span v-if="error" class="text-xs text-danger">{{ error }}</span>
    </div>
    <div ref="termEl" class="flex-1 bg-[#0a0d13] border border-line rounded-lg overflow-hidden p-2" />
    <p class="text-[11px] text-muted mt-2">
      {{ t('terminal.hint') }}
    </p>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import Icon from '../../components/Icon.vue'
import { wsUrl } from '../../api'
import { toastErr } from '../../toast'

const { t } = useI18n()
const props = defineProps({ id: { type: String, required: true } })

const termEl = ref(null)
const shell = ref('/bin/sh')
const customShell = ref('/bin/sh')
const connected = ref(false)
const error = ref('')
let term = null
let fit = null
let ws = null

function currentShell() {
  return shell.value === 'custom' ? customShell.value || '/bin/sh' : shell.value
}

function connect() {
  disconnect()
  error.value = ''
  try {
    ws = new WebSocket(wsUrl(`/containers/${props.id}/terminal?shell=${encodeURIComponent(currentShell())}`))
  } catch (e) {
    error.value = e.message
    return
  }
  ws.onopen = () => {
    connected.value = true
    sendResize()
  }
  ws.onmessage = (ev) => {
    if (term && typeof ev.data === 'object') {
      term.write(new Uint8Array(ev.data))
    } else if (term && typeof ev.data === 'string') {
      term.write(ev.data)
    }
  }
  ws.onclose = () => {
    connected.value = false
    ws = null
    if (term) term.write(`\r\n\x1b[31m[${t('terminal.disconnected')}]\x1b[0m\r\n`)
  }
  ws.onerror = () => {
    error.value = t('terminal.cantConnect')
    ws?.close()
  }
}

function disconnect() {
  if (ws) {
    ws.onclose = null
    ws.close()
    ws = null
  }
  connected.value = false
}

function sendResize() {
  if (ws && ws.readyState === WebSocket.OPEN && fit) {
    const { cols, rows } = fit.proposeDimensions() || { cols: 80, rows: 24 }
    ws.send(`resize:${cols},${rows}`)
  }
}

function toggle() {
  if (connected.value) disconnect()
  else connect()
}

function clearScreen() {
  term?.clear()
}

onMounted(() => {
  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: 'Consolas, "Cascadia Code", monospace',
    theme: { background: '#0a0d13' },
  })
  fit = new FitAddon()
  term.loadAddon(fit)
  term.open(termEl.value)
  fit.fit()
  term.onData((d) => ws?.send(d))
  window.addEventListener('resize', onResize)
})

function onResize() {
  if (connected.value) {
    fit?.fit()
    sendResize()
  }
}

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  disconnect()
  term?.dispose()
})
</script>
