<template>
  <div class="card p-5">
    <div class="flex items-center gap-2 mb-4">
      <Icon name="terminal" size="16" class="text-brand" />
      <h2 class="text-sm font-semibold">{{ t('terminal.quickCommand') }}</h2>
      <span class="text-[11px] text-muted">{{ t('terminal.quickCommandDesc') }}</span>
    </div>

    <!-- 新增 -->
    <form class="flex items-end gap-2 mb-4 flex-wrap" @submit.prevent="addCmd">
      <div class="flex-1 min-w-[140px]">
        <label class="label">{{ t('terminal.cmdName') }}</label>
        <input v-model="form.name" class="input" :placeholder="t('terminal.cmdNamePh')" />
      </div>
      <div class="flex-[2] min-w-[220px]">
        <label class="label">{{ t('terminal.cmdCommand') }}</label>
        <input v-model="form.command" class="input font-mono" :placeholder="t('terminal.cmdCommandPh')" spellcheck="false" />
      </div>
      <button class="btn btn-primary btn-sm" :disabled="busy" type="submit">
        <Icon name="plus" size="13" />
        {{ t('terminal.addCommand') }}
      </button>
    </form>

    <!-- 列表 -->
    <div class="overflow-x-auto">
      <table class="table">
        <thead>
          <tr>
            <th class="th w-32">{{ t('terminal.cmdName') }}</th>
            <th class="th">{{ t('terminal.cmdCommand') }}</th>
            <th class="th w-24">{{ t('containers.thActions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in commands" :key="c.id">
            <td class="td font-medium">{{ c.name }}</td>
            <td class="td font-mono text-[12px] text-muted">{{ c.command }}</td>
            <td class="td">
              <button class="btn btn-icon btn-sm text-danger" :title="t('common.delete')" @click="delCmd(c.id)">
                <Icon name="trash" size="13" />
              </button>
            </td>
          </tr>
          <tr v-if="!commands.length">
            <td colspan="3" class="td text-center text-muted py-6">{{ t('terminal.noCommands') }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../Icon.vue'
import { api } from '../../api'
import { toastErr, toastOk } from '../../toast'

const { t } = useI18n()

const commands = ref([])
const busy = ref(false)
const form = ref({ name: '', command: '' })

async function load() {
  try {
    const r = await api('/terminal/quick-commands')
    commands.value = r.commands || []
  } catch (e) {
    toastErr(e.message)
  }
}

async function addCmd() {
  if (!form.value.name.trim() || !form.value.command.trim()) return
  busy.value = true
  try {
    await api('/terminal/quick-commands', { method: 'POST', json: { name: form.value.name.trim(), command: form.value.command.trim() } })
    form.value = { name: '', command: '' }
    toastOk(t('terminal.added'))
    load()
  } catch (e) {
    toastErr(e.message)
  } finally {
    busy.value = false
  }
}

async function delCmd(id) {
  try {
    await api('/terminal/quick-commands/' + id, { method: 'DELETE' })
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

onMounted(load)
</script>
