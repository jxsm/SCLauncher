<template>
  <div class="app-box">
    <!-- 背景图片层 -->
    <div v-if="backgroundImage" class="background-layer" :style="{ backgroundImage: `url(${backgroundImage})` }">
      <div class="background-overlay"></div>
    </div>

    <n-config-provider :theme="naiveTheme" :theme-overrides="themeOverrides">
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

                  <n-tab-pane name="resources" :tab="t('nav.resources')">
                    <ResourcesView />
                  </n-tab-pane>

                  <n-tab-pane name="settings" :tab="t('nav.settings')">
                    <SettingsView />
                  </n-tab-pane>
                </n-tabs>
              </div>
            </div>

            <!-- 回到顶部按钮 -->
            <BackToTop />

            <!-- 下载进度条 -->
            <DownloadProgress />
          </n-notification-provider>
        </n-dialog-provider>
      </n-message-provider>
    </n-config-provider>
  </div>
</template>

<script setup lang="ts">
import { ref, h, onMounted, onUnmounted, computed, watch, provide } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { darkTheme, NAlert, NDialogProvider, NButton, type GlobalThemeOverrides } from "naive-ui";
import { useGameStore } from "./stores/game";
import { useDownloadStore } from "./stores/download";
import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
import { CheckUpdate, CheckUpdateForce, SetUpdateRemindDisabled, GetConfig, GetBackgroundImageBase64, SetTheme as SetThemeAPI } from "./api/config";
import HomeView from "./views/Home.vue";
import InstalledVersionsView from "./views/InstalledVersions.vue";
import VersionsView from "./views/Versions.vue";
import ResourcesView from "./views/Resources.vue";
import SettingsView from "./views/Settings.vue";
import BackToTop from "./components/BackToTop.vue";
import DownloadProgress from "./components/DownloadProgress.vue";

const { t, locale } = useI18n();
const router = useRouter();
const gameStore = useGameStore();
const downloadStore = useDownloadStore();
const activeTab = ref("home");
const backgroundImage = ref("");
const dontRemindCheckbox = ref(false);
const hasBackgroundImage = ref(false);

// ==================== 主题系统 ====================
const themeMode = ref<'light' | 'dark' | 'auto'>('auto');
const systemPrefersDark = ref(false);
const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

const resolvedTheme = computed(() => {
  if (themeMode.value === 'auto') {
    return systemPrefersDark.value ? 'dark' : 'light';
  }
  return themeMode.value;
});

// 提供主题状态给子组件
provide('themeMode', themeMode);
provide('resolvedTheme', resolvedTheme);
provide('setTheme', async (mode: 'light' | 'dark' | 'auto') => {
  themeMode.value = mode;
  try {
    await SetThemeAPI(mode);
  } catch (e) {
    console.error('Failed to save theme:', e);
  }
});

function handleSystemThemeChange(e: MediaQueryListEvent) {
  systemPrefersDark.value = e.matches;
}

