<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0"
      leave-active-class="transition duration-150 ease-in"
      leave-to-class="opacity-0"
    >
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
        @click.self="emit('close')"
      >
        <div class="card w-full max-h-[85vh] flex flex-col fade-up shadow-2xl shadow-black/40" :class="sizeClass">
          <div class="flex items-center justify-between px-5 py-4 border-b border-line">
            <h3 class="text-sm font-semibold">{{ title }}</h3>
            <button class="btn btn-icon btn-sm" @click="emit('close')">
              <Icon name="x" size="14" />
            </button>
          </div>
          <div class="px-5 py-4 overflow-y-auto">
            <slot />
          </div>
          <div v-if="$slots.footer" class="px-5 py-3.5 border-t border-line flex justify-end gap-2 bg-surface/60 rounded-b-[0.9rem]">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import Icon from './Icon.vue'

const props = defineProps({
  modelValue: Boolean,
  title: { type: String, default: '' },
  size: { type: String, default: 'lg' }, // lg | xl | 2xl
})
const emit = defineEmits(['close', 'update:modelValue'])

const sizeClass = computed(() => ({ lg: 'max-w-lg', xl: 'max-w-xl', '2xl': 'max-w-2xl' }[props.size] || 'max-w-lg'))
</script>
