<template>
  <n-modal
    :show="showModal"
    @update:show="$emit('update:show', $event)"
    preset="card"
    :title="t('installed.modpackInfo')"
    style="width: 700px; max-width: 90vw;"
  >
    <n-spin :show="loading">
      <div v-if="modpackInfo" class="modpack-info">
        <!-- scbbs（ScbbsPlus）格式 -->
        <template v-if="modpackInfo.format === 'scbbs'">
          <!-- 包内缺失文件警示 -->
          <n-alert v-if="modpackInfo.missingEmbeddedCount > 0" type="warning" style="margin-bottom: 16px;">
            {{ t('installed.scbbsMissingWarning', { count: modpackInfo.missingEmbeddedCount }) }}
          </n-alert>

          <!-- 基本信息 -->
          <div class="scbbs-header">
            <img v-if="iconSrc" :src="iconSrc" class="scbbs-icon" alt="icon" />
            <n-descriptions :column="2" bordered size="small" style="flex: 1;">
              <n-descriptions-item :label="t('installed.modpackName')" :span="2">
                <n-text strong style="font-size: 16px;">{{ modpackInfo.name }}</n-text>
              </n-descriptions-item>

              <n-descriptions-item :label="t('installed.modpackVersion')">
                <n-tag type="primary" size="small">{{ modpackInfo.version }}</n-tag>
              </n-descriptions-item>

              <n-descriptions-item :label="t('installed.scbbsCarrier')">
                <n-tag size="small" :bordered="false">
                  {{ modpackInfo.carrier === 'zip' ? t('installed.scbbsCarrierZip') : t('installed.scbbsCarrierJson') }}
                </n-tag>
              </n-descriptions-item>

              <n-descriptions-item :label="t('installed.scbbsPackageName')" :span="2">
                <n-text code>{{ modpackInfo.packageName }}</n-text>
              </n-descriptions-item>

              <n-descriptions-item v-if="modpackInfo.description" :label="t('installed.modpackDescription')" :span="2">
                {{ modpackInfo.description }}
              </n-descriptions-item>
            </n-descriptions>
          </div>

          <!-- 按类型统计 -->
          <div class="info-cards">
            <div v-for="stat in itemStats" :key="stat.type" class="info-card">
              <div class="card-content">
                <div class="card-label">{{ stat.label }}</div>
                <div class="card-value">{{ stat.count }} {{ t('textures.unitCount') }}</div>
              </div>
            </div>
          </div>

          <!-- 条目列表（可折叠） -->
          <n-collapse class="mod-details">
            <n-collapse-item :title="`${t('installed.scbbsItemsTitle')} (${modpackInfo.itemCount} ${t('textures.unitCount')})`">
              <n-list bordered size="small">
                <n-list-item v-for="(item, index) in modpackInfo.items" :key="index">
                  <div class="mod-item" :style="!item.isEnabled ? 'opacity: 0.5;' : ''">
                    <div class="mod-info">
                      <div class="mod-name">{{ item.displayName }}</div>
                      <div class="mod-version">{{ item.targetDirectory }}</div>
                    </div>
                    <n-space align="center">
                      <n-tag size="small" :type="itemTypeTagType(item.type)">{{ itemTypeLabel(item.type) }}</n-tag>
                      <n-tag size="small" type="default" :bordered="false">{{ itemSourceLabel(item.source) }}</n-tag>
                    </n-space>
                  </div>
                </n-list-item>
              </n-list>
            </n-collapse-item>
          </n-collapse>
        </template>

        <!-- v0.1 格式（原有展示） -->
        <template v-else>
        <!-- 外部链接警告 -->
        <n-alert v-if="modpackInfo.hasExternalLinks" type="warning" style="margin-bottom: 16px;">
          {{ t('installed.modpackExternalLinksWarning') }}
        </n-alert>

        <!-- 基本信息 -->
        <n-descriptions :column="2" bordered size="small">
          <n-descriptions-item :label="t('installed.modpackName')" :span="2">
            <n-text strong style="font-size: 16px;">{{ modpackInfo.name }}</n-text>
          </n-descriptions-item>

          <n-descriptions-item :label="t('installed.modpackVersion')">
            <n-tag type="primary" size="small">{{ modpackInfo.version }}</n-tag>
          </n-descriptions-item>

          <n-descriptions-item :label="t('installed.modpackAuthor')">
            <n-text>{{ modpackInfo.author }}</n-text>
          </n-descriptions-item>

          <n-descriptions-item v-if="modpackInfo.description" :label="t('installed.modpackDescription')" :span="2">
            {{ modpackInfo.description }}
          </n-descriptions-item>

          <n-descriptions-item v-if="modpackInfo.created" :label="t('installed.modpackCreated')">
            {{ modpackInfo.created }}
          </n-descriptions-item>

          <n-descriptions-item v-if="modpackInfo.changelog" :label="t('installed.modpackChangelog')">
            {{ modpackInfo.changelog }}
          </n-descriptions-item>
        </n-descriptions>

        <!-- 游戏信息和模组统计 -->
        <div class="info-cards">
          <!-- Windows 版本 -->
          <div v-if="modpackInfo.survivalcraft?.version?.windows" class="info-card">
            <div class="card-content">
              <div class="card-label">Windows</div>
              <div class="card-value">{{ modpackInfo.survivalcraft.version.windows.version }}</div>
            </div>
          </div>

          <!-- Android 版本 -->
          <div v-if="modpackInfo.survivalcraft?.version?.android" class="info-card">
            <div class="card-content">
              <div class="card-label">Android</div>
              <div class="card-value">{{ modpackInfo.survivalcraft.version.android.version }}</div>
            </div>
          </div>

          <!-- 模组数量 -->
          <div v-if="modpackInfo.mods && modpackInfo.mods.length > 0" class="info-card">
            <div class="card-content">
              <div class="card-label">{{ t('installed.modpackMods') }}</div>
              <div class="card-value">{{ modpackInfo.mods.length }} {{ t('textures.unitCount') }}</div>
            </div>
          </div>

          <!-- 覆盖文件 -->
          <div v-if="modpackInfo.overrides" class="info-card">
            <div class="card-content">
              <div class="card-label">{{ t('installed.modpackOverrides') }}</div>
              <div class="card-value">{{ modpackInfo.overrides }}</div>
            </div>
          </div>

          <!-- 模组路径 -->
          <div v-if="modpackInfo.modPath" class="info-card">
            <div class="card-content">
              <div class="card-label">{{ t('installed.modpackPath') }}</div>
              <div class="card-value">{{ modpackInfo.modPath }}</div>
            </div>
          </div>
        </div>

        <!-- 模组详情（可折叠） -->
        <n-collapse v-if="modpackInfo.mods && modpackInfo.mods.length > 0" class="mod-details">
          <n-collapse-item :title="`${t('installed.modpackViewDetails')} (${modpackInfo.mods.length} ${t('textures.unitCount')})`">
            <n-list bordered size="small">
              <n-list-item v-for="(mod, index) in modpackInfo.mods" :key="index">
                <div class="mod-item">
                  <div class="mod-info">
                    <div class="mod-name">{{ mod.name || `Mod #${mod.projectID}` }}</div>
                    <div class="mod-version">{{ mod.version }}</div>
                  </div>
                  <n-space align="center">
                    <n-tag v-if="mod.modPath" size="small" type="info">{{ mod.modPath }}</n-tag>
                    <n-tag v-else size="small" type="info">{{ modpackInfo.modPath || '/Mods' }}</n-tag>
                    <n-tag v-if="mod.required" type="error" size="small">{{ t('installed.modpackModRequired') }}</n-tag>
                    <n-tag v-else type="default" size="small">{{ t('installed.modpackModOptional') }}</n-tag>
                  </n-space>
                </div>
              </n-list-item>
            </n-list>
          </n-collapse-item>
        </n-collapse>
        </template>
      </div>
    </n-spin>

    <template #footer>
      <n-space justify="end">
        <n-button @click="$emit('update:show', false)">
          {{ t('common.cancel') }}
        </n-button>
        <n-button type="primary" @click="handleInstall" :loading="installing">
          {{ t('installed.startInstall') }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  show: boolean
  modpackData: any
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  'install': []
}>()

