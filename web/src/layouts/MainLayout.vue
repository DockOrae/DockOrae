<template>
  <div class="app-shell">
    <!-- 左侧折叠侧边栏(仿 3x-ui:72px 窄栏,悬停展开 220px,可图钉固定) -->
    <div
      class="app-sider"
      :class="{ expanded: expanded || pinned, pinned }"
      @mouseenter="onEnter"
      @mouseleave="onLeave"
    >
      <!-- 品牌区 -->
      <div class="sider-brand">
        <div class="brand-block" @click="$router.push('/')">
          <img src="/logo.svg" alt="logo" class="brand-logo" />
          <span v-if="expanded || pinned" class="brand-text">Docker Manager</span>
        </div>
        <div v-if="expanded || pinned" class="brand-actions">
          <button
            type="button"
            class="brand-btn"
            :class="{ active: pinned }"
            :title="t(pinned ? 'nav.unpin' : 'nav.pin')"
            :aria-label="t(pinned ? 'nav.unpin' : 'nav.pin')"
            @click="togglePinned"
          >
            <Icon :name="pinned ? 'pinFilled' : 'pin'" size="14" />
          </button>
          <SwitchAppearance />
        </div>
      </div>

      <!-- 导航菜单 -->
      <nav class="sider-nav">
        <router-link
          v-for="item in navs"
          :key="item.to"
          :to="item.to"
          class="nav-item"
          :class="[
            isActive(item) ? 'active' : '',
            !(expanded || pinned) ? 'is-collapsed' : '',
          ]"
          :title="expanded || pinned ? '' : t(item.labelKey)"
        >
          <Icon :name="item.icon" size="16" class="nav-icon" />
          <span v-if="expanded || pinned" class="nav-label">{{ t(item.labelKey) }}</span>
        </router-link>
      </nav>

      <!-- 底部:登出 + 版本(仿 3x-ui sider-utility) -->
      <div class="sider-footer">
        <button type="button" class="logout-item" :class="{ 'is-collapsed': !expanded }" @click="logout">
          <Icon name="logout" size="15" class="nav-icon" />
          <span v-if="expanded" class="nav-label">{{ t('nav.logout') }}</span>
        </button>
        <a
          href="https://github.com/MinimaxFlora/Docker_Manager_Go"
          target="_blank"
          rel="noopener"
          class="sider-version"
          :class="{ 'is-collapsed': !expanded }"
          :title="t('app.version')"
        >
          <Icon name="github" size="13" filled />
          <span v-if="expanded" class="version-text">{{ t('app.version') }}</span>
        </a>
      </div>
    </div>

    <!-- 主区域 -->
    <div :class="['panel-main', { expanded: expanded || pinned }]">
      <header class="app-header">
        <h1 class="page-title">{{ t($route.meta.title || '') }}</h1>
        <div class="header-actions">
          <ToggleLocale />
          <a
            href="https://manager.kejizero.xyz"
            target="_blank"
            rel="noopener"
            class="header-btn"
            :title="t('app.docs')"
          >
            <Icon name="book" size="18" />
          </a>
          <a
            href="https://github.com/MinimaxFlora/Docker_Manager_Go"
            target="_blank"
            rel="noopener"
            class="header-btn"
            :title="t('app.github')"
          >
            <Icon name="github" size="18" filled />
          </a>
        </div>
      </header>

      <!-- 默认密码警告横幅 -->
      <div
        v-if="user.mustChangePassword"
        class="pwd-banner"
      >
        <Icon name="alert" size="15" class="shrink-0" />
        <span class="flex-1">{{ t('banner.changePwd') }}</span>
        <router-link to="/settings" class="pwd-banner-link">
          {{ t('banner.goSettings') }}
        </router-link>
      </div>

      <main class="app-main">
        <router-view v-slot="{ Component }">
          <keep-alive include="DashboardView">
            <component :is="Component" />
          </keep-alive>
        </router-view>
      </main>

      <footer class="app-footer">
        <a href="https://github.com/MinimaxFlora" target="_blank" rel="noopener" class="hover:text-brand transition-colors">
          Copyright © {{ year }} MinimaxFlora
        </a>
        <div class="ml-auto flex items-center gap-3">
          <a href="https://github.com/MinimaxFlora/Docker_Manager_Go" target="_blank" rel="noopener" class="hover:text-brand transition-colors">
            {{ t('footer.project') }}
          </a>
          <span class="text-muted">|</span>
          <a href="https://github.com/MinimaxFlora/Docker_Manager_Go#readme" target="_blank" rel="noopener" class="hover:text-brand transition-colors">
            {{ t('footer.manual') }}
          </a>
          <span class="text-muted">|</span>
          <span class="flex items-center gap-1.5">
            <span :class="licenseActive ? 'text-brand font-semibold' : ''">
              {{ licenseActive ? t('license.pro') : t('license.community') }}
            </span>
            <span class="px-1.5 py-0.5 rounded-md bg-surface2 border border-line">{{ t('app.version') }}</span>
          </span>
        </div>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import ToggleLocale from '../components/ToggleLocale.vue'
