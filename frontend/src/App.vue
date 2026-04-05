<template>
  <div class="app-box">
    <!-- 背景图片层 -->
    <div v-if="backgroundImage" class="background-layer" :style="{ backgroundImage: `url(${backgroundImage})` }">
      <div class="background-overlay"></div>
    </div>

    <n-config-provider :theme="darkTheme">
      <n-message-provider>
        <n-dialog-provider ref="dialogProviderInst">
          <n-notification-provider>
            <div class="app-container">
              <!-- 固定在顶部的导航栏 -->
              <div class="app-nav">
                <n-tabs
                  v-model:value="activeTab"
                  type="line"
                  animated
                  @update:value="handleTabChange"
                >
                  <n-tab-pane name="home" :tab="t('nav.home')">
                    <HomeView />
                  </n-tab-pane>

                  <n-tab-pane name="installed" :tab="t('nav.installed')">
                    <InstalledVersionsView />
                  </n-tab-pane>

                  <n-tab-pane name="versions" :tab="t('nav.versions')">
                    <VersionsView />
                  </n-tab-pane>

                  <n-tab-pane name="mods" :tab="t('nav.mods')">
                    <ModsView />
                  </n-tab-pane>

                  <n-tab-pane name="savegames" :tab="t('nav.savegames')">
                    <SaveGamesView />
                  </n-tab-pane>

                  <n-tab-pane name="skins" :tab="t('nav.skins')">
                    <SkinsView />
                  </n-tab-pane>

                  <n-tab-pane name="settings" :tab="t('nav.settings')">
                    <SettingsView />
                  </n-tab-pane>
                </n-tabs>
              </div>
            </div>

            <!-- 回到顶部按钮 -->
            <BackToTop />
          </n-notification-provider>
        </n-dialog-provider>
      </n-message-provider>
    </n-config-provider>
  </div>
</template>

