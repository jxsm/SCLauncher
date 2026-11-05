<template>
  <div class="app-box">
    <!-- 背景图片层（模糊作用在图片层本体上，通过负 inset 外扩遮住边缘晕开） -->
    <div
      v-if="backgroundImage"
      class="background-layer"
      :style="{ backgroundImage: `url(${backgroundImage})` }"
    >
      <div class="background-overlay"></div>
    </div>

    <n-config-provider :theme="naiveTheme" :theme-overrides="themeOverrides">
      <n-message-provider>
        <n-dialog-provider ref="dialogProviderInst">
          <n-notification-provider>
            <!-- 拖拽文件处理 -->
            <DragDropHandler />

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

            <!-- .NET 运行时安装进度弹窗 -->
            <RuntimeInstallDialog />
          </n-notification-provider>
        </n-dialog-provider>
      </n-message-provider>
    </n-config-provider>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted, onUnmounted, computed, watch, provide } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { darkTheme, NAlert, NDialogProvider, NButton, type GlobalThemeOverrides } from "naive-ui";
import { useGameStore } from "./stores/game";
import { useDownloadStore } from "./stores/download";
import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
import { CheckUpdate, CheckUpdateForce, SetUpdateRemindDisabled, GetConfig, GetBackgroundImageBase64, SetBackgroundEffects, SetTheme as SetThemeAPI } from "./api/config";
import HomeView from "./views/Home.vue";
import InstalledVersionsView from "./views/InstalledVersions.vue";
import VersionsView from "./views/Versions.vue";
import ResourcesView from "./views/Resources.vue";
import SettingsView from "./views/Settings.vue";
import BackToTop from "./components/BackToTop.vue";
import DownloadProgress from "./components/DownloadProgress.vue";
import RuntimeInstallDialog from "./components/RuntimeInstallDialog.vue";
import DragDropHandler from "./components/DragDropHandler.vue";

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

// ==================== 背景特效 ====================
// 仅当设置了背景图片时生效：模糊半径 px / 遮罩浓度 % / 界面表面不透明度 %
const backgroundEffects = reactive({ blur: 0, dim: 40, surfaceOpacity: 85 });

function clampInt(value: unknown, min: number, max: number, fallback: number): number {
  const n = Number(value);
  if (!Number.isFinite(n)) return fallback;
  return Math.min(max, Math.max(min, Math.round(n)));
}

