<template>
  <div class="h-screen overflow-hidden">
    <!-- 毛玻璃背景(照片虚化) -->
    <div class="bg-layer" />

    <!-- 收起/展开悬浮按钮(仿 1Panel:骑在侧边栏右边缘的圆形按钮) -->
    <button
      class="side-toggle"
      :class="sidebarOpen ? 'is-open' : 'is-collapsed'"
      :title="t(sidebarOpen ? 'nav.collapse' : 'nav.expand')"
      @click="setSidebar(!sidebarOpen)"
    >
      <Icon :name="sidebarOpen ? 'chevronsLeft' : 'chevronsRight'" size="14" />
    </button>

    <!-- 侧边栏(仿 1Panel:180px ↔ 75px,0.3s 宽度过渡) -->
    <aside :class="['sidebar fixed left-0 top-0 bottom-0 bg-surface/95 backdrop-blur border-r border-line flex flex-col z-40', sidebarOpen ? 'is-open' : 'is-collapsed']">
      <!-- Logo(49px 高,仿 1Panel) -->
      <div class="flex items-center justify-center h-[49px] border-b border-line shrink-0">
        <img
          src="/logo.svg"
          alt="logo"
          :class="['object-contain transition-all', sidebarOpen ? 'w-9 h-9' : 'w-7 h-7']"
        />
      </div>

      <nav class="flex-1 px-[7px] py-2 overflow-y-auto overflow-x-hidden">
        <router-link
          v-for="item in navs"
          :key="item.to"
          :to="item.to"
          :title="sidebarOpen ? '' : t(item.labelKey)"
          class="side-link"
          :class="[
            $route.path === item.to || ($route.path.startsWith(item.to + '/') && item.to !== '/') ? 'active' : '',
            sidebarOpen ? '' : 'is-collapsed',
          ]"
        >
          <Icon :name="item.icon" size="16" class="shrink-0" />
          <span v-if="sidebarOpen" class="truncate">{{ t(item.labelKey) }}</span>
        </router-link>
      </nav>

      <!-- 用户区(仿 1Panel 主节点:主机图标 + 名称,点击右侧滑出菜单) -->
      <div class="user-menu-wrap p-2 border-t border-line shrink-0 relative">
        <button
          class="w-full flex items-center justify-center gap-2.5 px-2 py-2 rounded-lg hover:bg-surface2 transition-colors cursor-pointer"
          :class="sidebarOpen ? '' : '!justify-center'"
          @click="userMenuOpen = !userMenuOpen"
        >
          <Icon name="server" size="18" class="text-brand shrink-0" />
          <span v-if="sidebarOpen" class="flex-1 min-w-0 text-left">
            <span class="block text-[13px] font-medium truncate">{{ t('app.mainNode') }}</span>
            <span class="block text-[10px] text-muted truncate">{{ mainNodeName }}</span>
          </span>
          <Icon v-if="sidebarOpen" name="chevronsUp" size="14" class="text-muted shrink-0 transition-transform duration-200" :class="{ 'rotate-180': userMenuOpen }" />
        </button>

        <!-- 右侧滑出菜单(仿 1Panel Drawer:用户设置 / 许可证 / 退出) -->
        <Transition name="dm-slide">
          <div v-if="userMenuOpen" class="absolute left-full bottom-0 ml-2 z-50 w-48 rounded-xl border border-line bg-surface shadow-xl overflow-hidden">
            <router-link
              to="/settings"
              class="flex items-center gap-2.5 px-3.5 py-2.5 text-[13px] text-muted hover:text-text hover:bg-surface2 transition-colors"
              @click="userMenuOpen = false"
            >
              <Icon name="settings" size="14" /> {{ t('app.userSettings') }}
            </router-link>
            <router-link
              to="/settings?tab=license"
              class="flex items-center gap-2.5 px-3.5 py-2.5 text-[13px] text-muted hover:text-text hover:bg-surface2 transition-colors"
              @click="userMenuOpen = false"
            >
              <Icon name="key" size="14" />
              {{ t('license.title') }}
              <span class="ml-auto badge text-[10px]" :style="licenseActive ? okStyle : mutedStyle">
                {{ licenseActive ? t('license.pro') : t('license.community') }}
              </span>
            </router-link>
            <div class="border-t border-line" />
            <button
              class="w-full flex items-center gap-2.5 px-3.5 py-2.5 text-[13px] text-danger hover:bg-danger/10 transition-colors"
              @click="logout"
            >
              <Icon name="logout" size="14" /> {{ t('nav.logout') }}
            </button>
          </div>
        </Transition>
      </div>
    </aside>

    <!-- 主区域(仿 1Panel:100vh 内部滚动,margin-left 跟随) -->
    <div :class="['panel-main', sidebarOpen ? 'is-open' : 'is-collapsed']">
      <header class="shrink-0 h-14 px-6 flex items-center gap-3 bg-bg/70 backdrop-blur-xl border-b border-line">
        <h1 class="text-[15px] font-semibold">{{ t($route.meta.title || '') }}</h1>
        <div class="ml-auto flex items-center gap-3 text-[11px] text-muted">
          <SwitchAppearance />
          <ToggleLocale />
          <a
            href="https://manager.kejizero.xyz"
            target="_blank"
            rel="noopener"
            class="dm-social-link flex items-center justify-center w-9 h-9 rounded-lg hover:bg-surface2 transition-colors"
            :title="t('app.docs')"
          >
            <Icon name="book" size="20" />
          </a>
          <a
            href="https://github.com/MinimaxFlora/Docker_Manager_Go"
            target="_blank"
            rel="noopener"
            class="dm-social-link flex items-center justify-center w-9 h-9 rounded-lg hover:bg-surface2 transition-colors"
            :title="t('app.github')"
          >
            <Icon name="github" size="20" filled />
          </a>
        </div>
      </header>

      <!-- 默认密码警告横幅 -->
      <div
        v-if="user.mustChangePassword"
        class="shrink-0 flex items-center gap-3 px-6 py-2.5 text-[13px] bg-amber-500/15 border-b border-amber-500/30 text-amber-300"
      >
        <Icon name="alert" size="15" class="shrink-0" />
        <span class="flex-1">{{ t('banner.changePwd') }}</span>
        <router-link to="/settings" class="shrink-0 px-3 py-1 rounded-lg bg-amber-500/20 hover:bg-amber-500/30 font-medium transition-colors">
          {{ t('banner.goSettings') }}
        </router-link>
      </div>

      <main class="flex-1 overflow-y-auto p-4 md:p-5">
        <router-view v-slot="{ Component }">
          <keep-alive include="DashboardView">
            <component :is="Component" />
          </keep-alive>
        </router-view>
      </main>

      <!-- 页脚(仿 1Panel AppFooter:Copyright 左、链接右、跟随主题) -->
      <footer
        class="shrink-0 px-5 py-3 flex items-center gap-4 border-t border-line flex-wrap text-[13px]"
        style="color: var(--dm-footer-color); background: var(--dm-surface);"
      >
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
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import ToggleLocale from '../components/ToggleLocale.vue'
import SwitchAppearance from '../components/SwitchAppearance.vue'
import { getToken, setToken } from '../api'
import { licenseActive, loadLicense, resetUser, user } from '../store'