// Apple Design System 主题配置 - 浅色
const lightAppleTheme: GlobalThemeOverrides = {
  common: {
    primaryColor: '#0066cc',
    primaryColorHover: '#0071e3',
    primaryColorPressed: '#0055aa',
    primaryColorSuppl: '#2997ff',
    infoColor: '#0066cc',
    successColor: '#34c759',
    warningColor: '#ff9500',
    errorColor: '#ff3b30',
    textColorBase: '#1d1d1f',
    textColor1: '#1d1d1f',
    textColor2: '#333333',
    textColor3: '#7a7a7a',
    dividerColor: '#e0e0e0',
    borderColor: '#e0e0e0',
    borderRadius: '8px',
    borderRadiusSmall: '5px',
    fontFamily: '"SF Pro Text", "SF Pro Display", system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Fira Sans", "Droid Sans", "Helvetica Neue", sans-serif',
    fontSize: '17px',
    fontSizeMini: '12px',
    fontSizeTiny: '12px',
    fontSizeSmall: '14px',
    fontSizeMedium: '17px',
    fontSizeLarge: '17px',
    fontSizeHuge: '21px',
    heightMini: '28px',
    heightTiny: '32px',
    heightSmall: '36px',
    heightMedium: '40px',
    heightLarge: '44px',
    heightHuge: '48px',
    lineHeight: '1.47',
    hoverColor: 'rgba(0, 102, 204, 0.08)',
    cardColor: '#ffffff',
    modalColor: '#ffffff',
    bodyColor: '#ffffff',
    tagColor: '#f5f5f7',
    avatarColor: '#f5f5f7',
    inputColor: '#ffffff',
    inputColorDisabled: '#f5f5f7',
    tableColor: '#ffffff',
    tableColorHover: '#f5f5f7',
    codeColor: '#f5f5f7',
    tabColor: '#ffffff',
    closeIconColor: '#7a7a7a',
    closeIconColorHover: '#1d1d1f',
    closeColorHover: 'rgba(0, 0, 0, 0.06)',
    closeColorPressed: 'rgba(0, 0, 0, 0.1)',
    clearColor: '#7a7a7a',
    clearColorHover: '#1d1d1f',
    clearColorPressed: '#333333',
    scrollbarColor: 'rgba(0, 0, 0, 0.2)',
    scrollbarColorHover: 'rgba(0, 0, 0, 0.35)',
    progressRailColor: '#e0e0e0',
    railColor: '#e0e0e0',
    popoverColor: '#ffffff',
    tableHeaderColor: '#f5f5f7',
    invertedColor: '#1d1d1f',
  },
  Button: {
    textColorPrimary: '#ffffff',
    textColorHoverPrimary: '#ffffff',
    textColorPressedPrimary: '#ffffff',
    textColorFocusPrimary: '#ffffff',
    colorPrimary: '#0066cc',
    colorHoverPrimary: '#0071e3',
    colorPressedPrimary: '#0055aa',
    colorFocusPrimary: '#0066cc',
    borderPrimary: '#0066cc',
    borderHoverPrimary: '#0071e3',
    borderPressedPrimary: '#0055aa',
    borderFocusPrimary: '#0066cc',
    borderRadiusMedium: '9999px',
    borderRadiusLarge: '9999px',
    fontWeight: '400',
  },
  Card: {
    borderRadius: '18px',
    borderColor: '#e0e0e0',
    color: '#ffffff',
    colorModal: '#ffffff',
    colorPopover: '#ffffff',
    titleFontSizeMedium: '17px',
    titleFontWeight: '600',
    paddingMedium: '24px',
  },
  Input: {
    borderRadius: '9999px',
    borderHover: '1px solid #0066cc',
    borderFocus: '2px solid #0066cc',
    boxShadowFocus: '0 0 0 4px rgba(0, 102, 204, 0.15)',
    color: '#ffffff',
    colorDisabled: '#f5f5f7',
    heightMedium: '40px',
    heightLarge: '44px',
  },
  Select: {
    peers: {
      InternalSelection: {
        borderRadius: '9999px',
        borderHover: '1px solid #0066cc',
        borderFocus: '2px solid #0066cc',
        boxShadowFocus: '0 0 0 4px rgba(0, 102, 204, 0.15)',
        heightMedium: '40px',
      }
    }
  },
  Tag: {
    borderRadius: '9999px',
    heightSmall: '24px',
    heightMedium: '28px',
    fontSizeSmall: '12px',
    fontSizeMedium: '14px',
    fontWeight: '400',
  },
  Tabs: {
    tabFontWeightActive: '600',
    tabFontWeight: '400',
    barColor: '#0066cc',
    tabTextColorActiveLine: '#1d1d1f',
    tabTextColorLine: '#7a7a7a',
    tabTextColorHoverLine: '#1d1d1f',
    tabFontSizeMedium: '17px',
    tabFontSizeLarge: '21px',
  },
  Dialog: {
    borderRadius: '18px',
    titleFontSize: '21px',
    fontSize: '17px',
    padding: '24px',
  },
  Notification: {
    borderRadius: '18px',
    padding: '24px',
  },
  Message: {
    borderRadius: '9999px',
    padding: '12px 20px',
  },
  List: {
    borderRadius: '18px',
    color: '#ffffff',
    colorModal: '#ffffff',
  },
  DataTable: {
    borderRadius: '18px',
    thColor: '#f5f5f7',
  },
  Pagination: {
    itemBorderRadius: '9999px',
    itemSize: '36px',
  },
  Switch: {
    railColor: '#e0e0e0',
    railColorActive: '#0066cc',
  },
  Checkbox: {
    colorChecked: '#0066cc',
    borderChecked: '1px solid #0066cc',
  },
  Radio: {
    dotColorActive: '#0066cc',
    buttonBorderColorActive: '#0066cc',
    buttonColorActive: '#0066cc',
    buttonTextColorActive: '#ffffff',
  },
  Alert: {
    borderRadius: '18px',
  },
  Spin: {
    color: '#0066cc',
  },
  Empty: {
    textColor: '#7a7a7a',
  },
};

