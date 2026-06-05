<template>
  <n-modal
    v-model:show="show"
    preset="dialog"
    :title="dialogTitle"
    :positive-text="t('common.confirm')"
    :negative-text="t('common.cancel')"
    @positive-click="handleConfirm"
    @negative-click="handleCancel"
    @close="handleCancel"
  >
    <div class="version-select-content">
      <p class="file-info">
        {{ t('dragDrop.file') }}: <strong>{{ fileName }}</strong>
      </p>
      <n-select
        v-model:value="selectedVersionId"
        :options="versionOptions"
        :placeholder="t('dragDrop.selectVersion')"
        filterable
      />
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useVersionStore } from '../stores/version'

interface Props {
  show: boolean
  resourceType: string
  fileName: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  'select': [versionId: string]
  'cancel': []
}>()

const { t } = useI18n()
const versionStore = useVersionStore()
const selectedVersionId = ref<string>('')

// 控制弹窗显示
const show = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

// 资源类型名称
const resourceTypeName = computed(() => {
  const typeNames: Record<string, string> = {
    'mod': t('dragDrop.types.mod'),
    'texture': t('dragDrop.types.texture'),
    'furniture': t('dragDrop.types.furniture'),
    'savegame': t('dragDrop.types.savegame'),
    'skin': t('dragDrop.types.skin'),
    'modpack': t('dragDrop.types.modpack'),
  }
  return typeNames[props.resourceType] || ''
})

// 弹窗标题
const dialogTitle = computed(() => {
  return t('dragDrop.selectVersionTitle', { type: resourceTypeName.value })
})

// 版本选项
const versionOptions = computed(() => {
  return versionStore.installedVersions
    .filter(v => v.pathExists !== false && v.pathExists !== undefined)
    .map(v => ({
      label: v.name || v.id,
      value: v.id,
    }))
})

// 确认选择
function handleConfirm() {
  if (!selectedVersionId.value) {
    return false // 阻止关闭
  }
  emit('select', selectedVersionId.value)
  selectedVersionId.value = ''
  return true
}

// 取消选择
function handleCancel() {
  emit('cancel')
  selectedVersionId.value = ''
}

// 弹窗打开时加载版本列表
watch(() => props.show, async (newVal) => {
  if (newVal) {
    await versionStore.getVersions()
    await versionStore.getPrimaryVersion()

    // 默认选中主版本
    if (versionStore.primaryVersion) {
      selectedVersionId.value = versionStore.primaryVersion.id
    } else if (versionOptions.value.length > 0) {
      selectedVersionId.value = versionOptions.value[0].value
    }
  }
})

onMounted(async () => {
  await versionStore.getVersions()
  await versionStore.getPrimaryVersion()
})
</script>

<style scoped>
.version-select-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.file-info {
  font-size: 14px;
  color: var(--color-text-secondary, #DBDEE1);
}

.file-info strong {
  color: var(--color-text-primary, #F2F3F5);
}
</style>
