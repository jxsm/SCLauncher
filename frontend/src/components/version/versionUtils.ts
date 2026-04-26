import type { VersionType } from '../../types/version'

/**
 * 获取版本类型的本地化文本
 */
export function getTypeText(t: any, type: string): string {
  const types: Record<string, string> = {
    api: t('versions.apiVersion'),
    net: t('versions.netVersion'),
    original: t('versions.originalVersion'),
    modified: t('versions.modifiedVersion')
  }
  return types[type] || type
}

/**
 * 获取版本类型对应的标签颜色
 */
export function getTypeColor(type: string): 'info' | 'success' | 'warning' | 'error' | 'default' {
  switch (type) {
    case 'api': return 'info'
    case 'net': return 'warning'
    case 'original': return 'success'
    case 'modified': return 'error'
    default: return 'default'
  }
}
