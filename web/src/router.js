import { createRouter, createWebHistory } from 'vue-router'
import { getToken } from './api'

const routes = [
  { path: '/login', name: 'login', component: () => import('./views/LoginView.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('./layouts/MainLayout.vue'),
    children: [
      { path: '', name: 'dashboard', component: () => import('./views/DashboardView.vue'), meta: { title: 'nav.dashboard' } },
      { path: 'containers', name: 'containers', component: () => import('./views/ContainersView.vue'), meta: { title: 'nav.containers' } },
      { path: 'containers/new', name: 'container-create', component: () => import('./views/ContainerCreateView.vue'), meta: { title: 'createContainer.title' } },
      { path: 'containers/:id', name: 'container-detail', component: () => import('./views/ContainerDetailView.vue'), meta: { title: 'containerDetail.title' } },
      { path: 'images', name: 'images', component: () => import('./views/ImagesView.vue'), meta: { title: 'nav.images' } },
      { path: 'networks', name: 'networks', component: () => import('./views/NetworksView.vue'), meta: { title: 'nav.networks' } },
      { path: 'volumes', name: 'volumes', component: () => import('./views/VolumesView.vue'), meta: { title: 'nav.volumes' } },
      { path: 'compose', name: 'compose', component: () => import('./views/ComposeView.vue'), meta: { title: 'nav.compose' } },
      { path: 'terminal', name: 'terminal', component: () => import('./views/TerminalView.vue'), meta: { title: 'nav.terminal' } },
      { path: 'compose/:project', name: 'compose-detail', component: () => import('./views/ComposeDetailView.vue'), meta: { title: 'composeDetail.title' } },
      { path: 'settings', name: 'settings', component: () => import('./views/SettingsView.vue'), meta: { title: 'nav.settings' } },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  if (!to.meta.public && !getToken()) return { name: 'login' }
  if (to.name === 'login' && getToken()) return { name: 'dashboard' }
})

export default router
