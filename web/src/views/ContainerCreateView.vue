<template>
  <div class="space-y-4 fade-up">
    <div class="flex items-center gap-2">
      <router-link to="/containers" class="btn btn-ghost btn-sm"><Icon name="x" size="13" /> {{ t('common.back') }}</router-link>
      <h2 class="text-base font-semibold">{{ t('createContainer.title') }}</h2>
    </div>

    <div v-if="error" class="flex items-center gap-2 px-4 py-3 rounded-lg bg-danger/10 border border-danger/30 text-danger text-[13px]">
      <Icon name="alert" size="14" /> {{ error }}
    </div>

    <div class="card p-5 space-y-5">
      <!-- 基础 -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="label">{{ t('createContainer.nameLabel') }}</label>
          <input v-model="form.name" class="input" :placeholder="t('createContainer.namePh')" />
        </div>
        <div>
          <label class="label">{{ t('createContainer.imageLabel') }}</label>
          <input v-model="form.image" class="input" list="image-list" :placeholder="t('createContainer.imagePh')" />
          <datalist id="image-list">
            <option v-for="img in images" :key="img" :value="img" />
          </datalist>
        </div>
      </div>

      <div>
        <label class="label">{{ t('createContainer.cmdLabel') }}</label>
        <input v-model="form.cmdText" class="input" :placeholder="t('createContainer.cmdPh')" />
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <label class="label">{{ t('createContainer.restartPolicy') }}</label>
          <select v-model="form.restart_policy" class="input">
            <option value="no">{{ t('createContainer.policyNo') }}</option>
            <option value="always">{{ t('createContainer.policyAlways') }}</option>
            <option value="unless-stopped">{{ t('createContainer.policyUnless') }}</option>
            <option value="on-failure">{{ t('createContainer.policyOnFailure') }}</option>
          </select>
        </div>
        <div>
          <label class="label">{{ t('createContainer.network') }}</label>
          <select v-model="form.network" class="input">
            <option value="bridge">{{ t('createContainer.netBridge') }}</option>
            <option value="host">host</option>
            <option value="none">none</option>
            <option v-for="n in networks" :key="n.Name" :value="n.Name">{{ n.Name }}</option>
          </select>
        </div>
        <div class="flex items-end gap-4 pb-1">
          <label class="flex items-center gap-2 text-[13px] cursor-pointer select-none">
            <input v-model="form.tty" type="checkbox" class="accent-[#ec4899]" /> {{ t('createContainer.tty') }}
          </label>
          <label class="flex items-center gap-2 text-[13px] cursor-pointer select-none">
            <input v-model="form.privileged" type="checkbox" class="accent-[#ec4899]" /> {{ t('createContainer.privileged') }}
          </label>
        </div>
      </div>

      <!-- 端口映射 -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <label class="label !mb-0">{{ t('createContainer.portMapping') }}</label>
          <button class="btn btn-ghost btn-sm" @click="ports.push({ container: '', host: null, host_ip: '0.0.0.0' })">
            <Icon name="plus" size="13" /> {{ t('createContainer.addPort') }}
          </button>
        </div>
        <div v-for="(p, i) in ports" :key="i" class="flex gap-2 mb-2">
          <input v-model="p.container" class="input !w-36" :placeholder="t('createContainer.containerPortPh')" />
          <input v-model.number="p.host" type="number" min="1" max="65535" class="input !w-36" :placeholder="t('createContainer.hostPortPh')" />
          <input v-model="p.host_ip" class="input flex-1" :placeholder="t('createContainer.hostIpPh')" />
          <button class="btn btn-icon btn-sm" @click="ports.splice(i, 1)"><Icon name="trash" size="13" /></button>
        </div>
      </div>

      <!-- 环境变量 -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <label class="label !mb-0">{{ t('createContainer.envVars') }}</label>
          <button class="btn btn-ghost btn-sm" @click="envs.push({ k: '', v: '' })">
            <Icon name="plus" size="13" /> {{ t('createContainer.addVar') }}
          </button>
        </div>
        <div v-for="(e, i) in envs" :key="i" class="flex gap-2 mb-2">
          <input v-model="e.k" class="input !w-56" :placeholder="t('createContainer.envKPh')" />
          <input v-model="e.v" class="input flex-1" :placeholder="t('createContainer.envVPh')" />
          <button class="btn btn-icon btn-sm" @click="envs.splice(i, 1)"><Icon name="trash" size="13" /></button>
        </div>
      </div>

      <!-- 卷挂载 -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <label class="label !mb-0">{{ t('createContainer.volumeMounts') }}</label>
          <button class="btn btn-ghost btn-sm" @click="vols.push({ type: 'bind', host: '', volume: '', container: '', mode: 'rw' })">
            <Icon name="plus" size="13" /> {{ t('createContainer.addMount') }}
          </button>
        </div>
        <div v-for="(v, i) in vols" :key="i" class="flex gap-2 mb-2 flex-wrap">
          <select v-model="v.type" class="input !w-28">
            <option value="bind">{{ t('createContainer.bindDir') }}</option>
            <option value="volume">{{ t('createContainer.dataVolume') }}</option>
          </select>
          <input v-model="v.host" v-if="v.type === 'bind'" class="input !w-56" :placeholder="t('createContainer.hostPathPh')" />
          <input v-model="v.volume" v-else class="input !w-56" :placeholder="t('createContainer.volumeNamePh')" />
          <input v-model="v.container" class="input flex-1" :placeholder="t('createContainer.containerPathPh')" />
          <select v-model="v.mode" class="input !w-24">
            <option value="rw">rw</option>
            <option value="ro">ro</option>
          </select>
          <button class="btn btn-icon btn-sm" @click="vols.splice(i, 1)"><Icon name="trash" size="13" /></button>
        </div>
      </div>

      <div class="flex justify-end gap-2 pt-2 border-t border-line">
        <router-link to="/containers" class="btn btn-ghost">{{ t('common.cancel') }}</router-link>
        <button
          v-if="!licenseActive"
          class="btn btn-ghost !text-amber-400 border border-amber-400/40"
          :title="t('license.requiredHint')"
          @click="$router.push('/settings#license')"
        >
          <Icon name="lock" size="14" /> {{ t('license.required') }}
        </button>
        <button v-else class="btn btn-brand" :disabled="loading" @click="submit">
          <span v-if="loading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
          {{ loading ? t('createContainer.creating') : t('createContainer.create') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
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
