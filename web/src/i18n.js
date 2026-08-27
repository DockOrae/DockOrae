import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import en from './locales/en'

const LANG_KEY = 'dm_lang'

export const LANGS = [
  { code: 'zh-CN', labelKey: 'lang.zhCN' },
  { code: 'en', labelKey: 'lang.en' },
]

const SUPPORTED = LANGS.map((l) => l.code)
const DEFAULT_LANG = 'zh-CN'

/** 浏览器语言自动检测:精确匹配 → 语言前缀匹配 → 默认(与 OpenList-Docs 一致) */
export function detectLang() {
  try {
    const saved = localStorage.getItem(LANG_KEY)
    if (saved && SUPPORTED.includes(saved)) return saved
    const browserLangs = navigator.languages || [navigator.language || DEFAULT_LANG]
    for (const bl of browserLangs) {
      if (SUPPORTED.includes(bl)) return bl
    }
    for (const bl of browserLangs) {
      const prefix = bl.split('-')[0]
      const matched = SUPPORTED.find((l) => l.startsWith(prefix))
      if (matched) return matched
    }
    return DEFAULT_LANG
  } catch {
    return DEFAULT_LANG
  }
}

const i18n = createI18n({
  legacy: false,
  locale: detectLang(),
  fallbackLocale: 'zh-CN',
  messages: { 'zh-CN': zhCN, en },
})

export function setLang(code) {
  if (!SUPPORTED.includes(code)) return
  i18n.global.locale.value = code
  localStorage.setItem(LANG_KEY, code)
  document.documentElement.lang = code
}

export function t(key, params) {
  return i18n.global.t(key, params)
}

export default i18n
