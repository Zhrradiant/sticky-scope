import { createI18n } from 'vue-i18n'
import zh from './zh'
import en from './en'
import type { Lang } from '@/types'

function initialLocale(): Lang {
  const saved = localStorage.getItem('lang')
  return saved === 'en' || saved === 'zh' ? saved : 'zh'
}

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: initialLocale(),
  fallbackLocale: 'en',
  messages: { zh, en },
})

export function setLocale(l: Lang) {
  i18n.global.locale.value = l
  localStorage.setItem('lang', l)
  document.documentElement.lang = l
}

export function currentLocale(): Lang {
  return i18n.global.locale.value as Lang
}
