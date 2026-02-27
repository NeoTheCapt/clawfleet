<template>
  <div v-if="isAuthenticated()" class="flex h-screen">
    <Sidebar />
    <div class="flex-1 flex flex-col overflow-hidden">
      <div class="px-6 py-4 border-b border-border flex justify-between items-center">
        <h2 class="text-lg font-semibold">{{ pageTitle }}</h2>
        <div id="header-actions"><slot name="actions" /></div>
      </div>
      <div class="flex-1 overflow-y-auto p-6">
        <router-view />
      </div>
    </div>
  </div>
  <router-view v-else />
  <!-- Toast -->
  <div v-if="toastMsg" class="fixed bottom-6 right-6 text-white px-5 py-2.5 rounded-lg text-sm z-[200] toast-enter" :style="{ background: toastColor }">{{ toastMsg }}</div>
</template>

<script setup>
import { computed, provide, ref } from 'vue'
import { useRoute } from 'vue-router'
import { isAuthenticated } from './composables/useAuth'
import Sidebar from './components/Sidebar.vue'

const route = useRoute()
const toastMsg = ref('')
const toastColor = ref('#238636')

const pageTitle = computed(() => {
  const titles = { Dashboard: 'Dashboard', Nodes: 'Nodes', Agents: 'Agents', Companies: 'Company Management', CompanyDetail: 'Company Management', Settings: 'Settings', IM: 'Web IM' }
  return titles[route.name] || route.name || ''
})

function toast(msg, color = '#238636') {
  toastMsg.value = msg
  toastColor.value = color
  setTimeout(() => { toastMsg.value = '' }, 3000)
}

provide('toast', toast)
</script>
