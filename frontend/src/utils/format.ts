// 格式化文件大小
export function formatSize(bytes: number): string {
  const KB = 1024
  const MB = 1024 * KB
  const GB = 1024 * MB

  if (bytes >= GB) {
    return (bytes / GB).toFixed(2) + ' GB'
  } else if (bytes >= MB) {
    return (bytes / MB).toFixed(2) + ' MB'
  } else if (bytes >= KB) {
    return (bytes / KB).toFixed(2) + ' KB'
  } else {
    return bytes + ' B'
  }
}

// 格式化下载速度
export function formatSpeed(bytesPerSecond: number): string {
  return formatSize(bytesPerSecond) + '/s'
}

// 格式化时间
export function formatTime(seconds: number): string {
  if (seconds < 60) {
    return Math.round(seconds) + '秒'
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60)
    const secs = Math.round(seconds % 60)
    return `${minutes}分${secs}秒`
  } else {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    return `${hours}小时${minutes}分钟`
  }
}

// 计算剩余时间
export function calculateRemainingTime(downloaded: number, total: number, speed: number): number {
  if (speed === 0) return 0
  const remaining = total - downloaded
  return remaining / speed
}
