<template>
  <div class="page">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold">{{ t('volumes.count', { count: volumes.length }) }}</h2>
      <Button variant="brand" size="sm" @click="openCreate"><Icon name="plus" size="13" /> {{ t('volumes.createTitle') }}</Button>
    </div>

    <Card class="overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{{ t('volumes.volumeName') }}</TableHead>
            <TableHead>{{ t('volumes.driver') }}</TableHead>
            <TableHead>{{ t('volumes.thCreated') }}</TableHead>
            <TableHead class="w-20"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="v in volumes" :key="v.Name">
            <TableCell class="font-medium">{{ v.Name }}</TableCell>
            <TableCell class="text-muted">{{ v.Driver }}</TableCell>
            <TableCell class="text-muted text-[12px]">{{ formatDate(v.CreatedAt) }}</TableCell>
            <TableCell>
              <Button variant="icon" class="text-danger" :title="t('common.delete')" @click="remove(v)">
                <Icon name="trash" size="13" />
              </Button>
            </TableCell>
          </TableRow>
          <TableRow v-if="!volumes.length">
            <TableCell colspan="4" class="text-center text-muted py-10">{{ t('volumes.noVolumes') }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </Card>

    <Modal :model-value="createOpen" :title="t('volumes.createTitle')" @close="createOpen = false">
      <div class="space-y-3 max-w-[480px]">
        <div>
          <Label>{{ t('volumes.volumeName') }}</Label>
          <Input v-model="form.name" :placeholder="t('volumes.volumeNamePh')" />
        </div>

        <!-- 类型 -->
        <div>
          <Label>{{ t('volumes.type') }}</Label>
          <div class="flex gap-2">
            <Button type="button" size="sm" class="flex-1" :variant="form.type === 'local' ? 'brand' : 'ghost'" @click="form.type = 'local'">
              {{ t('volumes.typeLocal') }}
            </Button>
            <Button type="button" size="sm" class="flex-1" :variant="form.type === 'nfs' ? 'brand' : 'ghost'" @click="form.type = 'nfs'">
              {{ t('volumes.typeNfs') }}
            </Button>
          </div>
        </div>

        <!-- NFS 设置 -->
        <template v-if="form.type === 'nfs'">
          <div>
            <Label>{{ t('volumes.nfsAddress') }}</Label>
            <Input v-model="form.nfs.address" :placeholder="t('volumes.nfsAddressPh')" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <Label>{{ t('volumes.nfsVersion') }}</Label>
              <Select v-model="form.nfs.version">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="4">4</SelectItem>
                  <SelectItem value="3">3</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>{{ t('volumes.nfsMountPoint') }}</Label>
              <Input v-model="form.nfs.mountpoint" placeholder="/exports/data" />
            </div>
          </div>
          <div>
            <Label>{{ t('volumes.nfsOptions') }}</Label>
            <Input v-model="form.nfs.options" :placeholder="t('volumes.nfsOptionsPh')" />
          </div>
        </template>

        <!-- 本地卷提示 -->
        <p v-if="form.type === 'local'" class="text-[11px] text-muted">{{ t('volumes.localHint') }}</p>

        <!-- 驱动选项(仅本地;NFS 由上方字段自动生成 driver_opts) -->
        <template v-if="form.type === 'local'">
          <div>
            <Label>{{ t('volumes.driverOpts') }}</Label>
            <div class="space-y-1.5">
              <div v-for="(o, i) in form.opts" :key="i" class="flex gap-1.5">
                <Input v-model="o.key" class="!w-1/2" :placeholder="t('volumes.optKey')" />
                <Input v-model="o.value" class="!w-1/2" :placeholder="t('volumes.optValue')" />
                <Button type="button" variant="icon" class="text-danger" @click="form.opts.splice(i, 1)"><Icon name="x" size="12" /></Button>
              </div>
              <Button type="button" variant="ghost" @click="form.opts.push({ key: '', value: '' })">
                <Icon name="plus" size="12" /> {{ t('volumes.addOpt') }}
              </Button>
            </div>
          </div>
        </template>

        <!-- 标签(键值对) -->
        <div>
          <Label>{{ t('volumes.labels') }}</Label>
          <div class="space-y-1.5">
            <div v-for="(o, i) in form.labels" :key="i" class="flex gap-1.5">
              <Input v-model="o.key" class="!w-1/2" :placeholder="t('volumes.labelKey')" />
              <Input v-model="o.value" class="!w-1/2" :placeholder="t('volumes.labelValue')" />
              <Button type="button" variant="icon" class="text-danger" @click="form.labels.splice(i, 1)"><Icon name="x" size="12" /></Button>
            </div>
            <Button type="button" variant="ghost" @click="form.labels.push({ key: '', value: '' })">
              <Icon name="plus" size="12" /> {{ t('volumes.addLabel') }}
            </Button>
          </div>
        </div>

        <p v-if="error" class="text-xs text-danger">{{ error }}</p>
      </div>
      <template #footer>
        <Button variant="ghost" size="sm" @click="createOpen = false">{{ t('common.cancel') }}</Button>
        <Button variant="brand" size="sm" :disabled="!form.name" @click="create">{{ t('common.create') }}</Button>
      </template>
    </Modal>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { api } from '../api'
import { formatDate } from '../util'
import { useConfirm } from '../confirm'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const volumes = ref([])
const createOpen = ref(false)
const error = ref('')
const confirm = useConfirm()

const form = reactive({
  name: '',
  type: 'local',
  nfs: { address: '', version: '4', mountpoint: '', options: 'rw' },
  opts: [],
  labels: [],
})

function openCreate() {
  form.name = ''
  form.type = 'local'
  form.nfs = { address: '', version: '4', mountpoint: '', options: 'rw' }
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
    const opts = []
    opts.push('addr=' + form.nfs.address.trim())
    if (form.nfs.options.trim()) opts.push(form.nfs.options.trim())
    opts.push('nfsvers=' + form.nfs.version)
    payload.driver_opts = {
      type: 'nfs',
      o: opts.join(','),
      device: ':' + form.nfs.mountpoint.trim(),
    }
  } else {
    payload.driver_opts = kvToObj(form.opts)
  }
  return payload
}

async function load() {
  volumes.value = await api('/volumes')
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
})
</script>
