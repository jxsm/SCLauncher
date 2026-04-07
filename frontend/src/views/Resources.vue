<template>
  <div class="resources-container">
    <!-- 子导航栏 -->
    <n-tabs
      v-model:value="activeResourceTab"
      type="segment"
      animated
      class="resource-tabs"
    >
      <n-tab-pane name="mods" :tab="t('nav.mods')">
        <ModsView :version-id-from-route="versionIdFromRoute" />
      </n-tab-pane>

      <n-tab-pane name="furniture" :tab="t('nav.furniture')">
        <FurnitureView :version-id-from-route="versionIdFromRoute" />
      </n-tab-pane>

      <n-tab-pane name="textures" :tab="t('nav.textures')">
        <TexturesView :version-id-from-route="versionIdFromRoute" />
      </n-tab-pane>

      <n-tab-pane name="savegames" :tab="t('nav.savegames')">
        <SaveGamesView :version-id-from-route="versionIdFromRoute" />
      </n-tab-pane>

      <n-tab-pane name="skins" :tab="t('nav.skins')">
        <SkinsView :version-id-from-route="versionIdFromRoute" />
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import ModsView from "./Mods.vue";
import SaveGamesView from "./SaveGames.vue";
import SkinsView from "./Skins.vue";
import FurnitureView from "./Furniture.vue";
import TexturesView from "./Textures.vue";

const { t } = useI18n();
const route = useRoute();
const activeResourceTab = ref("mods");

// 从路由获取版本ID
const versionIdFromRoute = computed(() => route.query.versionId as string | undefined);
</script>

<style scoped>
.resources-container {
  width: 100%;
  height: 100%;
}

.resource-tabs {
  margin-bottom: 16px;
}
</style>
