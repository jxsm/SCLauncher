import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN'
import enUS from './en-US'
import ruRU from './ru-RU'
import ptBR from './pt-BR'
import hiIN from './hi-IN'
import idID from './id-ID'
import arSA from './ar-SA'

const messages = {
  'zh-CN': zhCN,
  'en-US': enUS,
  'ru-RU': ruRU,
  'pt-BR': ptBR,
  'hi-IN': hiIN,
  'id-ID': idID,
  'ar-SA': arSA
}

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages
})

export default i18n
