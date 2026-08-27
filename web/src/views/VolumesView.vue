<template>
  <div class="space-y-4 fade-up">
    <div class="flex items-center justify-between">
      <p class="text-[13px] text-muted">{{ t('volumes.count', { count: volumes.length }) }}</p>
      <button class="btn btn-brand btn-sm" @click="createOpen = true"><Icon name="plus" size="14" /> {{ t('volumes.newVolume') }}</button>
    </div>

    <div class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="table">
          <thead>
            <tr>
              <th class="th">{{ t('volumes.thName') }}</th>
              <th class="th">{{ t('volumes.thDriver') }}</th>
              <th class="th">{{ t('volumes.thMountpoint') }}</th>
              <th class="th">{{ t('volumes.thCreated') }}</th>
              <th class="th w-20">{{ t('volumes.thActions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="v in volumes" :key="v.Name">
              <td class="td font-medium">{{ v.Name }}</td>
              <td class="td text-muted">{{ v.Driver }}</td>
              <td class="td font-mono text-[12px] text-muted">{{ v.Mountpoint }}</td>
              <td class="td text-muted text-[12px]">{{ formatDate(v.CreatedAt) }}</td>
              <td class="td">
                <button class="btn btn-icon btn-sm text-danger" :title="t('common.delete')" @click="remove(v)">
                  <Icon name="trash" size="13" />
                </button>
              </td>
            </tr>
            <tr v-if="!volumes.length">
              <td colspan="5" class="td text-center text-muted py-10">{{ t('volumes.noVolumes') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <Modal :model-value="createOpen" :title="t('volumes.createTitle')" @close="createOpen = false">
      <div class="space-y-3">
        <div>
          <label class="label">{{ t('volumes.volumeName') }}</label>
          <input v-model="form.name" class="input" :placeholder="t('volumes.volumeNamePh')" />
        </div>
        <div>
          <label class="label">{{ t('volumes.driver') }}</label>
          <select v-model="form.driver" class="input">
            <option value="local">local</option>
            <option value="nfs">nfs</option>
          </select>
        </div>
        <p v-if="error" class="text-xs text-danger">{{ error }}</p>
      </div>
      <template #footer>
        <button class="btn btn-ghost btn-sm" @click="createOpen = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-brand btn-sm" :disabled="!form.name" @click="create">{{ t('common.create') }}</button>
      </template>
    </Modal>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'
import { api } from '../api'
import { formatDate } from '../util'
import { useConfirm } from '../confirm'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const volumes = ref([])
const createOpen = ref(false)
const error = ref('')
const form = reactive({ name: '', driver: 'local' })
const confirm = useConfirm()

async function load() {
  volumes.value = await api('/volumes')
}

async function create() {
  error.value = ''
  try {
    await api('/volumes', { method: 'POST', json: { name: form.name, driver: form.driver } })
    toastOk(t('volumes.toastCreated'))
    createOpen.value = false
    form.name = ''
    load()
  } catch (e) {
    error.value = e.message
  }
}

async function remove(v) {
  const ok = await confirm(t('volumes.confirmDelete', { name: v.Name }), {
    title: t('volumes.confirmDeleteTitle'),
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await api(`/volumes/${v.Name}`, { method: 'DELETE' })
    toastOk(t('common.deleted'))
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

onMounted(load)
</script>