// Apple Design System 主题配置 - 深色
const darkAppleTheme: GlobalThemeOverrides = {
  common: {
    primaryColor: '#2997ff',
    primaryColorHover: '#40a3ff',
    primaryColorPressed: '#1a8af0',
    primaryColorSuppl: '#2997ff',
    infoColor: '#2997ff',
    successColor: '#30d158',
    warningColor: '#ff9f0a',
    errorColor: '#ff453a',
    textColorBase: '#ffffff',
    textColor1: '#ffffff',
    textColor2: '#cccccc',
    textColor3: '#888888',
    dividerColor: '#38383a',
    borderColor: '#38383a',
    borderRadius: '8px',
    borderRadiusSmall: '5px',
    fontFamily: '"SF Pro Text", "SF Pro Display", system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Fira Sans", "Droid Sans", "Helvetica Neue", sans-serif',
    fontSize: '17px',
    fontSizeMini: '12px',
    fontSizeTiny: '12px',
    fontSizeSmall: '14px',
    fontSizeMedium: '17px',
    fontSizeLarge: '17px',
    fontSizeHuge: '21px',
    heightMini: '28px',
    heightTiny: '32px',
    heightSmall: '36px',
    heightMedium: '40px',
    heightLarge: '44px',
    heightHuge: '48px',
    lineHeight: '1.47',
    hoverColor: 'rgba(41, 151, 255, 0.12)',
    cardColor: '#1c1c1e',
    modalColor: '#2c2c2e',
    bodyColor: '#000000',
    tagColor: '#2c2c2e',
    avatarColor: '#2c2c2e',
    inputColor: '#1c1c1e',
    inputColorDisabled: '#2c2c2e',
    tableColor: '#1c1c1e',
    tableColorHover: '#2c2c2e',
    codeColor: '#2c2c2e',
    tabColor: '#1c1c1e',
    closeIconColor: '#888888',
    closeIconColorHover: '#ffffff',
    closeColorHover: 'rgba(255, 255, 255, 0.08)',
    closeColorPressed: 'rgba(255, 255, 255, 0.12)',
    clearColor: '#888888',
    clearColorHover: '#ffffff',
    clearColorPressed: '#cccccc',
    scrollbarColor: 'rgba(255, 255, 255, 0.2)',
    scrollbarColorHover: 'rgba(255, 255, 255, 0.35)',
    progressRailColor: '#38383a',
    railColor: '#38383a',
    popoverColor: '#2c2c2e',
    tableHeaderColor: '#2c2c2e',
    invertedColor: '#ffffff',
  },
  Button: {
    textColorPrimary: '#ffffff',
    textColorHoverPrimary: '#ffffff',
    textColorPressedPrimary: '#ffffff',
    textColorFocusPrimary: '#ffffff',
    colorPrimary: '#2997ff',
    colorHoverPrimary: '#40a3ff',
    colorPressedPrimary: '#1a8af0',
    colorFocusPrimary: '#2997ff',
    borderPrimary: '#2997ff',
    borderHoverPrimary: '#40a3ff',
    borderPressedPrimary: '#1a8af0',
    borderFocusPrimary: '#2997ff',
    borderRadiusMedium: '9999px',
    borderRadiusLarge: '9999px',
    fontWeight: '400',
  },
  Card: {
    borderRadius: '18px',
    borderColor: '#38383a',
    color: '#1c1c1e',
    colorModal: '#2c2c2e',
    colorPopover: '#2c2c2e',
    titleFontSizeMedium: '17px',
    titleFontWeight: '600',
    paddingMedium: '24px',
  },
  Input: {
    borderRadius: '9999px',
    borderHover: '1px solid #2997ff',
    borderFocus: '2px solid #2997ff',
    boxShadowFocus: '0 0 0 4px rgba(41, 151, 255, 0.2)',
    color: '#1c1c1e',
    colorDisabled: '#2c2c2e',
    heightMedium: '40px',
    heightLarge: '44px',
  },
  Select: {
    peers: {
      InternalSelection: {
        borderRadius: '9999px',
        borderHover: '1px solid #2997ff',
        borderFocus: '2px solid #2997ff',
        boxShadowFocus: '0 0 0 4px rgba(41, 151, 255, 0.2)',
        heightMedium: '40px',
      }
    }
  },
  Tag: {
    borderRadius: '9999px',
    heightSmall: '24px',
    heightMedium: '28px',
    fontSizeSmall: '12px',
    fontSizeMedium: '14px',
    fontWeight: '400',
  },
  Tabs: {
    tabFontWeightActive: '600',
    tabFontWeight: '400',
    barColor: '#2997ff',
    tabTextColorActiveLine: '#ffffff',
    tabTextColorLine: '#888888',
    tabTextColorHoverLine: '#ffffff',
    tabFontSizeMedium: '17px',
    tabFontSizeLarge: '21px',
  },
  Dialog: {
    borderRadius: '18px',
    titleFontSize: '21px',
    fontSize: '17px',
    padding: '24px',
  },
  Notification: {
    borderRadius: '18px',
    padding: '24px',
  },
  Message: {
    borderRadius: '9999px',
    padding: '12px 20px',
  },
  List: {
    borderRadius: '18px',
    color: '#1c1c1e',
    colorModal: '#2c2c2e',
  },
  DataTable: {
    borderRadius: '18px',
    thColor: '#2c2c2e',
  },
  Pagination: {
    itemBorderRadius: '9999px',
    itemSize: '36px',
  },
  Switch: {
    railColor: '#38383a',
    railColorActive: '#2997ff',
  },
  Checkbox: {
    colorChecked: '#2997ff',
    borderChecked: '1px solid #2997ff',
  },
  Radio: {
    dotColorActive: '#2997ff',
    buttonBorderColorActive: '#2997ff',
    buttonColorActive: '#2997ff',
    buttonTextColorActive: '#ffffff',
  },
  Alert: {
    borderRadius: '18px',
  },
  Spin: {
    color: '#2997ff',
  },
  Empty: {
    textColor: '#888888',
  },
};

