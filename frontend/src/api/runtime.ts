import * as AppBindings from '../../wailsjs/go/main/App'

// .NET 运行时检查结果
export interface RuntimeStatus {
  enabled: boolean // 是否开启了自动检查
  required: boolean // 游戏是否依赖系统运行时
  installed: boolean // 本机是否已安装所需大版本
  majorVersion: number // 所需大版本（如 10）
  source?: string
  tfm?: string
}

// 检查指定版本游戏所需的 .NET 运行时
export function CheckDotNetRuntime(versionId: string): Promise<RuntimeStatus> {
  return AppBindings.CheckDotNetRuntime(versionId) as Promise<RuntimeStatus>
}

// 安装指定版本游戏所需的 .NET 运行时
export function InstallDotNetRuntime(versionId: string): Promise<void> {
  return AppBindings.InstallDotNetRuntime(versionId)
}

// 设置是否在启动游戏前自动检查 .NET 运行时
export function SetAutoCheckRuntime(enabled: boolean): Promise<void> {
  return AppBindings.SetAutoCheckRuntime(enabled)
}
