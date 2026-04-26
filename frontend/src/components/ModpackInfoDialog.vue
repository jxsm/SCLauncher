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
              <div class="card-value">{{ modpackInfo.mods.length }} 个</div>
            </div>
          </div>

          <!-- 覆盖文件 -->
          <div v-if="modpackInfo.overrides" class="info-card">
            <div class="card-content">
              <div class="card-label">{{ t('installed.modpackOverrides') }}</div>
              <div class="card-value">{{ modpackInfo.overrides }}</div>
            </div>
          </div>
        </div>

        <!-- 模组详情（可折叠） -->
        <n-collapse v-if="modpackInfo.mods && modpackInfo.mods.length > 0" class="mod-details">
          <n-collapse-item :title="`查看模组详情 (${modpackInfo.mods.length} 个)`">
            <n-list bordered size="small">
              <n-list-item v-for="(mod, index) in modpackInfo.mods" :key="index">
                <div class="mod-item">
                  <div class="mod-name">{{ mod.name || `Mod #${mod.projectID}` }}</div>
                  <div class="mod-version">{{ mod.version }}</div>
                  <n-tag v-if="mod.required" type="error" size="small">{{ t('installed.modpackModRequired') }}</n-tag>
                  <n-tag v-else type="default" size="small">{{ t('installed.modpackModOptional') }}</n-tag>
                </div>
              </n-list-item>
            </n-list>
          </n-collapse-item>
        </n-collapse>
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
import { useMessage } from 'naive-ui'

const props = defineProps<{
  show: boolean
  modpackData: any
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  'install': []
}>()

const { t } = useI18n()
const message = useMessage()

const showModal = ref(props.show)
const loading = ref(false)
const installing = ref(false)

// 从父组件接收已解析的信息
const modpackInfo = computed(() => props.modpackData)

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
  gap: 8px;
}

.mod-name {
  flex: 1;
  font-weight: 500;
}

.mod-version {
  font-size: 12px;
  color: #666;
}
</style>
