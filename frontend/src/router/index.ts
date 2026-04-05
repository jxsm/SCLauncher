import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Home',
    component: () => import('../views/Home.vue')
  },
  {
    path: '/installed',
    name: 'Installed',
    component: () => import('../views/InstalledVersions.vue')
  },
  {
    path: '/versions',
    name: 'Versions',
    component: () => import('../views/Versions.vue')
  },
  {
    path: '/mods',
    name: 'Mods',
    component: () => import('../views/Mods.vue')
  },
  {
    path: '/savegames',
    name: 'SaveGames',
    component: () => import('../views/SaveGames.vue')
  },
  {
    path: '/skins',
    name: 'Skins',
    component: () => import('../views/Skins.vue')
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('../views/Settings.vue')
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