// 根据主题和背景图片动态设置主题覆盖
const themeOverrides = computed<GlobalThemeOverrides>(() => {
  const baseTheme = resolvedTheme.value === 'dark' ? darkAppleTheme : lightAppleTheme;

  if (hasBackgroundImage.value) {
    const isDark = resolvedTheme.value === 'dark';
    return {
      common: {
        ...baseTheme.common,
        color: isDark ? 'rgba(0, 0, 0, 0.75)' : 'rgba(255, 255, 255, 0.75)',
        cardColor: isDark ? 'rgba(0, 0, 0, 0.75)' : 'rgba(255, 255, 255, 0.75)',
        modalColor: isDark ? 'rgba(0, 0, 0, 0.85)' : 'rgba(255, 255, 255, 0.85)',
        bodyColor: isDark ? 'rgba(0, 0, 0, 0.75)' : 'rgba(255, 255, 255, 0.75)',
      },
      ...baseTheme,
    };
  }
  return baseTheme;
});

// Naive UI 主题 (darkTheme 或 undefined)
const naiveTheme = computed(() => resolvedTheme.value === 'dark' ? darkTheme : undefined);

// CSS 自定义属性注入
watch(resolvedTheme, (theme) => {
  const root = document.documentElement;
  const isDark = theme === 'dark';

  if (isDark) {
    root.classList.remove('theme-light');
    root.classList.add('theme-dark');
  } else {
    root.classList.remove('theme-dark');
    root.classList.add('theme-light');
  }

  const vars: Record<string, string> = isDark ? {
    '--color-bg': '#000000',
    '--color-surface': '#1c1c1e',
    '--color-surface-elevated': '#2c2c2e',
    '--color-text-primary': '#ffffff',
    '--color-text-secondary': '#cccccc',
    '--color-text-tertiary': '#888888',
    '--color-border': '#38383a',
    '--color-primary': '#2997ff',
    '--color-primary-hover': '#40a3ff',
    '--color-primary-hover-bg': 'rgba(41, 151, 255, 0.12)',
    '--color-error': '#ff453a',
    '--color-error-hover': '#ff6961',
    '--color-scroll-thumb': 'rgba(255, 255, 255, 0.2)',
    '--color-scroll-thumb-hover': 'rgba(255, 255, 255, 0.35)',
    '--color-overlay': 'rgba(0, 0, 0, 0.5)',
    '--color-nav-bg': 'rgba(28, 28, 30, 0.72)',
  } : {
    '--color-bg': '#ffffff',
    '--color-surface': '#ffffff',
    '--color-surface-elevated': '#f5f5f7',
    '--color-text-primary': '#1d1d1f',
    '--color-text-secondary': '#333333',
    '--color-text-tertiary': '#7a7a7a',
    '--color-border': '#e0e0e0',
    '--color-primary': '#0066cc',
    '--color-primary-hover': '#0071e3',
    '--color-primary-hover-bg': 'rgba(0, 102, 204, 0.08)',
    '--color-error': '#ff3b30',
    '--color-error-hover': '#ff453a',
    '--color-scroll-thumb': 'rgba(0, 0, 0, 0.2)',
    '--color-scroll-thumb-hover': 'rgba(0, 0, 0, 0.35)',
    '--color-overlay': 'rgba(255, 255, 255, 0.5)',
    '--color-nav-bg': 'rgba(255, 255, 255, 0.72)',
  };

  for (const [key, value] of Object.entries(vars)) {
    root.style.setProperty(key, value);
  }
}, { immediate: true });

