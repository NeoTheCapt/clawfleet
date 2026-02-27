<template>
  <div class="w-[220px] bg-bg-card border-r border-border flex flex-col">
    <div class="p-5 text-lg font-bold border-b border-border">
      <span class="mr-2">🚢</span> Clawfleet
    </div>
    <div class="flex-1 py-3">
      <router-link v-for="item in navItems" :key="item.to" :to="item.to"
        class="flex items-center px-5 py-2.5 text-sm text-text-muted transition-all hover:bg-bg-hover hover:text-text"
        :class="isActive(item.to) && 'bg-bg-hover !text-accent-blue border-r-2 border-accent-blue'">
        <span class="mr-2.5 text-base">{{ item.icon }}</span> {{ item.label }}
      </router-link>
    </div>
    <div class="p-3 border-t border-border">
      <button @click="doLogout" class="flex items-center px-5 py-2.5 text-sm text-accent-red w-full hover:bg-bg-hover transition-all">
        <span class="mr-2.5 text-base">🚪</span> Logout
      </button>
    </div>
  </div>
</template>

<script setup>
import { useRoute, useRouter } from 'vue-router'
import { clearToken } from '../composables/useAuth'

const route = useRoute()
const router = useRouter()

const navItems = [
  { to: '/', icon: '📊', label: 'Dashboard' },
  { to: '/nodes', icon: '🖥️', label: 'Nodes' },
  { to: '/agents', icon: '🤖', label: 'Agents' },
  { to: '/im', icon: '💬', label: 'Web IM' },
  { to: '/companies', icon: '🏢', label: 'Company' },
  { to: '/settings', icon: '⚙️', label: 'Settings' }
]

function isActive(to) {
  if (to === '/') return route.path === '/'
  return route.path.startsWith(to)
}

function doLogout() {
  clearToken()
  router.push('/login')
}
</script>