const { t } = useI18n()

const showModal = ref(props.show)
const loading = ref(false)
const installing = ref(false)

// 从父组件接收已解析的信息
const modpackInfo = computed(() => props.modpackData)

// scbbs：图标（包内 base64 data URL 或 http 链接）
const iconSrc = computed(() => {
  const info: any = modpackInfo.value
  if (!info || info.format !== 'scbbs') return ''
  return info.iconData || info.icon || ''
})

// scbbs：按类型统计条目（只显示数量 > 0 的类型）
const itemStats = computed(() => {
  const info: any = modpackInfo.value
  if (!info || info.format !== 'scbbs' || !Array.isArray(info.items)) return []
  const counts: Record<string, number> = {}
  for (const item of info.items) {
    counts[item.type] = (counts[item.type] || 0) + 1
  }
  const order = ['mod', 'world', 'texture', 'skin', 'furniture', 'data']
  return order
    .filter((type) => counts[type])
    .map((type) => ({ type, label: itemTypeLabel(type), count: counts[type] }))
})

// scbbs：条目类型显示名
function itemTypeLabel(type: string): string {
  const map: Record<string, string> = {
    mod: t('installed.scbbsItemTypeMod'),
    world: t('installed.scbbsItemTypeWorld'),
    texture: t('installed.scbbsItemTypeTexture'),
    skin: t('installed.scbbsItemTypeSkin'),
    furniture: t('installed.scbbsItemTypeFurniture'),
    data: t('installed.scbbsItemTypeData'),
  }
  return map[type] || type
}

