import { t } from './i18n'

/** Backend returns i18n keys or English messages; translate keys, keep the rest. */
function errMsg(m) {
  try {
    const s = t(m)
    return s && s !== m ? s : m
  } catch {
    return m
  }
}

const TOKEN_KEY = 'dm_token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}
export function setToken(t) {
  if (t) localStorage.setItem(TOKEN_KEY, t)
  else localStorage.removeItem(TOKEN_KEY)
}

/** 登录前可访问:安全入口(webBasePath)等前端启动配置(不带 token) */
let cachedBase = ''
export async function getPublicConfig() {
  try {
    const res = await fetch('/api/system/public-config', { cache: 'no-store' })
    if (!res.ok) return {}
    const j = await res.json()
    cachedBase = j.basePath && j.basePath !== '/' ? j.basePath.replace(/\/$/, '') : ''
    return j
  } catch {
    return {}
  }
}

/** 安全入口前缀(API/WS/页面跳转统一使用;留空 = 无前缀) */
export function apiBase() {
  return cachedBase
}
export function entrancePath(p) {
  return cachedBase ? cachedBase + p : p
}

export async function api(path, opts = {}) {
  const headers = { ...(opts.headers || {}) }
  if (opts.json !== undefined) headers['Content-Type'] = 'application/json'
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await fetch(entrancePath('/api' + path), { ...opts, headers, body: opts.json !== undefined ? JSON.stringify(opts.json) : opts.body })
  if (res.status === 401) {
    // 登录/2FA 接口的 401 带具体错误消息(如 login.errPwd),优先展示
    let msg = ''
    try { msg = (await res.json()).error || '' } catch { /* ignore */ }
    if (msg) throw new Error(errMsg(msg))
    setToken(null)
    if (!location.pathname.endsWith('/login')) location.href = entrancePath('/login')
    throw new Error(errMsg('err.notLogin'))
  }
  if (!res.ok) {
    let msg
    try { msg = (await res.json()).error } catch { msg = res.statusText }
    throw new Error(errMsg(msg || 'err.requestFailed'))
  }
  return res.json()
}

export function wsUrl(path) {
  const p = entrancePath('/api' + path)
  const sep = p.includes('?') ? '&' : '?'
  return `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}${p}${sep}token=${encodeURIComponent(getToken() || '')}`
}

/** Pull image progress stream (NDJSON) */
export async function pullImageStream(payload, onLine) {
  const headers = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await fetch(entrancePath('/api/images/pull'), { method: 'POST', headers, body: JSON.stringify(payload) })
  if (!res.ok) {
    let msg
    try { msg = (await res.json()).error } catch { msg = res.statusText }
    throw new Error(errMsg(msg || 'err.pullFailed'))
  }
  const reader = res.body.getReader()
  const dec = new TextDecoder()
  let buf = ''
  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buf += dec.decode(value, { stream: true })
    let idx
    while ((idx = buf.indexOf('\n')) >= 0) {
      const line = buf.slice(0, idx)
      buf = buf.slice(idx + 1)
      if (!line.trim()) continue
      try { onLine(JSON.parse(line)) } catch { /* ignore non-JSON lines */ }
    }
  }
}

/**
 * Compose deploy/update progress stream (NDJSON).
 * Line callback `{type:'line',data}`;结束时收到 `{type:'done',ok,error}` → resolve/reject。
 */
export async function composeStream(path, payload, onLine) {
  const headers = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await fetch(entrancePath('/api' + path), { method: 'POST', headers, body: JSON.stringify(payload) })
  if (!res.ok) {
    let msg
    try { msg = (await res.json()).error } catch { msg = res.statusText }
    throw new Error(errMsg(msg || 'err.requestFailed'))
  }
  const reader = res.body.getReader()
  const dec = new TextDecoder()
  let buf = ''
  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buf += dec.decode(value, { stream: true })
    let idx
    while ((idx = buf.indexOf('\n')) >= 0) {
      const line = buf.slice(0, idx)
      buf = buf.slice(idx + 1)
      if (!line.trim()) continue
      let msg
      try { msg = JSON.parse(line) } catch { continue }
      if (msg.type === 'line') {
        onLine(msg.data)
      } else if (msg.type === 'done') {
        if (msg.ok) return
        throw new Error(errMsg(msg.error || 'err.execFailed'))
      }
    }
  }
}

// ---------- System: registry mirrors (daemon.json) ----------
export async function getRegistryMirrors() {
  return api('/system/registry-mirrors')
}

export async function saveRegistryMirrors(mirrors) {
  return api('/system/registry-mirrors', { method: 'PUT', body: JSON.stringify({ mirrors }) })
}

// ---------- License ----------
export async function getLicense() {
  return api('/license')
}

export async function activateLicense(key) {
  return api('/license/activate', { method: 'POST', json: { key } })
}

/** 上传许可文件激活(FormData multipart) */
export async function activateLicenseFile(file) {
  const fd = new FormData()
  fd.append('file', file)
  return api('/license/activate-file', { method: 'POST', body: fd })
}

export async function deactivateLicense() {
  return api('/license/deactivate', { method: 'POST' })
}

// 手动触发一次在线验证(配置了授权服务器时生效;吊销即时触达)
export async function verifyLicense() {
  return api('/license/verify', { method: 'POST' })
}

export async function getDemoKey() {
  return api('/license/demo')
}
