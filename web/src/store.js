import { reactive, ref } from 'vue'
import { getToken, api } from './api'

/** 当前登录用户信息(登录 / 改资料后更新,供全局使用) */
export const user = reactive({
  username: '',
  nickname: '',
  avatar: '',
  mustChangePassword: false,
  totpEnabled: false,
})

// ---------------- 主题(亮色 / 暗色) ----------------

const THEME_KEY = 'dm_theme'
export const theme = ref(localStorage.getItem(THEME_KEY) || 'dark')

export function applyTheme(t) {
  theme.value = t
  localStorage.setItem(THEME_KEY, t)
  document.documentElement.dataset.theme = t
}

export function initTheme() {
  applyTheme(localStorage.getItem(THEME_KEY) || 'dark')
}

export function toggleTheme() {
  applyTheme(theme.value === 'dark' ? 'light' : 'dark')
}

/**
 * 主题切换 + 圆形扩散过渡(复刻 Valaxy 的 toggleDarkWithTransition):
 * 从点击位置圆形展开/收缩,浏览器不支持 View Transitions API 时直接切换
 */
export function toggleThemeWithTransition(event) {
  const next = theme.value === 'dark' ? 'light' : 'dark'
  if (!document.startViewTransition) {
    applyTheme(next)
    return
  }
  const x = event?.clientX ?? innerWidth / 2
  const y = event?.clientY ?? innerHeight / 2
  const endRadius = Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y))

  const transition = document.startViewTransition(() => {
    applyTheme(next)
  })

  transition.ready.then(() => {
    const clipPath = [
      `circle(0px at ${x}px ${y}px)`,
      `circle(${endRadius}px at ${x}px ${y}px)`,
    ]
    // 进入 dark:旧画面从点击处收缩消失;进入 light:新画面从点击处扩散出现
    document.documentElement.animate(
      {
        clipPath: theme.value === 'dark' ? clipPath.reverse() : clipPath,
      },
      {
        duration: 300,
        easing: 'ease-in',
        pseudoElement: theme.value === 'dark' ? '::view-transition-old(root)' : '::view-transition-new(root)',
      },
    )
  })
}

export function applyUser(u) {
  user.username = u.username || ''
  user.nickname = u.nickname || ''
  user.avatar = u.avatar || ''
  user.mustChangePassword = !!u.must_change_password
  user.totpEnabled = !!u.totp_enabled
}

// ---------- 终端配置(全局共享,终端 tab 与配置 tab 同步) ----------
export const termSettings = reactive({
  font_family: "JetBrains Mono, Consolas, 'Courier New', monospace",
  font_size: 13,
  background: '#0a0d13',
  foreground: '#e5e7eb',
  cursor_blink: true,
  scrollback: 2000,
  default_shell: '/bin/sh',
})

export function resetUser() {
  applyUser({})
}

// ---------- 许可证状态(全局共享) ----------
export const licenseActive = ref(false)
export const licenseInfo = ref(null)

export async function loadLicense() {
  try {
    const r = await api('/license')
    licenseActive.value = !!r.active
    licenseInfo.value = r.info
    return r
  } catch {
    licenseActive.value = false
    licenseInfo.value = null
    return null
  }
}

export function displayName() {
  return user.nickname || user.username || 'admin'
}

/** 头像 URL:<img> 无法带 Authorization 头,复用 WS 的 ?token= 认证方式;未设置时默认使用 bg.jpg */
export function avatarUrl() {
  if (!user.avatar) return '/bg.jpg'
  const t = getToken()
  return `/api/avatar/${user.avatar}?token=${encodeURIComponent(t || '')}`
}
