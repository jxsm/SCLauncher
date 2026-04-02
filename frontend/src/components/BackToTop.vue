<template>
  <transition name="fade">
    <n-button
      v-if="visible"
      class="back-to-top"
      type="primary"
      circle
      size="large"
      @click="scrollToTop"
    >
      <template #icon>
        <n-icon><ArrowUpIcon /></n-icon>
      </template>
    </n-button>
  </transition>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ArrowUp as ArrowUpIcon } from '@vicons/ionicons5'

const visible = ref(false)
const SCROLL_THRESHOLD = 300 // 滚动超过300px时显示按钮
let scrollContainer: HTMLElement | null = null

function handleScroll() {
  if (!scrollContainer) return
  const scrollTop = scrollContainer.scrollTop
  visible.value = scrollTop > SCROLL_THRESHOLD
}

function scrollToTop() {
  if (!scrollContainer) return
  scrollContainer.scrollTo({
    top: 0,
    behavior: 'smooth'
  })
}

onMounted(() => {
  // 获取 .app-container 元素
  scrollContainer = document.querySelector('.app-container')
  if (scrollContainer) {
    scrollContainer.addEventListener('scroll', handleScroll)
  }
})

onUnmounted(() => {
  if (scrollContainer) {
    scrollContainer.removeEventListener('scroll', handleScroll)
  }
})
</script>

<style scoped>
.back-to-top {
  position: fixed;
  bottom: 40px;
  right: 40px;
  z-index: 1000;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

/* 淡入淡出动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