/** 把主题里的不透明色（#RRGGBB）转成指定透明度的 rgba */
function withAlpha(color: string, alpha: number): string {
  const m = color.match(/^#([0-9a-f]{6})$/i);
  if (!m) return color;
  const n = parseInt(m[1], 16);
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
}

// 提供给设置页的背景特效编辑（滑块拖动时即时预览，停止后再持久化）
provide('backgroundEffects', backgroundEffects);
// 设置页设置/清除背景图片后立即重载背景（无需离开设置页再回来）
provide('reloadBackground', () => loadBackgroundImage());
let backgroundEffectsSaveTimer: ReturnType<typeof setTimeout> | null = null;
provide('updateBackgroundEffects', (patch: Partial<{ blur: number; dim: number; surfaceOpacity: number }>) => {
  if (patch.blur !== undefined) backgroundEffects.blur = clampInt(patch.blur, 0, 40, 0);
  if (patch.dim !== undefined) backgroundEffects.dim = clampInt(patch.dim, 0, 100, 40);
  if (patch.surfaceOpacity !== undefined) backgroundEffects.surfaceOpacity = clampInt(patch.surfaceOpacity, 50, 100, 85);
  if (backgroundEffectsSaveTimer) clearTimeout(backgroundEffectsSaveTimer);
  backgroundEffectsSaveTimer = setTimeout(() => {
    SetBackgroundEffects(backgroundEffects.blur, backgroundEffects.dim, backgroundEffects.surfaceOpacity)
      .catch((e) => console.error('Failed to save background effects:', e));
  }, 400);
});

// Discord Design System 主题配置 - 浅色
const discordLightTheme: GlobalThemeOverrides = {
  common: {
    primaryColor: '#5865F2',
    primaryColorHover: '#4752C4',
    primaryColorPressed: '#3C45A5',
    primaryColorSuppl: '#5865F2',
    infoColor: '#5865F2',
    successColor: '#23A559',
    warningColor: '#F0B232',
    errorColor: '#DA373C',
    textColorBase: '#060607',
    textColor1: '#060607',
    textColor2: '#313338',
    textColor3: '#5D6167',
    dividerColor: 'rgba(0, 0, 0, 0.08)',
    borderColor: 'rgba(0, 0, 0, 0.08)',
    borderRadius: '4px',
    borderRadiusSmall: '2px',
    fontFamily: "'Noto Sans', Helvetica, Arial, sans-serif",
    fontSize: '16px',
    fontSizeMini: '12px',
    fontSizeTiny: '12px',
    fontSizeSmall: '14px',
    fontSizeMedium: '16px',
    fontSizeLarge: '16px',
    fontSizeHuge: '20px',
    heightMini: '28px',
    heightTiny: '32px',
    heightSmall: '36px',
    heightMedium: '40px',
    heightLarge: '44px',
    heightHuge: '48px',
    lineHeight: '1.25',
    hoverColor: 'rgba(0, 0, 0, 0.08)',
    cardColor: '#F2F3F5',
    modalColor: '#FFFFFF',
    bodyColor: '#FFFFFF',
    tagColor: 'rgba(0, 0, 0, 0.06)',
    avatarColor: '#F2F3F5',
    inputColor: '#E3E5E8',
    inputColorDisabled: '#F2F3F5',
    tableColor: '#F2F3F5',
    tableColorHover: '#E3E5E8',
    codeColor: '#E3E5E8',
    tabColor: '#FFFFFF',
    closeIconColor: '#5D6167',
    closeIconColorHover: '#060607',
    closeColorHover: 'rgba(0, 0, 0, 0.08)',
    closeColorPressed: 'rgba(0, 0, 0, 0.12)',
    clearColor: '#5D6167',
    clearColorHover: '#060607',
    clearColorPressed: '#313338',
    scrollbarColor: 'rgba(0, 0, 0, 0.2)',
    scrollbarColorHover: 'rgba(0, 0, 0, 0.35)',
    progressRailColor: 'rgba(0, 0, 0, 0.08)',
    railColor: 'rgba(0, 0, 0, 0.08)',
    popoverColor: '#FFFFFF',
    tableHeaderColor: '#F2F3F5',
    invertedColor: '#060607',
  },
  Button: {
    textColorPrimary: '#FFFFFF',
    textColorHoverPrimary: '#FFFFFF',
    textColorPressedPrimary: '#FFFFFF',
    textColorFocusPrimary: '#FFFFFF',
    colorPrimary: '#5865F2',
    colorHoverPrimary: '#4752C4',
    colorPressedPrimary: '#3C45A5',
    colorFocusPrimary: '#5865F2',
    borderPrimary: '#5865F2',
    borderHoverPrimary: '#4752C4',
    borderPressedPrimary: '#3C45A5',
    borderFocusPrimary: '#5865F2',
    borderRadiusMedium: '4px',
    borderRadiusLarge: '4px',
    fontWeight: '500',
  },
  Card: {
    borderRadius: '8px',
    borderColor: 'rgba(0, 0, 0, 0.08)',
    color: '#F2F3F5',
    colorModal: '#FFFFFF',
    colorPopover: '#FFFFFF',
    titleFontSizeMedium: '16px',
    titleFontWeight: '700',
    paddingMedium: '16px',
  },
  Input: {
    borderRadius: '4px',
    borderHover: '1px solid #5865F2',
    borderFocus: '1px solid #5865F2',
    boxShadowFocus: '0 0 0 1px #5865F2',
    color: '#E3E5E8',
    colorDisabled: '#F2F3F5',
    heightMedium: '40px',
    heightLarge: '44px',
  },
  Select: {
    peers: {
      InternalSelection: {
        borderRadius: '4px',
        borderHover: '1px solid #5865F2',
        borderFocus: '1px solid #5865F2',
        boxShadowFocus: '0 0 0 1px #5865F2',
        heightMedium: '40px',
      }
    }
  },
  Tag: {
    borderRadius: '4px',
    heightSmall: '24px',
    heightMedium: '28px',
    fontSizeSmall: '12px',
    fontSizeMedium: '14px',
    fontWeight: '500',
  },
  Tabs: {
    tabFontWeightActive: '700',
    tabFontWeight: '500',
    barColor: '#5865F2',
    tabTextColorActiveLine: '#060607',
    tabTextColorLine: '#5D6167',
    tabTextColorHoverLine: '#060607',
    tabFontSizeMedium: '16px',
    tabFontSizeLarge: '20px',
  },
  Dialog: {
    borderRadius: '8px',
    titleFontSize: '20px',
    fontSize: '16px',
    padding: '16px',
  },
  Notification: {
    borderRadius: '8px',
    padding: '16px',
  },
  Message: {
    borderRadius: '4px',
    padding: '10px 16px',
  },
  List: {
    borderRadius: '8px',
    color: '#F2F3F5',
    colorModal: '#FFFFFF',
  },
  DataTable: {
    borderRadius: '8px',
    thColor: '#F2F3F5',
  },
  Pagination: {
    itemBorderRadius: '4px',
    itemSize: '32px',
  },
  Switch: {
    railColor: 'rgba(0, 0, 0, 0.08)',
    railColorActive: '#5865F2',
  },
  Checkbox: {
    colorChecked: '#5865F2',
    borderChecked: '1px solid #5865F2',
  },
  Radio: {
    dotColorActive: '#5865F2',
    buttonBorderColorActive: '#5865F2',
    buttonColorActive: '#5865F2',
    buttonTextColorActive: '#FFFFFF',
  },
  Alert: {
    borderRadius: '8px',
  },
  Spin: {
    color: '#5865F2',
  },
  Empty: {
    textColor: '#5D6167',
  },
};

// Discord Design System 主题配置 - 深色
const discordDarkTheme: GlobalThemeOverrides = {
  common: {
    primaryColor: '#5865F2',
    primaryColorHover: '#4752C4',
    primaryColorPressed: '#3C45A5',
    primaryColorSuppl: '#5865F2',
    infoColor: '#5865F2',
    successColor: '#23A559',
    warningColor: '#F0B232',
    errorColor: '#DA373C',
    textColorBase: '#DBDEE1',
    textColor1: '#F2F3F5',
    textColor2: '#B5BAC1',
    textColor3: '#949BA4',
    dividerColor: 'rgba(255, 255, 255, 0.06)',
    borderColor: 'rgba(255, 255, 255, 0.06)',
    borderRadius: '4px',
    borderRadiusSmall: '2px',
    fontFamily: "'Noto Sans', Helvetica, Arial, sans-serif",
    fontSize: '16px',
    fontSizeMini: '12px',
    fontSizeTiny: '12px',
    fontSizeSmall: '14px',
    fontSizeMedium: '16px',
    fontSizeLarge: '16px',
    fontSizeHuge: '20px',
    heightMini: '28px',
    heightTiny: '32px',
    heightSmall: '36px',
    heightMedium: '40px',
    heightLarge: '44px',
    heightHuge: '48px',
    lineHeight: '1.25',
    hoverColor: 'rgba(255, 255, 255, 0.08)',
    cardColor: '#2B2D31',
    modalColor: '#313338',
    bodyColor: '#313338',
    tagColor: 'rgba(255, 255, 255, 0.06)',
    avatarColor: '#2B2D31',
    inputColor: '#1E1F22',
    inputColorDisabled: '#2B2D31',
    tableColor: '#2B2D31',
    tableColorHover: '#232428',
    codeColor: '#1E1F22',
    tabColor: '#313338',
    closeIconColor: '#949BA4',
    closeIconColorHover: '#DBDEE1',
    closeColorHover: 'rgba(255, 255, 255, 0.08)',
    closeColorPressed: 'rgba(255, 255, 255, 0.12)',
    clearColor: '#949BA4',
    clearColorHover: '#DBDEE1',
    clearColorPressed: '#F2F3F5',
    scrollbarColor: 'rgba(255, 255, 255, 0.1)',
    scrollbarColorHover: 'rgba(255, 255, 255, 0.15)',
    progressRailColor: 'rgba(255, 255, 255, 0.06)',
    railColor: 'rgba(255, 255, 255, 0.06)',
    popoverColor: '#111214',
    tableHeaderColor: '#2B2D31',
    invertedColor: '#F2F3F5',
  },
  Button: {
    textColorPrimary: '#FFFFFF',
    textColorHoverPrimary: '#FFFFFF',
    textColorPressedPrimary: '#FFFFFF',
    textColorFocusPrimary: '#FFFFFF',
    colorPrimary: '#5865F2',
    colorHoverPrimary: '#4752C4',
    colorPressedPrimary: '#3C45A5',
    colorFocusPrimary: '#5865F2',
    borderPrimary: '#5865F2',
    borderHoverPrimary: '#4752C4',
    borderPressedPrimary: '#3C45A5',
    borderFocusPrimary: '#5865F2',
    borderRadiusMedium: '4px',
    borderRadiusLarge: '4px',
    fontWeight: '500',
  },
  Card: {
    borderRadius: '8px',
    borderColor: 'rgba(255, 255, 255, 0.06)',
    color: '#2B2D31',
    colorModal: '#313338',
    colorPopover: '#111214',
    titleFontSizeMedium: '16px',
    titleFontWeight: '700',
    paddingMedium: '16px',
  },
  Input: {
    borderRadius: '4px',
    borderHover: '1px solid #5865F2',
    borderFocus: '1px solid #5865F2',
    boxShadowFocus: '0 0 0 1px #5865F2',
    color: '#1E1F22',
    colorDisabled: '#2B2D31',
    heightMedium: '40px',
    heightLarge: '44px',
  },
  Select: {
    peers: {
      InternalSelection: {
        borderRadius: '4px',
        borderHover: '1px solid #5865F2',
        borderFocus: '1px solid #5865F2',
        boxShadowFocus: '0 0 0 1px #5865F2',
        heightMedium: '40px',
      }
    }
  },
  Tag: {
    borderRadius: '4px',
    heightSmall: '24px',
    heightMedium: '28px',
    fontSizeSmall: '12px',
    fontSizeMedium: '14px',
    fontWeight: '500',
  },
  Tabs: {
    tabFontWeightActive: '700',
    tabFontWeight: '500',
    barColor: '#5865F2',
    tabTextColorActiveLine: '#F2F3F5',
    tabTextColorLine: '#949BA4',
    tabTextColorHoverLine: '#F2F3F5',
    tabFontSizeMedium: '16px',
    tabFontSizeLarge: '20px',
  },
  Dialog: {
    borderRadius: '8px',
    titleFontSize: '20px',
    fontSize: '16px',
    padding: '16px',
  },
  Notification: {
    borderRadius: '8px',
    padding: '16px',
  },
  Message: {
    borderRadius: '4px',
    padding: '10px 16px',
  },
  List: {
    borderRadius: '8px',
    color: '#2B2D31',
    colorModal: '#313338',
  },
  DataTable: {
    borderRadius: '8px',
    thColor: '#2B2D31',
  },
  Pagination: {
    itemBorderRadius: '4px',
    itemSize: '32px',
  },
  Switch: {
    railColor: 'rgba(255, 255, 255, 0.06)',
    railColorActive: '#5865F2',
  },
  Checkbox: {
    colorChecked: '#5865F2',
    borderChecked: '1px solid #5865F2',
  },
  Radio: {
    dotColorActive: '#5865F2',
    buttonBorderColorActive: '#5865F2',
    buttonColorActive: '#5865F2',
    buttonTextColorActive: '#FFFFFF',
  },
  Alert: {
    borderRadius: '8px',
  },
  Spin: {
    color: '#5865F2',
  },
  Empty: {
    textColor: '#949BA4',
  },
};

// 根据主题和背景图片动态设置主题覆盖：
// 设置背景图后卡片/列表/表格等表面按配置变半透明（浮动层保持更高不透明度保证可读性）
const themeOverrides = computed<GlobalThemeOverrides>(() => {
  const baseTheme = resolvedTheme.value === 'dark' ? discordDarkTheme : discordLightTheme;
  if (!hasBackgroundImage.value) return baseTheme;

  const alpha = backgroundEffects.surfaceOpacity / 100;
  const common = baseTheme.common ?? {};
  const surfaceColor = withAlpha(common.cardColor as string, alpha);

  // 只让页面内表面（卡片/列表/表格/分段标签）半透明；
  // 弹窗/下拉等浮动层保持完全不透明，避免背后的内容产生干扰
  return {
    ...baseTheme,
    common: {
      ...common,
      cardColor: surfaceColor,
      tableColor: surfaceColor,
      tableHeaderColor: surfaceColor,
      tabColor: withAlpha(common.tabColor as string, alpha),
    },
    Card: {
      ...baseTheme.Card,
      color: surfaceColor,
    },
    List: {
      ...baseTheme.List,
      color: surfaceColor,
    },
    DataTable: {
      ...baseTheme.DataTable,
      thColor: surfaceColor,
    },
  };
});

// Naive UI 主题 (darkTheme 或 undefined)
const naiveTheme = computed(() => resolvedTheme.value === 'dark' ? darkTheme : undefined);

// CSS 自定义属性注入（主题色 + 背景特效：
// 设置背景图后表面类变量按配置变半透明，透出背景图片）
watch(
  [resolvedTheme, hasBackgroundImage, () => backgroundEffects.surfaceOpacity, () => backgroundEffects.dim, () => backgroundEffects.blur],
  () => {
    const root = document.documentElement;
    const isDark = resolvedTheme.value === 'dark';

    if (isDark) {
      root.classList.remove('theme-light');
      root.classList.add('theme-dark');
    } else {
      root.classList.remove('theme-dark');
      root.classList.add('theme-light');
    }

    // 背景特效：html 标记（导航栏磨砂等场景）+ 背景层变量（模糊半径 / 遮罩浓度）
    root.classList.toggle('app-has-bg', hasBackgroundImage.value);
    root.style.setProperty('--app-bg-blur', `${backgroundEffects.blur}px`);
    root.style.setProperty('--app-bg-dim', String(backgroundEffects.dim / 100));

    const surfaceAlpha = hasBackgroundImage.value ? backgroundEffects.surfaceOpacity / 100 : 1;
    const surface = (color: string) => withAlpha(color, surfaceAlpha);

    const palette = isDark ? {
      bg: '#313338',
      surface: '#2B2D31',
      surfaceElevated: '#1E1F22',
      textPrimary: '#F2F3F5',
      textSecondary: '#DBDEE1',
      textTertiary: '#949BA4',
      border: 'rgba(255, 255, 255, 0.06)',
      primary: '#5865F2',
      primaryHover: '#4752C4',
      primaryHoverBg: 'rgba(88, 101, 242, 0.1)',
      error: '#DA373C',
      errorHover: '#A12D31',
      scrollThumb: 'rgba(255, 255, 255, 0.1)',
      scrollThumbHover: 'rgba(255, 255, 255, 0.15)',
      navBg: '#2B2D31',
      hover: 'rgba(255, 255, 255, 0.08)',
    } : {
      bg: '#FFFFFF',
      surface: '#F2F3F5',
      surfaceElevated: '#E3E5E8',
      textPrimary: '#060607',
      textSecondary: '#313338',
      textTertiary: '#5D6167',
      border: 'rgba(0, 0, 0, 0.08)',
      primary: '#5865F2',
      primaryHover: '#4752C4',
      primaryHoverBg: 'rgba(88, 101, 242, 0.1)',
      error: '#DA373C',
      errorHover: '#A12D31',
      scrollThumb: 'rgba(0, 0, 0, 0.2)',
      scrollThumbHover: 'rgba(0, 0, 0, 0.35)',
      navBg: '#FFFFFF',
      hover: 'rgba(0, 0, 0, 0.08)',
    };

    const vars: Record<string, string> = {
      '--color-bg': palette.bg,
      '--color-surface': surface(palette.surface),
      '--color-surface-elevated': surface(palette.surfaceElevated),
      '--color-text-primary': palette.textPrimary,
      '--color-text-secondary': palette.textSecondary,
      '--color-text-tertiary': palette.textTertiary,
      '--color-border': palette.border,
      '--color-primary': palette.primary,
      '--color-primary-hover': palette.primaryHover,
      '--color-primary-hover-bg': palette.primaryHoverBg,
      '--color-error': palette.error,
      '--color-error-hover': palette.errorHover,
      '--color-scroll-thumb': palette.scrollThumb,
      '--color-scroll-thumb-hover': palette.scrollThumbHover,
      '--color-nav-bg': surface(palette.navBg),
      '--hover-color': palette.hover,
      // 内容区浅色底：页面内容较少（如版本页大多折叠）时，底部不露出未处理的背景
      '--app-content-wash': hasBackgroundImage.value
        ? withAlpha(palette.surface, surfaceAlpha * 0.55)
        : 'transparent',
    };

    for (const [key, value] of Object.entries(vars)) {
      root.style.setProperty(key, value);
    }
  },
  { immediate: true }
);

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
    // 加载背景图片与背景特效
    backgroundEffects.blur = clampInt(config?.backgroundBlur, 0, 40, 0);
    backgroundEffects.dim = clampInt(config?.backgroundDim, 0, 100, 40);
    backgroundEffects.surfaceOpacity = clampInt(config?.backgroundSurfaceOpacity, 50, 100, 85);
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

  if (dialogProviderInst.value) {
    const dialog = dialogProviderInst.value;
    // @ts-ignore
    dialog.create({
      title: t('home.gameCrash') || "游戏崩溃",
      content: () => {
        // 自动滚动到底部
        setTimeout(() => {
          const preEl = document.querySelector('.crash-log-content')
          if (preEl) preEl.scrollTop = preEl.scrollHeight
        }, 50)

        return h("div", { style: "text-align: left;" }, [
          h("p", { style: "margin-bottom: 12px; font-weight: bold;" },
            `${t('versions.version')}: ${versionName}`),
          h("p", { style: "margin-bottom: 12px;" },
            `${t('installed.exitCode')}: ${exitCode}`),
          h("p", { style: "margin-bottom: 12px;" },
            `${t('installed.crashTime')}: ${crashTime}`),
          h("div", { style: "display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;" }, [
            h("span", { style: "font-weight: 600; font-size: 14px;" },
              t('home.gameLog') || "运行日志"),
            h(NButton, {
              size: "small",
              onClick: () => {
                navigator.clipboard.writeText(log).then(() => {
                  // 复制成功提示
                })
              }
            }, { default: () => t('common.copy') || "复制" }),
          ]),
          h("pre", {
            class: "crash-log-content",
            style: "max-height: 300px; overflow-y: auto; background: #DA373C; color: #fff; padding: 12px; border-radius: 8px; font-size: 12px; white-space: pre-wrap; word-wrap: break-word; font-family: 'SF Mono', SFMono-Regular, Menlo, Monaco, Consolas, monospace; text-align: left;",
          }, log),
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
  /* 滚动只发生在 .app-container 内；裁掉背景层的外扩部分，避免出现文档级滚动条 */
  overflow: hidden;
  --wails-drop-target: drop;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Noto Sans', Helvetica, Arial, sans-serif;
  background-color: var(--color-bg, #313338);
  color: var(--color-text-primary, #F2F3F5);
  transition: background-color 0.17s ease, color 0.17s ease;
}

.app-container {
  width: 100vw;
  height: 100vh;
  overflow-y: auto;
}

/* 设置背景图后：内容区整体铺一层浅色底，短页面底部不再露出未处理的背景 */
.app-has-bg .app-container {
  background-color: var(--app-content-wash, transparent);
}

/* Discord 风格顶部导航栏 */
.app-nav {
  position: sticky;
  top: 0;
  width: 100%;
  padding: 8px 16px;
  background-color: var(--color-nav-bg, #2B2D31);
  border-bottom: 1px solid var(--color-border, rgba(255, 255, 255, 0.06));
  z-index: 1000;
}

/* 滚动条样式 - Discord 风格 */
.app-container::-webkit-scrollbar {
  width: 8px;
}

.app-container::-webkit-scrollbar-track {
  background: transparent;
}

.app-container::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}

.app-container::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.15);
}

/* 背景图片层（模糊作用在图片本体上，由 --app-bg-blur 控制）。
   模糊时用负 inset 向四周外扩遮住边缘晕开——不用 transform 放大，
   否则变换后的固定定位元素会被算进文档滚动区域，右侧多出滚动条 */
.background-layer {
  position: fixed;
  top: calc(var(--app-bg-blur, 0px) * -1.25);
  right: calc(var(--app-bg-blur, 0px) * -1.25);
  bottom: calc(var(--app-bg-blur, 0px) * -1.25);
  left: calc(var(--app-bg-blur, 0px) * -1.25);
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  z-index: 0;
  pointer-events: none;
  filter: blur(var(--app-bg-blur, 0px));
}

/* 背景遮罩层：浓度由 --app-bg-dim（0-1）控制 */
.background-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, var(--app-bg-dim, 0.4));
}

/* 设置背景图后：顶部导航磨砂玻璃（配合半透明 --color-nav-bg） */
.app-has-bg .app-nav {
  backdrop-filter: blur(14px) saturate(1.1);
  -webkit-backdrop-filter: blur(14px) saturate(1.1);
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

/* Discord 风格按钮过渡 */
.n-button {
  transition: background-color 0.17s ease, color 0.17s ease !important;
}

/* Discord 风格卡片 */
.n-card {
  border: 1px solid var(--color-border, rgba(255, 255, 255, 0.06));
  box-shadow: none;
  transition: border-color 0.17s ease, background-color 0.17s ease;
}

/* Discord 风格列表项 */
.n-list-item {
  transition: background-color 0.17s ease;
}

.n-list-item:hover {
  background-color: rgba(255, 255, 255, 0.08) !important;
}

/* Discord 风格标签 */
.n-tag {
  font-weight: 500;
}
</style>
