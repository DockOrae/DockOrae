<template>
  <div class="space-y-4 fade-up">
    <div class="flex items-center justify-between">
      <p class="text-[13px] text-muted">{{ t('networks.count', { count: networks.length }) }}</p>
      <Button variant="brand" size="sm" @click="createOpen = true"><Icon name="plus" size="14" /> {{ t('networks.newNetwork') }}</Button>
    </div>

    <Card class="overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{{ t('networks.thName') }}</TableHead>
            <TableHead>{{ t('networks.thDriver') }}</TableHead>
            <TableHead>{{ t('networks.thScope') }}</TableHead>
            <TableHead>{{ t('networks.thSubnet') }}</TableHead>
            <TableHead>{{ t('networks.thGateway') }}</TableHead>
            <TableHead>{{ t('networks.thContainers') }}</TableHead>
            <TableHead class="w-20">{{ t('networks.thActions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="n in networks" :key="n.Id">
            <TableCell class="font-medium">{{ n.Name }}</TableCell>
            <TableCell class="text-muted">{{ n.Driver }}</TableCell>
            <TableCell class="text-muted">{{ n.Scope }}</TableCell>
            <TableCell class="font-mono text-[12px] text-muted">{{ subnet(n) }}</TableCell>
            <TableCell class="font-mono text-[12px] text-muted">{{ gateway(n) }}</TableCell>
            <TableCell>{{ n.Containers ? Object.keys(n.Containers).length : 0 }}</TableCell>
            <TableCell>
              <Button variant="icon" class="text-danger" :title="t('common.delete')" @click="remove(n)">
                <Icon name="trash" size="13" />
              </Button>
            </TableCell>
          </TableRow>
          <TableRow v-if="!networks.length">
            <TableCell colspan="7" class="text-center text-muted py-10">{{ t('networks.noNetworks') }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </Card>

    <Modal :model-value="createOpen" :title="t('networks.createTitle')" @close="createOpen = false">
      <div class="space-y-3">
        <div>
          <Label>{{ t('networks.networkName') }}</Label>
          <Input v-model="form.name" :placeholder="t('networks.networkNamePh')" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <Label>{{ t('networks.driver') }}</Label>
            <Select v-model="form.driver">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="bridge">bridge</SelectItem>
                <SelectItem value="macvlan">macvlan</SelectItem>
                <SelectItem value="ipvlan">ipvlan</SelectItem>
                <SelectItem value="overlay">overlay</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="flex items-end pb-1">
            <label class="flex items-center gap-2 text-[13px] cursor-pointer select-none">
              <Checkbox v-model="form.internal" /> {{ t('networks.internalOnly') }}
            </label>
          </div>
        </div>
        <div>
          <Label>{{ t('networks.subnet') }}</Label>
          <Input v-model="form.subnet" :placeholder="t('networks.subnetPh')" />
        </div>
        <div>
          <Label>{{ t('networks.gateway') }}</Label>
          <Input v-model="form.gateway" :placeholder="t('networks.gatewayPh')" />
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
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { api } from '../api'
import { useConfirm } from '../confirm'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const networks = ref([])
const createOpen = ref(false)
const error = ref('')
const form = reactive({ name: '', driver: 'bridge', subnet: '', gateway: '', internal: false })
const confirm = useConfirm()

const subnet = (n) => n.IPAM?.Config?.[0]?.Subnet || '-'
const gateway = (n) => n.IPAM?.Config?.[0]?.Gateway || '-'

async function load() {
  networks.value = await api('/networks')
}

async function create() {
  error.value = ''
  try {
    await api('/networks', {
      method: 'POST',
      json: { name: form.name, driver: form.driver, subnet: form.subnet || null, gateway: form.gateway || null, internal: form.internal },
    })
    toastOk(t('networks.toastCreated'))
    createOpen.value = false
    form.name = ''
    form.subnet = ''
    form.gateway = ''
    form.internal = false
    load()
  } catch (e) {
    error.value = e.message
  }
}

async function remove(n) {
  const ok = await confirm(t('networks.confirmDelete', { name: n.Name }), {
    title: t('networks.confirmDeleteTitle'),
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await api(`/networks/${n.Id}`, { method: 'DELETE' })
    toastOk(t('common.deleted'))
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

onMounted(load)
</script>
