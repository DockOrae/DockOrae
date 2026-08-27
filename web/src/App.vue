<template>
  <router-view />
  <ConfirmModal />
  <ToastContainer />
</template>

<script setup>
import { onMounted } from 'vue'
import ConfirmModal from './components/ConfirmModal.vue'
import ToastContainer from './components/ToastContainer.vue'
import { api, getToken } from './api'
import { applyUser } from './store'

// 已登录时拉取最新用户资料(昵称/头像/改密标记/2FA 状态)
onMounted(async () => {
  if (!getToken()) return
  try {
    const me = await api('/me')
    applyUser(me)
  } catch { /* 401 由 api 层处理跳转登录 */ }
})
</script>
