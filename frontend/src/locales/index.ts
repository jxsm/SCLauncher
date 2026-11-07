import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN'
import enUS from './en-US'
import ruRU from './ru-RU'
import ptBR from './pt-BR'
import hiIN from './hi-IN'
import idID from './id-ID'
import arSA from './ar-SA'
import esES from './es-ES'

const messages = {
  'zh-CN': zhCN,
  'en-US': enUS,
  'ru-RU': ruRU,
  'pt-BR': ptBR,
  'hi-IN': hiIN,
  'id-ID': idID,
  'ar-SA': arSA,
  'es-ES': esES
}

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  // 缺键回退：其他语言缺键时回退到英文（而不是中文）
  fallbackLocale: {
    'zh-CN': ['en-US'],
    default: ['en-US']
  },
  messages
})

export default i18n