<script setup lang="ts">
import { ref, h, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { darkTheme, NAlert, NDialogProvider, NButton } from "naive-ui";
import { useGameStore } from "./stores/game";
import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
import { CheckUpdate, GetConfig, GetBackgroundImageBase64 } from "./api/config";
import HomeView from "./views/Home.vue";
import InstalledVersionsView from "./views/InstalledVersions.vue";
import VersionsView from "./views/Versions.vue";
import ModsView from "./views/Mods.vue";
import SaveGamesView from "./views/SaveGames.vue";
import SkinsView from "./views/Skins.vue";
import SettingsView from "./views/Settings.vue";
import BackToTop from "./components/BackToTop.vue";

const { t, locale } = useI18n();
const router = useRouter();
const gameStore = useGameStore();
const activeTab = ref("home");
const backgroundImage = ref("");

const dialogProviderInst = ref<InstanceType<typeof NDialogProvider> | null>(
  null,
);

// 加载背景图片和语言设置
async function loadBackgroundImage() {
  try {
    const config = await GetConfig();
    // 加载语言设置
    if (config?.language) {
      locale.value = config.language;
    }
    // 加载背景图片
    if (config?.backgroundImage) {
      const base64 = await GetBackgroundImageBase64();
      backgroundImage.value = base64;
    } else {
      backgroundImage.value = "";
    }
  } catch (error) {
    console.error("Failed to load background image:", error);
    backgroundImage.value = "";
  }
}

function handleTabChange(value: string) {
  router.push({ name: value.charAt(0).toUpperCase() + value.slice(1) });
}

// 处理游戏崩溃事件
function handleGameCrash(data: any) {
  const { versionName, exitCode, log, crashTime } = data;

  // 使用 dialogProvider 的实例
  if (dialogProviderInst.value) {
    const dialog = dialogProviderInst.value;
    // 通过 create 方法创建对话框
    // @ts-ignore
    dialog.create({
      title: t('home.gameCrash') || "游戏崩溃",
      content: () => {
        return h("div", [
          h(
            "p",
            { style: "margin-bottom: 12px; font-weight: bold;" },
            `${t('versions.version')}: ${versionName}`,
          ),
          h("p", { style: "margin-bottom: 12px;" }, `${t('installed.exitCode')}: ${exitCode}`),
          h("p", { style: "margin-bottom: 12px;" }, `${t('installed.crashTime')}: ${crashTime}`),
          h(
            NAlert,
            {
              type: "error",
              title: t('home.gameLog') || "运行日志",
            },
            {
              default: () =>
                h(
                  "pre",
                  {
                    style:
                      "max-height: 300px; overflow-y: auto; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-size: 12px; white-space: pre-wrap; word-wrap: break-word;",
                  },
                  log,
                ),
            },
          ),
        ]);
      },
      positiveText: t('common.confirm'),
    });
  }
}

onMounted(async () => {
  // 加载背景图片
  await loadBackgroundImage();

  // 监听游戏崩溃事件
  EventsOn("game:crashed", handleGameCrash);

  // 监听路由变化
  router.afterEach((to) => {
    activeTab.value = to.name?.toString().toLowerCase() || "home";

    // 当从设置页面离开时，重新加载背景图片
    if (to.name !== "Settings") {
      loadBackgroundImage();
    }
  });

  // 初始化时检查游戏状态
  try {
    await gameStore.updateStatus();
    await gameStore.updateProcessInfo();

    // 如果游戏正在运行，启动状态检查
    if (gameStore.status === "running") {
      gameStore.startStatusCheck?.();
    }

    // 检查更新
    try {
      const updateInfo = await CheckUpdate();
      console.log("[Update Check] Update info:", updateInfo);

      if (updateInfo.hasUpdate) {
        // 有新版本，显示更新对话框
        if (dialogProviderInst.value) {
          const dialog = dialogProviderInst.value;
          // @ts-ignore
          dialog.create({
            title: t('settings.updateAvailable') || "发现新版本",
            content: () => {
              return h("div", [
                h("p", { style: "margin-bottom: 12px;" }, `${t('settings.currentVersion')}: ${updateInfo.currentVersion}`),
                h("p", { style: "margin-bottom: 12px; font-weight: bold; color: #18a058;" }, `${t('settings.latestVersion')}: ${updateInfo.latestVersion}`),
                h("p", { style: "margin-bottom: 12px;" }, `${t('settings.releaseDate')}: ${new Date(updateInfo.publishedAt).toLocaleString()}`),
                h(NAlert, {
                  type: "info",
                  title: t('settings.updateContent') || "更新内容"
                }, {
                  default: () => h("pre", {
                    style: "max-height: 200px; overflow-y: auto; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-size: 12px; white-space: pre-wrap;"
                  }, updateInfo.body || t('settings.noUpdateContent') || "暂无更新说明")
                })
              ]);
            },
            positiveText: t('settings.goToDownload') || "前往下载",
            negativeText: t('common.later') || "稍后提醒",
            onPositiveClick: () => {
              // 打开 GitHub releases 页面
              window.open(updateInfo.url, "_blank");
            }
          });
        }
      } else {
        console.log("[Update Check] No update available");
      }
    } catch (e) {
      // 检查更新失败，不影响使用
      console.warn("Failed to check for updates:", e);
    }
  } catch (e) {
    console.error("Failed to initialize game status:", e);
  }
});

onUnmounted(() => {
  // 移除崩溃事件监听
  EventsOff("game:crashed");
});
</script>

<style>
.app-box{
  width: 100vw;
  height: 100vh;
}


* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family:
    -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu,
    Cantarell, "Helvetica Neue", sans-serif;
}

.app-container {
  width: 100vw;
  height: 100vh;
  overflow-y: auto;
}

/* 固定在顶部的导航栏 */
.app-nav {
  position: sticky;
  top: 0;
  width: 100%;
  padding: 20px;
  background-color: var(--n-color);
  z-index: 1000;
  backdrop-filter: blur(10px);
}

/* 隐藏滚动条但保持滚动功能 */
.app-container::-webkit-scrollbar {
  width: 8px;
}

.app-container::-webkit-scrollbar-track {
  background: transparent;
}

.app-container::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 4px;
}

.app-container::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

/* 背景图片层 */
.background-layer {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  z-index: 0;
  pointer-events: none;
}

/* 背景图片遮罩层 - 降低透明度 */
.background-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5); /* 50% 黑色遮罩，可根据需要调整 */
  backdrop-filter: blur(2px); /* 轻微模糊效果 */
}

/* 确保内容在背景之上 */
.app-box,
.n-config-provider,
.n-message-provider,
.n-dialog-provider,
.n-notification-provider,
.app-container {
  position: relative;
  z-index: 1;
}
</style>