const dialogProviderInst = ref<InstanceType<typeof NDialogProvider> | null>(
  null,
);

// 加载背景图片、语言和主题设置
async function loadBackgroundImage() {
  try {
    const config = await GetConfig();
    // 加载语言设置
    if (config?.language) {
      locale.value = config.language;
    }
    // 加载主题设置
    if (config?.theme && ['light', 'dark', 'auto'].includes(config.theme)) {
      themeMode.value = config.theme as 'light' | 'dark' | 'auto';
    }
    // 加载背景图片
    if (config?.backgroundImage) {
      const base64 = await GetBackgroundImageBase64();
      backgroundImage.value = base64;
      hasBackgroundImage.value = true;
    } else {
      backgroundImage.value = "";
      hasBackgroundImage.value = false;
    }
  } catch (error) {
    console.error("Failed to load background image:", error);
    backgroundImage.value = "";
    hasBackgroundImage.value = false;
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
                      "max-height: 300px; overflow-y: auto; background: var(--n-code-color); color: var(--n-text-color); padding: 12px; border-radius: 8px; font-size: 12px; white-space: pre-wrap; word-wrap: break-word; border: 1px solid var(--n-border-color); font-family: 'SF Mono', SFMono-Regular, Menlo, Monaco, Consolas, monospace;",
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
  // 初始化系统主题检测
  systemPrefersDark.value = mediaQuery.matches;
  mediaQuery.addEventListener('change', handleSystemThemeChange);

  // 初始化下载进度监听
  downloadStore.initEventListeners();

  // 加载背景图片、语言和主题
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

      // 如果是因为用户设置了不再提醒而跳过的，不显示对话框
      if (updateInfo.skipped) {
        console.log("[Update Check] Skipped due to user preference");
        return;
      }

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
                h("p", { style: "margin-bottom: 12px; font-weight: bold; color: var(--n-primary-color);" }, `${t('settings.latestVersion')}: ${updateInfo.latestVersion}`),
                h("p", { style: "margin-bottom: 12px;" }, `${t('settings.releaseDate')}: ${new Date(updateInfo.publishedAt).toLocaleString()}`),
                h(NAlert, {
                  type: "info",
                  title: t('settings.updateContent') || "更新内容"
                }, {
                  default: () => h("pre", {
                    style: "max-height: 200px; overflow-y: auto; background: var(--n-code-color); color: var(--n-text-color); padding: 12px; border-radius: 8px; font-size: 12px; white-space: pre-wrap; border: 1px solid var(--n-border-color); font-family: 'SF Mono', SFMono-Regular, Menlo, Monaco, Consolas, monospace;"
                  }, updateInfo.body || t('settings.noUpdateContent') || "暂无更新说明")
                }),
                h("div", { style: "margin-top: 16px;" }, [
                  h("input", {
                    type: "checkbox",
                    id: "dont-remind-checkbox",
                    style: "margin-right: 8px;",
                    onChange: (e: any) => {
                      dontRemindCheckbox.value = e.target.checked;
                    }
                  }),
                  h("label", {
                    for: "dont-remind-checkbox",
                    style: "cursor: pointer;"
                  }, t('settings.dontRemindFor30Days') || "30天内不再提醒")
                ])
              ]);
            },
            positiveText: t('settings.goToDownload') || "前往下载",
            negativeText: t('common.cancel') || "取消",
            onPositiveClick: () => {
              // 打开 GitHub releases 页面
              window.open(updateInfo.url, "_blank");
            },
            onNegativeClick: async () => {
              // 如果用户勾选了"30天内不再提醒"，设置不再提醒
              if (dontRemindCheckbox.value) {
                try {
                  await SetUpdateRemindDisabled(true);
                  console.log("[Update Check] Update reminders disabled for 30 days");
                } catch (error) {
                  console.error("[Update Check] Failed to disable update reminders:", error);
                }
              }
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

  // 移除下载进度监听
  downloadStore.removeEventListeners();

  // 移除系统主题监听
  mediaQuery.removeEventListener('change', handleSystemThemeChange);
});
</script>