const { t } = useI18n()
const router = useRouter()
const sidebarOpen = ref(localStorage.getItem('dm_sidebar') !== '0')
const userMenuOpen = ref(false)
const year = new Date().getFullYear()
const mainNodeName = ref('')
const okStyle = { color: '#34d399', background: 'rgba(52,211,153,.12)', border: '1px solid rgba(52,211,153,.3)' }
const mutedStyle = { color: '#8b93a7', background: 'rgba(139,147,167,.12)', border: '1px solid rgba(139,147,167,.3)' }

function setSidebar(v) {
  sidebarOpen.value = v
  localStorage.setItem('dm_sidebar', v ? '1' : '0')
}

onMounted(() => {
  loadLicense().then((r) => {
    // 主节点名 = 设备 ID 的主机部分(如 ZHAO@docker-manager → ZHAO)
    if (r && r.device_id) mainNodeName.value = String(r.device_id).split('@')[0]
  })
  document.addEventListener('click', onDocClick)
})
onBeforeUnmount(() => document.removeEventListener('click', onDocClick))

function onDocClick(e) {
  if (userMenuOpen.value && !e.target.closest('.user-menu-wrap')) userMenuOpen.value = false
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
  { to: '/', labelKey: 'nav.dashboard', icon: 'dashboard' },
  { to: '/containers', labelKey: 'nav.containers', icon: 'container' },
  { to: '/images', labelKey: 'nav.images', icon: 'image' },
  { to: '/networks', labelKey: 'nav.networks', icon: 'network' },
  { to: '/volumes', labelKey: 'nav.volumes', icon: 'volume' },
  { to: '/compose', labelKey: 'nav.compose', icon: 'compose' },
  { to: '/terminal', labelKey: 'nav.terminal', icon: 'terminal' },
  { to: '/settings', labelKey: 'nav.settings', icon: 'settings' },
]

