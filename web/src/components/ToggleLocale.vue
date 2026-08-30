<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        variant="ghost"
        size="icon"
        :title="t('lang.toggle')"
        :aria-label="t('lang.toggle')"
      >
        <Icon name="translate" size="16" filled />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="w-52 max-h-80 overflow-y-auto">
      <DropdownMenuItem
        v-for="l in LANGS"
        :key="l.code"
        :class="l.code === current.code ? 'text-brand' : ''"
        @select="onPick(l.code)"
      >
        <span class="mr-2" aria-hidden="true">{{ l.flag }}</span>
        <span>{{ l.label }}</span>
        <Icon v-if="locale === l.code" name="check" size="14" class="ml-auto text-brand" />
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { LANGS, setLang } from '../i18n'

const { t, locale } = useI18n()

const current = computed(() => LANGS.find((l) => l.code === locale.value) || LANGS[0])

function onPick(code) {
  setLang(code)
}
</script>
