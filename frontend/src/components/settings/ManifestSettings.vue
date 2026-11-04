<template>
  <n-card :title="t('settings.manifest')">
    <template #header-extra>
      <n-button size="small" type="primary" @click="$emit('add-source')">
        <template #icon>
          <n-icon><AddIcon /></n-icon>
        </template>
        {{ t('settings.addManifestSource') }}
      </n-button>
    </template>

    <n-space vertical size="medium">
      <!-- 清单源下拉选择 -->
      <n-select
        :value="currentSourceId"
        :options="sourceOptions"
        :consistent-menu-width="false"
        @update:value="handleSelect"
      />

      <!-- 默认清单源：双线路自动测速信息 -->
      <div v-if="isDefaultSelected && speedInfo" class="speed-info">
        <div
          v-for="line in lines"
          :key="line.url"
          class="speed-line"
          :class="{ selected: line.isSelected }"
        >
          <span class="line-name">{{ line.name }}</span>
          <span class="line-host">{{ line.host }}</span>
          <span class="line-latency" :class="line.status">{{ line.latencyText }}</span>
          <n-tag v-if="line.isSelected" type="success" size="small" round>
            {{ t('settings.manifestSpeedInUse') }}
          </n-tag>
        </div>
        <div class="speed-footer">
          <span class="last-test">{{ lastTestText }}</span>
          <n-button
            size="tiny"
            :loading="testing"
            :disabled="testing"
            @click="$emit('retest')"
          >
            {{ t('settings.manifestSpeedRetest') }}
          </n-button>
        </div>
      </div>

      <!-- 自定义清单源：显示地址与删除按钮 -->
      <n-space v-else-if="currentCustomSource" align="center" justify="space-between">
        <n-text depth="3" style="font-size: 12px; word-break: break-all;">
          {{ currentCustomSource.url }}
        </n-text>
        <n-button size="tiny" type="error" @click="$emit('delete', currentCustomSource)">
          {{ t('common.delete') }}
        </n-button>
      </n-space>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Add as AddIcon } from '@vicons/ionicons5'
import type { ManifestSource } from '../../types/manifest-source'
import type { DefaultManifestInfo } from '../../types/config'

const { t } = useI18n()

const props = defineProps<{
  sources: ManifestSource[]
  currentSourceId: string
  speedInfo?: DefaultManifestInfo | null
  testing?: boolean
}>()

const emit = defineEmits<{
  'add-source': []
  'set-current': [source: ManifestSource]
  delete: [source: ManifestSource]
  retest: []
}>()

// 下拉选项：默认源固定显示本地化名称（不使用存储的名称）
const sourceOptions = computed(() =>
  props.sources.map(s => ({
    label: s.id === 'default' ? t('settings.defaultManifestSourceName') : s.name,
    value: s.id
  }))
)

const isDefaultSelected = computed(() => props.currentSourceId === 'default')

const currentCustomSource = computed(() =>
  props.sources.find(s => s.id === props.currentSourceId && s.id !== 'default')
)

function handleSelect(sourceId: string) {
  const source = props.sources.find(s => s.id === sourceId)
  if (source) {
    emit('set-current', source)
  }
}

// 两条线路的展示信息（名称 / 域名 / 延迟 / 是否当前使用）
const lines = computed(() => {
  const info = props.speedInfo
  if (!info) return []
  const hasTested = (info.lastTest ?? 0) > 0

  const build = (name: string, url: string, latencyMs: number) => {
    let latencyText: string
    let status: 'ok' | 'failed' | 'untested'
    if (!hasTested || latencyMs === 0) {
      latencyText = t('settings.manifestSpeedNeverTested')
      status = 'untested'
    } else if (latencyMs < 0) {
      latencyText = t('settings.manifestSpeedFailed')
      status = 'failed'
    } else {
      latencyText = `${latencyMs} ms`
      status = 'ok'
    }
    let host = url
    try {
      host = new URL(url).host
    } catch {
      // URL 解析失败时原样显示
    }
    return {
      name,
      host,
      url,
      latencyText,
      status,
      isSelected: url === info.selectedUrl
    }
  }

  return [
    build(t('settings.manifestSpeedLineMain'), info.mainUrl, info.mainLatencyMs ?? 0),
    build(t('settings.manifestSpeedLineMirror'), info.mirrorUrl, info.mirrorLatencyMs ?? 0)
  ]
})

const lastTestText = computed(() => {
  const info = props.speedInfo
  if (!info || !info.lastTest) {
    return t('settings.manifestSpeedNeverTested')
  }
  return t('settings.manifestSpeedLastTest', { time: new Date(info.lastTest * 1000).toLocaleString() })
})
</script>

<style scoped>
.speed-info {
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--n-color-embedded, rgba(128, 128, 128, 0.08));
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.speed-line {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  padding: 2px 0;
}

.speed-line.selected .line-name {
  font-weight: 600;
}

.line-name {
  min-width: 36px;
}

.line-host {
  opacity: 0.75;
  flex: 1;
}

.line-latency {
  font-variant-numeric: tabular-nums;
}

.line-latency.ok {
  color: var(--n-success-color, #18a058);
  font-weight: 600;
}

.line-latency.failed {
  color: var(--n-error-color, #d03050);
}

.line-latency.untested {
  opacity: 0.55;
}

.speed-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 2px;
}

.last-test {
  font-size: 12px;
  opacity: 0.6;
}
</style>
