import { createRouter, createWebHistory } from 'vue-router'
import { getToken, getPublicConfig } from './api'

const routes = [
  { path: '/login', name: 'login', component: () => import('./views/LoginView.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('./layouts/MainLayout.vue'),
    children: [
      { path: '', name: 'dashboard', component: () => import('./views/DashboardView.vue'), meta: { title: 'nav.systemStatus' } },
      { path: 'apps', name: 'appstore', component: () => import('./views/AppStoreView.vue'), meta: { title: 'nav.appStore' } },
      { path: 'containers', name: 'containers', component: () => import('./views/ContainersView.vue'), meta: { title: 'nav.containers' } },
      { path: 'containers/new', name: 'container-create', component: () => import('./views/ContainerCreateView.vue'), meta: { title: 'createContainer.title' } },
      { path: 'containers/:id', name: 'container-detail', component: () => import('./views/ContainerDetailView.vue'), meta: { title: 'containerDetail.title' } },
      { path: 'images', name: 'images', component: () => import('./views/ImagesView.vue'), meta: { title: 'nav.images' } },
      { path: 'networks', name: 'networks', component: () => import('./views/NetworksView.vue'), meta: { title: 'nav.networks' } },
      { path: 'volumes', name: 'volumes', component: () => import('./views/VolumesView.vue'), meta: { title: 'nav.volumes' } },
      { path: 'compose', name: 'compose', component: () => import('./views/ComposeView.vue'), meta: { title: 'nav.compose' } },
      { path: 'compose/:project', name: 'compose-detail', component: () => import('./views/ComposeDetailView.vue'), meta: { title: 'composeDetail.title' } },
      { path: 'settings', name: 'settings', component: () => import('./views/SettingsView.vue'), meta: { title: 'nav.settings' } },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

// 安全入口(webBasePath)作为 router base:设置后页面 URL 带 /<入口> 前缀,
// 后端会把不带入口的请求 302 重定向到入口路径。
export async function createAppRouter() {
  const cfg = await getPublicConfig()
  const base = cfg.basePath && cfg.basePath !== '/' ? cfg.basePath.replace(/\/$/, '') : ''
  const router = createRouter({
    history: createWebHistory(base),
    routes,
  })

  router.beforeEach((to) => {
    if (!to.meta.public && !getToken()) return { name: 'login' }
    if (to.name === 'login' && getToken()) return { name: 'dashboard' }
  })

  return router
}

export default null
