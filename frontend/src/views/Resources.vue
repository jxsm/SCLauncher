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
        <ModsView />
      </n-tab-pane>

      <n-tab-pane name="savegames" :tab="t('nav.savegames')">
        <SaveGamesView />
      </n-tab-pane>

      <n-tab-pane name="skins" :tab="t('nav.skins')">
        <SkinsView />
      </n-tab-pane>

      <n-tab-pane name="furniture" :tab="t('nav.furniture')">
        <FurnitureView />
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import ModsView from "./Mods.vue";
import SaveGamesView from "./SaveGames.vue";
import SkinsView from "./Skins.vue";
import FurnitureView from "./Furniture.vue";

const { t } = useI18n();
const activeResourceTab = ref("mods");
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
