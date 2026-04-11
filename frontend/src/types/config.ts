import type { ManifestSource } from './manifest-source'

// 应用配置
export interface AppConfig {
  manifestUrl: string
  manifestSources: ManifestSource[]
  currentManifestSourceId: string
  versionsDir: string
  dataDir: string
  downloadsDir: string
  maxConcurrent: number
  currentVersion: string
  theme: string
  language: string
  backgroundImage: string
}
