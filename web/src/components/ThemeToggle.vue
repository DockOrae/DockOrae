<template>
  <Button
    variant="ghost"
    size="icon"
    :aria-label="t('theme.toggle')"
    :title="t('theme.toggle')"
    @click="toggleThemeWithTransition($event)"
  >
    <Icon :name="theme === 'dark' ? 'sun' : 'moon'" size="16" filled />
  </Button>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import { Button } from '@/components/ui/button'
import { theme, toggleThemeWithTransition } from '../store'

const { t } = useI18n()
</script>