import SwitchAppearance from '../components/SwitchAppearance.vue'
import { getToken, setToken } from '../api'
import { licenseActive, resetUser, user } from '../store'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const PINNED_KEY = 'dm_sidebar_pinned'

// 3x-ui 交互:默认折叠,悬停展开;图钉固定后保持展开
const hovered = ref(false)
const pinned = ref(localStorage.getItem(PINNED_KEY) === 'true')
const expanded = computed(() => hovered.value || pinned.value)
const year = new Date().getFullYear()

function onEnter() {
  hovered.value = true
}
function onLeave() {
  hovered.value = false
}
function togglePinned() {
  pinned.value = !pinned.value
  localStorage.setItem(PINNED_KEY, pinned.value ? 'true' : 'false')
}

function isActive(item) {
  if (item.to === '/') return route.path === '/'
  return route.path.startsWith(item.to)
}

// 从 JWT 解析用户名(登录后 /me 会刷新完整资料)
const usernameFromToken = ref('')
try {
  const payload = (getToken() || '').split('.')[1]
  if (payload) usernameFromToken.value = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/'))).sub || ''
} catch { /* ignore */ }
if (!user.username && usernameFromToken.value) {
  user.username = usernameFromToken.value
}

const navs = [
  { to: '/', labelKey: 'nav.systemStatus', icon: 'dashboard' },
  { to: '/containers', labelKey: 'nav.containers', icon: 'container' },
  { to: '/images', labelKey: 'nav.images', icon: 'image' },
  { to: '/networks', labelKey: 'nav.networks', icon: 'network' },
  { to: '/volumes', labelKey: 'nav.volumes', icon: 'volume' },
  { to: '/compose', labelKey: 'nav.compose', icon: 'compose' },
  { to: '/settings', labelKey: 'nav.settings', icon: 'settings' },
]

function logout() {
  setToken(null)
  resetUser()
  router.push('/login')
}
</script>

<style scoped>
.app-shell {
  display: flex;
  min-height: 100vh;
  background: var(--dm-bg);
}

/* ---------- 侧边栏(仿 3x-ui:72px ↔ 220px,悬停展开) ---------- */
.app-sider {
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  z-index: 40;
  width: 72px;
  display: flex;
  flex-direction: column;
  background: var(--dm-surface);
  border-right: 1px solid var(--dm-line);
  transition: width 0.25s ease;
  overflow: visible;
}
.app-sider.expanded {
  width: 220px;
}

/* 品牌区 */
.sider-brand {
  display: flex;
  align-items: center;
  height: 56px;
  padding: 0 14px;
  border-bottom: 1px solid var(--dm-line);
  gap: 8px;
  flex-shrink: 0;
  overflow: hidden;
  white-space: nowrap;
}
.brand-block {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  cursor: pointer;
  flex: 1;
}
.brand-logo {
  width: 30px;
  height: 30px;
  object-fit: contain;
  flex-shrink: 0;
}
.brand-text {
  font-size: 14px;
  font-weight: 700;
  color: var(--dm-text);
  letter-spacing: 0.3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.brand-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}
