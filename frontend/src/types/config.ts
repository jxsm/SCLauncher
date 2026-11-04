import type { ManifestSource } from './manifest-source'

// 默认清单源线路测速结果
export interface DefaultManifestInfo {
  mainUrl: string
  mirrorUrl: string
  selectedUrl: string
  mainLatencyMs: number
  mirrorLatencyMs: number
  lastTest: number
}

// 应用配置
export interface AppConfig {
  manifestUrl: string
  manifestSources: ManifestSource[]
  currentManifestSourceId: string
  defaultManifest: DefaultManifestInfo
  versionsDir: string
  dataDir: string
  downloadsDir: string
  maxConcurrent: number
  currentVersion: string
  theme: string
  language: string
  backgroundImage: string
}

// 单条线路的测速结果（重新测速接口返回）
export interface ManifestSpeedResult {
  line: 'main' | 'mirror'
  url: string
  latencyMs: number
  ok: boolean
  error?: string
}

// 重新测速接口返回
export interface ManifestSpeedTestResponse {
  results: ManifestSpeedResult[]
  selectedUrl: string
  lastTest: number
}