// scbbs：条目类型标签颜色
function itemTypeTagType(type: string): 'primary' | 'success' | 'info' | 'warning' | 'error' | 'default' {
  const map: Record<string, 'primary' | 'success' | 'info' | 'warning' | 'error' | 'default'> = {
    mod: 'primary',
    world: 'success',
    texture: 'info',
    skin: 'warning',
    furniture: 'error',
    data: 'default',
  }
  return map[type] || 'default'
}

// scbbs：条目来源显示名
function itemSourceLabel(source: string): string {
  const map: Record<string, string> = {
    network: t('installed.scbbsSourceNetwork'),
    embedded: t('installed.scbbsSourceEmbedded'),
    community: t('installed.scbbsSourceCommunity'),
  }
  return map[source] || source
}

watch(() => props.show, (newVal) => {
  showModal.value = newVal
})

async function handleInstall() {
  installing.value = true
  try {
    emit('install')
    emit('update:show', false)
  } finally {
    installing.value = false
  }
}
</script>

<style scoped>
.modpack-info {
  max-height: 500px;
  overflow-y: auto;
}

/* scbbs 格式头部（图标 + 基本信息） */
.scbbs-header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

.scbbs-icon {
  width: 72px;
  height: 72px;
  border-radius: 8px;
  object-fit: cover;
  border: 1px solid var(--n-border-color);
  flex-shrink: 0;
}

.section {
  margin-top: 16px;
}

.section h3 {
  margin: 0 0 8px 0;
  font-size: 14px;
  font-weight: 600;
}

/* 信息卡片布局 */
.info-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.info-card {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: var(--n-color-modal);
  border: 1px solid var(--n-border-color);
  border-radius: 8px;
  transition: all 0.2s;
}

.info-card:hover {
  border-color: var(--n-primary-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.card-content {
  flex: 1;
  width: 100%;
}

.card-label {
  font-size: 12px;
  color: var(--n-text-color-2);
  margin-bottom: 4px;
}

.card-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--n-text-color-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 模组详情 */
.mod-details {
  margin-top: 16px;
}

.mod-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.mod-info {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
}

.mod-name {
  font-weight: 500;
}

.mod-version {
  font-size: 12px;
  color: #7a7a7a;
}
</style>