function logout() {
  setToken(null)
  resetUser()
  router.push('/login')
}
</script>

<style scoped>
/* 亮色主题:照片背景淡化为浅色毛玻璃(若隐若现,整体保持浅色调) */
[data-theme="light"] .bg-layer {
  opacity: 0.32;
  filter: blur(18px) brightness(2.6) saturate(0.5);
}
[data-theme="light"] .bg-layer::after {
  background: radial-gradient(ellipse at center, transparent 25%, rgba(255, 255, 255, 0.65) 100%);
}
.bg-layer {
  position: fixed;
  inset: 0;
  z-index: 0;
  background-color: #0b0e14;
  background-image: url('/bg.jpg');
  background-size: cover;
  background-position: center;
  filter: blur(12px) brightness(0.45) saturate(0.9);
  transform: scale(1.12);
  transition: opacity 0.3s ease;
}
.bg-layer::after {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse at center, transparent 20%, rgba(5, 8, 14, 0.55) 100%);
}

/* ---------- 侧边栏(仿 1Panel:180px ↔ 75px,0.3s) ---------- */
.sidebar {
  width: 180px;
  transition: width 0.3s;
  font-size: 0;
}
.sidebar.is-collapsed {
  width: 75px;
}

/* 收起按钮:骑在侧边栏右边缘的悬浮圆形按钮(仿 1Panel el-affix) */
.side-toggle {
  position: fixed;
  top: 8px;
  z-index: 60;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--dm-surface);
  border: 1px solid var(--dm-line);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.25);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--dm-muted);
  cursor: pointer;
  transition:
    left 0.3s,
    color 0.2s,
    border-color 0.2s;
  left: 168px;
}
.side-toggle:hover {
  color: var(--dm-brand);
  border-color: var(--dm-brand);
}
.side-toggle.is-collapsed {
  left: 63px;
}

/* 菜单项(仿 1Panel:42px 高、圆角 4px、细阴影、active 左侧竖条 + inset 描边) */
.side-link {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 42px;
  margin: 7px 0;
  padding: 0 14px;
  border-radius: 4px;
  font-size: 13px;
  font-weight: 500;
  background: var(--dm-surface2);
  box-shadow: 0 0 4px rgba(0, 0, 0, 0.1);
  color: var(--dm-muted);
  position: relative;
  transition: color 0.2s;
  white-space: nowrap;
}
.side-link:hover {
  color: var(--dm-brand);
}
.side-link.active {
  color: var(--dm-brand);
  box-shadow:
    0 0 4px rgba(0, 0, 0, 0.1),
    inset 0 0 0 2px var(--dm-brand);
}
.side-link.active::before {
  content: '';
  position: absolute;
  left: 12px;
  width: 4px;
  height: 14px;
  border-radius: 4px;
  background: var(--dm-brand);
}
.side-link.is-collapsed {
  justify-content: center;
  padding: 0;
}
.side-link.is-collapsed.active::before {
  display: none;
}

/* 主区域(仿 1Panel:100vh 内部滚动,margin-left 0.3s 过渡) */
.panel-main {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  transition: margin-left 0.3s;
}
.panel-main.is-open {
  margin-left: 180px;
}
.panel-main.is-collapsed {
  margin-left: 75px;
}
</style>
