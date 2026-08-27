<template>
  <div class="space-y-4 fade-up">
    <div class="flex items-center justify-between">
      <p class="text-[13px] text-muted">{{ t('networks.count', { count: networks.length }) }}</p>
      <button class="btn btn-brand btn-sm" @click="createOpen = true"><Icon name="plus" size="14" /> {{ t('networks.newNetwork') }}</button>
    </div>

    <div class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="table">
          <thead>
            <tr>
              <th class="th">{{ t('networks.thName') }}</th>
              <th class="th">{{ t('networks.thDriver') }}</th>
              <th class="th">{{ t('networks.thScope') }}</th>
              <th class="th">{{ t('networks.thSubnet') }}</th>
              <th class="th">{{ t('networks.thGateway') }}</th>
              <th class="th">{{ t('networks.thContainers') }}</th>
              <th class="th w-20">{{ t('networks.thActions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="n in networks" :key="n.Id">
              <td class="td font-medium">{{ n.Name }}</td>
              <td class="td text-muted">{{ n.Driver }}</td>
              <td class="td text-muted">{{ n.Scope }}</td>
              <td class="td font-mono text-[12px] text-muted">{{ subnet(n) }}</td>
              <td class="td font-mono text-[12px] text-muted">{{ gateway(n) }}</td>
              <td class="td">{{ n.Containers ? Object.keys(n.Containers).length : 0 }}</td>
              <td class="td">
                <button class="btn btn-icon btn-sm text-danger" :title="t('common.delete')" @click="remove(n)">
                  <Icon name="trash" size="13" />
                </button>
              </td>
            </tr>
            <tr v-if="!networks.length">
              <td colspan="7" class="td text-center text-muted py-10">{{ t('networks.noNetworks') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <Modal :model-value="createOpen" :title="t('networks.createTitle')" @close="createOpen = false">
      <div class="space-y-3">
        <div>
          <label class="label">{{ t('networks.networkName') }}</label>
          <input v-model="form.name" class="input" :placeholder="t('networks.networkNamePh')" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="label">{{ t('networks.driver') }}</label>
            <select v-model="form.driver" class="input">
              <option value="bridge">bridge</option>
              <option value="macvlan">macvlan</option>
              <option value="ipvlan">ipvlan</option>
              <option value="overlay">overlay</option>
            </select>
          </div>
          <div class="flex items-end pb-1">
            <label class="flex items-center gap-2 text-[13px] cursor-pointer select-none">
              <input v-model="form.internal" type="checkbox" class="accent-[#ec4899]" /> {{ t('networks.internalOnly') }}
            </label>
          </div>
        </div>
        <div>
          <label class="label">{{ t('networks.subnet') }}</label>
          <input v-model="form.subnet" class="input" :placeholder="t('networks.subnetPh')" />
        </div>
        <div>
          <label class="label">{{ t('networks.gateway') }}</label>
          <input v-model="form.gateway" class="input" :placeholder="t('networks.gatewayPh')" />
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