<style>
.app-box {
  width: 100vw;
  height: 100vh;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: "SF Pro Text", "SF Pro Display", system-ui, -apple-system, BlinkMacSystemFont,
    "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Fira Sans", "Droid Sans",
    "Helvetica Neue", sans-serif;
  background-color: var(--color-bg, #ffffff);
  color: var(--color-text-primary, #1d1d1f);
  transition: background-color 0.3s ease, color 0.3s ease;
}

.app-container {
  width: 100vw;
  height: 100vh;
  overflow-y: auto;
}

/* Apple 风格顶部导航栏 */
.app-nav {
  position: sticky;
  top: 0;
  width: 100%;
  padding: 12px 24px;
  background-color: var(--color-nav-bg, rgba(255, 255, 255, 0.72));
  backdrop-filter: saturate(180%) blur(20px);
  -webkit-backdrop-filter: saturate(180%) blur(20px);
  border-bottom: 1px solid var(--color-border, #e0e0e0);
  z-index: 1000;
}

/* 滚动条样式 */
.app-container::-webkit-scrollbar {
  width: 8px;
}

.app-container::-webkit-scrollbar-track {
  background: transparent;
}

.app-container::-webkit-scrollbar-thumb {
  background: var(--color-scroll-thumb, rgba(0, 0, 0, 0.2));
  border-radius: 4px;
}

.app-container::-webkit-scrollbar-thumb:hover {
  background: var(--color-scroll-thumb-hover, rgba(0, 0, 0, 0.35));
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

/* 背景图片遮罩层 */
.background-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: var(--color-overlay, rgba(255, 255, 255, 0.5));
  backdrop-filter: saturate(180%) blur(20px);
  -webkit-backdrop-filter: saturate(180%) blur(20px);
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

/* 全局过渡动画 */
.n-button {
  transition: all 0.2s ease !important;
}

.n-button:active {
  transform: scale(0.95);
}

/* Apple 风格卡片 */
.n-card {
  border: 1px solid var(--color-border, #e0e0e0);
  box-shadow: none;
  transition: border-color 0.3s ease, background-color 0.3s ease;
}

/* Apple 风格列表项 */
.n-list-item {
  transition: background-color 0.2s ease;
}

.n-list-item:hover {
  background-color: var(--color-surface-elevated, #f5f5f7) !important;
}

/* Apple 风格标签 */
.n-tag {
  font-weight: 400;
}
</style>
