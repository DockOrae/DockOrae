<template>
  <div class="card p-5">
    <div class="flex items-center gap-2 mb-4">
      <Icon name="settings" size="16" class="text-brand" />
      <h2 class="text-sm font-semibold">{{ t('terminal.configTitle') }}</h2>
    </div>

    <form class="grid grid-cols-1 md:grid-cols-2 gap-4" @submit.prevent="save">
      <div>
        <label class="label">{{ t('terminal.cfgFontFamily') }}</label>
        <input v-model="form.font_family" class="input" :placeholder="t('terminal.cfgFontFamilyPh')" />
      </div>
      <div>
        <label class="label">{{ t('terminal.cfgFontSize') }}</label>
        <input v-model.number="form.font_size" type="number" min="10" max="24" class="input" />
      </div>
      <div>
        <label class="label">{{ t('terminal.cfgBackground') }}</label>
        <div class="flex items-center gap-2">
          <input v-model="form.background" type="color" class="w-10 h-9 rounded-lg border border-line cursor-pointer bg-surface2 p-1" />
          <input v-model="form.background" class="input font-mono !text-[12px]" placeholder="#0a0d13" spellcheck="false" />
        </div>
      </div>
      <div>
        <label class="label">{{ t('terminal.cfgForeground') }}</label>
        <div class="flex items-center gap-2">
          <input v-model="form.foreground" type="color" class="w-10 h-9 rounded-lg border border-line cursor-pointer bg-surface2 p-1" />
          <input v-model="form.foreground" class="input font-mono !text-[12px]" placeholder="#e5e7eb" spellcheck="false" />
        </div>
      </div>
      <div>
        <label class="label">{{ t('terminal.cfgScrollback') }}</label>
        <input v-model.number="form.scrollback" type="number" min="500" max="10000" step="500" class="input" />
      </div>
      <div>
        <label class="label">{{ t('terminal.cfgDefaultShell') }}</label>
        <select v-model="form.default_shell" class="input cursor-pointer">
          <option value="/bin/sh">/bin/sh</option>
          <option value="/bin/bash">/bin/bash</option>
        </select>
      </div>
      <div class="md:col-span-2 flex items-center gap-3">
        <label class="flex items-center gap-2 cursor-pointer select-none">
          <input v-model="form.cursor_blink" type="checkbox" class="w-4 h-4 accent-pink-500" />
          <span class="text-[13px]">{{ t('terminal.cfgCursorBlink') }}</span>
        </label>
      </div>
      <div class="md:col-span-2 flex items-center gap-3">
        <button class="btn btn-primary btn-sm" :disabled="busy" type="submit">
          <Icon name="save" size="13" />
          {{ t('terminal.saveConfig') }}
        </button>
        <span v-if="msg" class="text-[11px]" :class="ok ? 'text-ok' : 'text-danger'">{{ msg }}</span>
      </div>
    </form>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../Icon.vue'
import { api } from '../../api'
import { termSettings } from '../../store'

const { t } = useI18n()

const form = ref({ ...termSettings })
const busy = ref(false)
const msg = ref('')
const ok = ref(false)

async function load() {
  try {
    const s = await api('/terminal/settings')
    Object.assign(form.value, s)
  } catch { /* 使用默认值 */ }
}

async function save() {
  busy.value = true
  msg.value = ''
  try {
    const r = await api('/terminal/settings', { method: 'PUT', json: form.value })
    Object.assign(termSettings, r.settings)
    ok.value = true
    msg.value = t('terminal.saved')
  } catch (e) {
    ok.value = false
    msg.value = e.message
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>
