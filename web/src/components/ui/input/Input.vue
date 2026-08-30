<script setup>
import { computed } from "vue";
import { cn } from "@/lib/utils";

const props = defineProps({
  defaultValue: { type: [String, Number], required: false },
  modelValue: { type: [String, Number], required: false },
  modelModifiers: { type: Object, required: false, default: () => ({}) },
  class: {
    type: [Boolean, null, String, Object, Array],
    required: false,
    skipCheck: true,
  },
});

const emits = defineEmits(["update:modelValue"]);

// 支持 v-model.number / v-model.trim(与原生行为一致:parseFloat 失败保留原值)
const modelValue = computed({
  get: () => props.modelValue ?? props.defaultValue ?? "",
  set: (v) => {
    if (props.modelModifiers?.number) {
      const n = parseFloat(v)
      v = Number.isNaN(n) ? v : n
    }
    if (props.modelModifiers?.trim) v = String(v).trim()
    emits("update:modelValue", v)
  },
});
</script>

<template>
  <input
    v-model="modelValue"
    data-slot="input"
    :class="
      cn(
        'file:text-foreground placeholder:text-muted-foreground selection:bg-primary selection:text-primary-foreground border-input bg-input h-9 w-full min-w-0 rounded-[0.6rem] border px-3 py-2 text-[0.875rem] text-text shadow-none transition-[color,box-shadow] outline-none file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50',
        'focus-visible:border-brand focus-visible:ring-brand/30 focus-visible:ring-2',
        'aria-invalid:ring-destructive/20 aria-invalid:border-destructive',
        props.class,
      )
    "
  />
</template>
