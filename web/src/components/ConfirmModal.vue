<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0"
      leave-active-class="transition duration-150 ease-in"
      leave-to-class="opacity-0"
    >
      <div
        v-if="confirmState.visible"
        class="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
        @click.self="resolveConfirm(false)"
      >
        <div class="card w-full fade-up shadow-2xl shadow-black/40 overflow-hidden" :class="sizeClass">
          <div class="px-5 py-4">
            <div class="flex items-center gap-3 mb-2">
              <span
                class="w-9 h-9 rounded-xl flex items-center justify-center shrink-0"
                :class="confirmState.danger ? 'bg-danger/15 text-danger' : 'bg-brand/15 text-brand'"
              >
                <Icon :name="confirmState.danger ? 'alert' : 'info'" :size="18" />
              </span>
              <h3 class="text-sm font-semibold">{{ confirmState.title }}</h3>
            </div>
            <p class="text-sm text-muted leading-relaxed whitespace-pre-wrap">{{ confirmState.message }}</p>
          </div>
          <div class="px-5 py-3.5 bg-surface/60 flex justify-end gap-2">
            <button class="btn btn-ghost btn-sm" @click="resolveConfirm(false)">{{ t('common.cancel') }}</button>
            <button
              class="btn btn-sm"
              :class="confirmState.danger ? 'btn-danger' : 'btn-brand'"
              @click="resolveConfirm(true)"
            >
              {{ confirmState.confirmText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import Icon from './Icon.vue'
import { confirmState, resolveConfirm } from '../confirm'
import { useI18n } from 'vue-i18n'

// 支持 ESC 取消
import { computed, onMounted, onUnmounted } from 'vue'
const { t } = useI18n()
const sizeClass = computed(() => ({ sm: 'max-w-sm', lg: 'max-w-lg', xl: 'max-w-xl' }[confirmState.size] || 'max-w-sm'))
function onKey(e) {
  if (e.key === 'Escape' && confirmState.visible) resolveConfirm(false)
}
onMounted(() => window.addEventListener('keydown', onKey))
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>
