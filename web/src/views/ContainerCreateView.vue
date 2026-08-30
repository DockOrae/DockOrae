<template>
  <div class="space-y-4 fade-up">
    <div class="flex items-center gap-2">
      <router-link :class="buttonVariants({ variant: 'ghost', size: 'sm' })" to="/containers"><Icon name="x" size="13" /> {{ t('common.back') }}</router-link>
      <h2 class="text-base font-semibold">{{ t('createContainer.title') }}</h2>
    </div>

    <div v-if="error" class="flex items-center gap-2 px-4 py-3 rounded-lg bg-danger/10 border border-danger/30 text-danger text-[13px]">
      <Icon name="alert" size="14" /> {{ error }}
    </div>

    <Card class="p-5 space-y-5">
      <!-- 基础 -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <Label>{{ t('createContainer.nameLabel') }}</Label>
          <Input v-model="form.name" :placeholder="t('createContainer.namePh')" />
        </div>
        <div>
          <Label>{{ t('createContainer.imageLabel') }}</Label>
          <Input v-model="form.image" list="image-list" :placeholder="t('createContainer.imagePh')" />
          <datalist id="image-list">
            <option v-for="img in images" :key="img" :value="img" />
          </datalist>
        </div>
      </div>

      <div>
        <Label>{{ t('createContainer.cmdLabel') }}</Label>
        <Input v-model="form.cmdText" :placeholder="t('createContainer.cmdPh')" />
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <Label>{{ t('createContainer.restartPolicy') }}</Label>
          <Select v-model="form.restart_policy">
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="no">{{ t('createContainer.policyNo') }}</SelectItem>
              <SelectItem value="always">{{ t('createContainer.policyAlways') }}</SelectItem>
              <SelectItem value="unless-stopped">{{ t('createContainer.policyUnless') }}</SelectItem>
              <SelectItem value="on-failure">{{ t('createContainer.policyOnFailure') }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div>
          <Label>{{ t('createContainer.network') }}</Label>
          <Select v-model="form.network">
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="bridge">{{ t('createContainer.netBridge') }}</SelectItem>
              <SelectItem value="host">host</SelectItem>
              <SelectItem value="none">none</SelectItem>
              <SelectItem v-for="n in networks" :key="n.Name" :value="n.Name">{{ n.Name }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="flex items-end gap-4 pb-1">
          <label class="flex items-center gap-2 text-[13px] cursor-pointer select-none">
            <Checkbox v-model="form.tty" /> {{ t('createContainer.tty') }}
          </label>
          <label class="flex items-center gap-2 text-[13px] cursor-pointer select-none">
            <Checkbox v-model="form.privileged" /> {{ t('createContainer.privileged') }}
          </label>
        </div>
      </div>

      <!-- 端口映射 -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <Label class="!mb-0">{{ t('createContainer.portMapping') }}</Label>
          <Button variant="ghost" size="sm" @click="ports.push({ container: '', host: null, host_ip: '0.0.0.0' })">
            <Icon name="plus" size="13" /> {{ t('createContainer.addPort') }}
          </Button>
        </div>
        <div v-for="(p, i) in ports" :key="i" class="flex gap-2 mb-2">
          <Input v-model="p.container" class="!w-36" :placeholder="t('createContainer.containerPortPh')" />
          <Input v-model.number="p.host" type="number" min="1" max="65535" class="!w-36" :placeholder="t('createContainer.hostPortPh')" />
          <Input v-model="p.host_ip" class="flex-1" :placeholder="t('createContainer.hostIpPh')" />
          <Button variant="icon" @click="ports.splice(i, 1)"><Icon name="trash" size="13" /></Button>
        </div>
      </div>

      <!-- 环境变量 -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <Label class="!mb-0">{{ t('createContainer.envVars') }}</Label>
          <Button variant="ghost" size="sm" @click="envs.push({ k: '', v: '' })">
            <Icon name="plus" size="13" /> {{ t('createContainer.addVar') }}
          </Button>
        </div>
        <div v-for="(e, i) in envs" :key="i" class="flex gap-2 mb-2">
          <Input v-model="e.k" class="!w-56" :placeholder="t('createContainer.envKPh')" />
          <Input v-model="e.v" class="flex-1" :placeholder="t('createContainer.envVPh')" />
          <Button variant="icon" @click="envs.splice(i, 1)"><Icon name="trash" size="13" /></Button>
        </div>
      </div>

      <!-- 卷挂载 -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <Label class="!mb-0">{{ t('createContainer.volumeMounts') }}</Label>
          <Button variant="ghost" size="sm" @click="vols.push({ type: 'bind', host: '', volume: '', container: '', mode: 'rw' })">
            <Icon name="plus" size="13" /> {{ t('createContainer.addMount') }}
          </Button>
        </div>
        <div v-for="(v, i) in vols" :key="i" class="flex gap-2 mb-2 flex-wrap">
          <Select v-model="v.type" class="!w-28">
            <SelectTrigger class="!w-28"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="bind">{{ t('createContainer.bindDir') }}</SelectItem>
              <SelectItem value="volume">{{ t('createContainer.dataVolume') }}</SelectItem>
            </SelectContent>
          </Select>
          <Input v-model="v.host" v-if="v.type === 'bind'" class="!w-56" :placeholder="t('createContainer.hostPathPh')" />
          <Input v-model="v.volume" v-else class="!w-56" :placeholder="t('createContainer.volumeNamePh')" />
          <Input v-model="v.container" class="flex-1" :placeholder="t('createContainer.containerPathPh')" />
          <Select v-model="v.mode" class="!w-24">
            <SelectTrigger class="!w-24"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="rw">rw</SelectItem>
              <SelectItem value="ro">ro</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="icon" @click="vols.splice(i, 1)"><Icon name="trash" size="13" /></Button>
        </div>
      </div>

      <div class="flex justify-end gap-2 pt-2 border-t border-line">
        <router-link :class="buttonVariants({ variant: 'ghost' })" to="/containers">{{ t('common.cancel') }}</router-link>
        <Button
          v-if="!licenseActive"
          variant="ghost"
          class="!text-amber-400 border-amber-400/40"
          :title="t('license.requiredHint')"
          @click="$router.push('/settings#license')"
        >
          <Icon name="lock" size="14" /> {{ t('license.required') }}
        </Button>
        <Button v-else variant="brand" :disabled="loading" @click="submit">
          <span v-if="loading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
          {{ loading ? t('createContainer.creating') : t('createContainer.create') }}
        </Button>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card } from '@/components/ui/card'
import { api } from '../api'
import { licenseActive } from '../store'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const router = useRouter()
const form = reactive({ name: '', image: '', cmdText: '', restart_policy: 'no', network: 'bridge', tty: false, privileged: false })
const ports = ref([])
const envs = ref([])
const vols = ref([])
const images = ref([])
const networks = ref([])
const loading = ref(false)
const error = ref('')

onMounted(async () => {
  try {
    const [imgs, nets] = await Promise.all([api('/images'), api('/networks')])
    images.value = imgs.flatMap((i) => i.RepoTags || []).filter(Boolean)
    networks.value = nets.filter((n) => n.Name && !n.Name.startsWith('ingress'))
  } catch (e) {
    toastErr(e.message)
  }
})

async function submit() {
  error.value = ''
  if (!form.image) {
    error.value = t('createContainer.errImage')
    return
  }
  const payload = {
    name: form.name || null,
    image: form.image,
    cmd: form.cmdText.trim() ? form.cmdText.trim().split(/\s+/) : null,
    restart_policy: form.restart_policy,
    network: form.network === 'bridge' ? null : form.network,
    tty: form.tty || null,
    privileged: form.privileged || null,
    ports: ports.value.filter((p) => p.container && p.host),
    env: envs.value.filter((e) => e.k).map((e) => `${e.k}=${e.v}`),
    volumes: vols.value
      .filter((v) => v.container && (v.host || v.volume))
      .map((v) => ({
        host: v.type === 'bind' ? v.host : null,
        volume: v.type === 'volume' ? v.volume : null,
        container: v.container,
        mode: v.mode,
      })),
  }
  loading.value = true
  try {
    const r = await api('/containers', { method: 'POST', json: payload })
    toastOk(t('createContainer.toastCreated'))
    router.push('/containers/' + r.id)
  } catch (e) {
    error.value = e.message
    toastErr(e.message)
  } finally {
    loading.value = false
  }
}
</script>