.brand-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--dm-muted);
  cursor: pointer;
  transition: color 0.2s, background 0.2s;
}
.brand-btn:hover,
.brand-btn.active {
  color: var(--dm-brand);
  background: color-mix(in srgb, var(--dm-brand) 10%, transparent);
}

/* 导航菜单(仿 3x-ui antd Menu) */
.sider-nav {
  flex: 1;
  padding: 8px 6px;
  overflow-y: auto;
  overflow-x: hidden;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 40px;
  margin: 4px 0;
  padding: 0 12px;
  border-radius: 8px;
  font-size: 13.5px;
  font-weight: 500;
  color: var(--dm-muted);
  text-decoration: none;
  white-space: nowrap;
  transition: background 0.15s, color 0.15s;
}
.nav-item:hover {
  color: var(--dm-text);
  background: var(--dm-surface2);
}
.nav-item.active {
  color: var(--dm-brand);
  background: color-mix(in srgb, var(--dm-brand) 12%, transparent);
}
.nav-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  width: 3px;
  height: 20px;
  border-radius: 0 3px 3px 0;
  background: var(--dm-brand);
}
.nav-item {
  position: relative;
}
.nav-item.is-collapsed {
  justify-content: center;
  padding: 0;
}
.nav-item.is-collapsed .nav-label {
  display: none;
}
.nav-icon {
  flex-shrink: 0;
}

/* 底部登出(仿 3x-ui sider-utility) */
.sider-footer {
  border-top: 1px solid var(--dm-line);
  padding: 8px 6px;
  flex-shrink: 0;
}
.logout-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 40px;
  margin: 4px 0;
  padding: 0 12px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: #f87171;
  font-size: 13.5px;
  font-weight: 500;
  cursor: pointer;
  text-align: left;
  white-space: nowrap;
  transition: background 0.15s;
}
.logout-item:hover {
  background: rgba(248, 113, 113, 0.12);
}
.logout-item.is-collapsed {
  justify-content: center;
  padding: 0;
}
.logout-item.is-collapsed .nav-label {
  display: none;
}

/* 版本徽章(仿 3x-ui sider-version) */
.sider-version {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-top: 2px;
  padding: 7px 0;
  border-radius: 8px;
  color: var(--dm-muted);
  font-size: 11.5px;
  text-decoration: none;
  transition: color 0.15s, background 0.15s;
}
.sider-version:hover {
  color: var(--dm-brand);
  background: var(--dm-surface2);
}
.sider-version.is-collapsed {
  padding: 8px 0;
}

/* ---------- 主区域 ---------- */
.panel-main {
  flex: 1;
  min-width: 0;
  margin-left: 72px;
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  transition: margin-left 0.25s ease;
}
.panel-main.expanded {
  margin-left: 220px;
}

.app-header {
  display: flex;
  align-items: center;
  height: 56px;
  padding: 0 20px;
  gap: 12px;
  background: var(--dm-bg);
  border-bottom: 1px solid var(--dm-line);
  flex-shrink: 0;
}
.page-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--dm-text);
}
.header-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 4px;
}
.header-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 8px;
  color: var(--dm-muted);
  transition: color 0.2s, background 0.2s;
}
.header-btn:hover {
  color: var(--dm-text);
  background: var(--dm-surface2);
}

.pwd-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 9px 20px;
  font-size: 13px;
  background: rgba(245, 158, 11, 0.14);
  border-bottom: 1px solid rgba(245, 158, 11, 0.3);
  color: #fbbf24;
  flex-shrink: 0;
}
.pwd-banner-link {
  padding: 3px 10px;
  border-radius: 8px;
  background: rgba(245, 158, 11, 0.2);
  font-weight: 500;
  text-decoration: none;
  color: #fbbf24;
  flex-shrink: 0;
}

.app-main {
  flex: 1;
  overflow-y: auto;
  padding: 18px 20px;
}

.app-footer {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 20px;
  font-size: 13px;
  color: var(--dm-footer-color);
  background: var(--dm-surface);
  border-top: 1px solid var(--dm-line);
  flex-shrink: 0;
  flex-wrap: wrap;
}
</style>
