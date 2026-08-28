<template>
  <div class="page">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold">{{ t('volumes.title') }}</h2>
      <button class="btn btn-brand btn-sm" @click="openCreate"><Icon name="plus" size="13" /> {{ t('volumes.createTitle') }}</button>
    </div>

    <div class="panel p-0">
      <table class="table w-full">
        <thead>
          <tr>
            <th class="th">{{ t('volumes.volumeName') }}</th>
            <th class="th">{{ t('volumes.driver') }}</th>
            <th class="th">{{ t('volumes.createdAt') }}</th>
            <th class="th"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="v in volumes" :key="v.Name">
            <td class="td font-medium">{{ v.Name }}</td>
            <td class="td text-muted">{{ v.Driver }}</td>
            <td class="td text-muted text-[12px]">{{ formatDate(v.CreatedAt) }}</td>
            <td class="td">
              <button class="btn btn-icon btn-sm text-danger" :title="t('common.delete')" @click="remove(v)">
                <Icon name="trash" size="13" />
              </button>
            </td>
          </tr>
          <tr v-if="!volumes.length">
            <td colspan="4" class="td text-center text-muted py-10">{{ t('volumes.noVolumes') }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <Modal :model-value="createOpen" :title="t('volumes.createTitle')" @close="createOpen = false">
      <div class="space-y-3 max-w-[480px]">
        <div>
          <label class="label">{{ t('volumes.volumeName') }}</label>
          <input v-model="form.name" class="input" :placeholder="t('volumes.volumeNamePh')" />
        </div>

        <!-- 类型 -->
        <div>
          <label class="label">{{ t('volumes.type') }}</label>
          <div class="flex gap-2">
            <button type="button" class="btn btn-sm flex-1" :class="form.type === 'local' ? 'btn-brand' : 'btn-ghost'" @click="form.type = 'local'">
              {{ t('volumes.typeLocal') }}
            </button>
            <button type="button" class="btn btn-sm flex-1" :class="form.type === 'nfs' ? 'btn-brand' : 'btn-ghost'" @click="form.type = 'nfs'">
              {{ t('volumes.typeNfs') }}
            </button>
            <button type="button" class="btn btn-sm flex-1" :class="form.type === 'custom' ? 'btn-brand' : 'btn-ghost'" @click="form.type = 'custom'">
              {{ t('volumes.typeCustom') }}
            </button>
          </div>
        </div>

        <!-- NFS 设置 -->
        <template v-if="form.type === 'nfs'">
          <div>
            <label class="label">{{ t('volumes.nfsAddress') }}</label>
            <input v-model="form.nfs.address" class="input" :placeholder="t('volumes.nfsAddressPh')" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="label">{{ t('volumes.nfsVersion') }}</label>
              <select v-model="form.nfs.version" class="input">
                <option value="4">4</option>
                <option value="3">3</option>
              </select>
            </div>
            <div>
              <label class="label">{{ t('volumes.nfsMountPoint') }}</label>
              <input v-model="form.nfs.mountpoint" class="input" placeholder="/exports/data" />
            </div>
          </div>
          <div>
            <label class="label">{{ t('volumes.nfsOptions') }}</label>
            <input v-model="form.nfs.options" class="input" :placeholder="t('volumes.nfsOptionsPh')" />
          </div>
        </template>

        <!-- 自定义驱动 -->
        <template v-if="form.type === 'custom'">
          <div>
            <label class="label">{{ t('volumes.driver') }}</label>
            <select v-model="form.custom.driver" class="input">
              <option v-for="d in drivers" :key="d" :value="d">{{ d }}</option>
            </select>
          </div>
        </template>

        <!-- 驱动选项(键值对) -->
        <div>
          <label class="label">{{ t('volumes.driverOpts') }}</label>
          <div class="space-y-1.5">
            <div v-for="(o, i) in form.opts" :key="i" class="flex gap-1.5">
              <input v-model="o.key" class="input !w-1/2" :placeholder="t('volumes.optKey')" />
              <input v-model="o.value" class="input !w-1/2" :placeholder="t('volumes.optValue')" />
              <button type="button" class="btn btn-icon btn-sm text-danger" @click="form.opts.splice(i, 1)"><Icon name="x" size="12" /></button>
            </div>
            <button type="button" class="btn btn-ghost btn-xs" @click="form.opts.push({ key: '', value: '' })">
              <Icon name="plus" size="12" /> {{ t('volumes.addOpt') }}
            </button>
          </div>
        </div>

        <!-- 标签(键值对) -->
        <div>
          <label class="label">{{ t('volumes.labels') }}</label>
          <div class="space-y-1.5">
            <div v-for="(o, i) in form.labels" :key="i" class="flex gap-1.5">
              <input v-model="o.key" class="input !w-1/2" :placeholder="t('volumes.labelKey')" />
              <input v-model="o.value" class="input !w-1/2" :placeholder="t('volumes.labelValue')" />
              <button type="button" class="btn btn-icon btn-sm text-danger" @click="form.labels.splice(i, 1)"><Icon name="x" size="12" /></button>
            </div>
            <button type="button" class="btn btn-ghost btn-xs" @click="form.labels.push({ key: '', value: '' })">
              <Icon name="plus" size="12" /> {{ t('volumes.addLabel') }}
            </button>
          </div>
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
const drivers = ref(['local'])
const createOpen = ref(false)
const error = ref('')
const confirm = useConfirm()

const form = reactive({
  name: '',
  type: 'local',
  nfs: { address: '', version: '4', mountpoint: '', options: 'rw' },
  custom: { driver: 'local' },
  opts: [],
  labels: [],
})

function openCreate() {
  form.name = ''
  form.type = 'local'
  form.nfs = { address: '', version: '4', mountpoint: '', options: 'rw' }
  form.custom = { driver: 'local' }
  form.opts = []
  form.labels = []
  error.value = ''
  createOpen.value = true
}

function kvToObj(pairs) {
  const out = {}
  for (const p of pairs) {
    const k = (p.key || '').trim()
    if (k) out[k] = p.value || ''
  }
  return Object.keys(out).length ? out : undefined
}

function buildPayload() {
  const payload = { name: form.name.trim(), labels: kvToObj(form.labels) }
  if (form.type === 'nfs') {
    // NFS 卷 = local 驱动 + driver_opts(type=nfs, o=addr=..., device=:path)
    if (!form.nfs.address.trim() || !form.nfs.mountpoint.trim()) return null
    payload.driver = 'local'
    const opts = []
    opts.push('addr=' + form.nfs.address.trim())
    if (form.nfs.options.trim()) opts.push(form.nfs.options.trim())
    opts.push('nfsvers=' + form.nfs.version)
    payload.driver_opts = {
      type: 'nfs',
      o: opts.join(','),
      device: ':' + form.nfs.mountpoint.trim(),
    }
  } else if (form.type === 'custom') {
    payload.driver = form.custom.driver || 'local'
    payload.driver_opts = kvToObj(form.opts)
  } else {
    payload.driver = 'local'
    payload.driver_opts = kvToObj(form.opts)
  }
  return payload
}

async function load() {
  volumes.value = await api('/volumes')
}

async function loadDrivers() {
  try {
    const r = await api('/volumes/drivers')
    drivers.value = r.drivers || ['local']
  } catch { /* 插件列表失败不阻塞 */ }
}

async function create() {
  error.value = ''
  const payload = buildPayload()
  if (!payload) {
    error.value = t('volumes.nfsRequired')
    return
  }
  try {
    await api('/volumes', { method: 'POST', json: payload })
    toastOk(t('volumes.toastCreated'))
    createOpen.value = false
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

onMounted(() => {
  load()
  loadDrivers()
})
</script>
