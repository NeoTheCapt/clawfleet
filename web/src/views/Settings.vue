<template>
  <div class="max-w-xl">
    <h3 class="font-semibold mb-4">ℹ️ System Info</h3>
    <div class="bg-bg-card border border-border rounded-lg p-5 mb-6">
      <div class="grid grid-cols-2 gap-2 text-sm">
        <div class="text-text-muted">Frontend:</div>
        <div class="font-mono">{{ versions.web || 'loading...' }}</div>
        <div class="text-text-muted">Server:</div>
        <div class="font-mono">{{ versions.server || 'loading...' }}</div>
        <div class="text-text-muted">Node Agents:</div>
        <div class="font-mono">{{ versions.nodes || 'loading...' }}</div>
      </div>
      <button @click="loadVersions" class="mt-3 text-xs text-accent-blue hover:underline">🔄 Refresh</button>
    </div>

    <h3 class="font-semibold mb-4">🔐 Admin Credentials</h3>

    <div class="bg-bg-card border border-border rounded-lg p-5">
      <p class="text-sm text-text-muted mb-4">
        Update the dashboard login username/password. After saving, you will be logged out.
      </p>

      <FieldLabel>Current Password 当前密码</FieldLabel>
      <input v-model="form.current_password" type="password" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 font-mono focus:outline-none focus:border-accent-blue">

      <FieldLabel>New Username 新用户名</FieldLabel>
      <input v-model="form.new_username" type="text" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 font-mono focus:outline-none focus:border-accent-blue">

      <FieldLabel>New Password 新密码</FieldLabel>
      <input v-model="form.new_password" type="password" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 font-mono focus:outline-none focus:border-accent-blue">

      <FieldLabel>Confirm New Password 确认新密码</FieldLabel>
      <input v-model="form.confirm_password" type="password" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-4 font-mono focus:outline-none focus:border-accent-blue">

      <label class="flex items-center gap-2 text-sm text-text-muted mb-4">
        <input type="checkbox" v-model="form.rotate_jwt" class="rounded border-border accent-accent-blue">
        Invalidate existing sessions (recommended)
      </label>

      <div class="flex justify-end gap-2">
        <button @click="save" class="px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">Save</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, inject, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api, apiPut } from '../api'
import { clearToken } from '../composables/useAuth'
import FieldLabel from '../components/FieldLabel.vue'
import { WEB_VERSION, WEB_COMMIT, WEB_BUILD_TIME } from '../version'

const toast = inject('toast')
const router = useRouter()

const versions = ref({
  web: '',
  server: '',
  nodes: ''
})

onMounted(() => {
  loadVersions()
})

async function loadVersions() {
  // Frontend version
  versions.value.web = `${WEB_VERSION} (${(WEB_COMMIT || 'unknown').slice(0,8)})`

  // Server version
  try {
    const serverInfo = await api('/api/version')
    versions.value.server = `${serverInfo.version || 'unknown'} (${(serverInfo.commit || '').slice(0,8)})`
  } catch (e) {
    versions.value.server = 'error'
  }

  // Node versions (from nodes list)
  try {
    const nodes = await api('/api/nodes')
    if (nodes && nodes.length > 0) {
      const nodeVers = nodes.map(n => {
        const v = n.resources?.agent_version || 'unknown'
        const c = (n.resources?.agent_commit || '').slice(0,8)
        return `${n.name}: ${v}${c ? ' ('+c+')' : ''}`
      }).join(', ')
      versions.value.nodes = nodeVers || 'no agents'
    } else {
      versions.value.nodes = 'no nodes'
    }
  } catch (e) {
    versions.value.nodes = 'error'
  }
}

const form = ref({
  current_password: '',
  new_username: 'admin',
  new_password: '',
  confirm_password: '',
  rotate_jwt: true
})

async function save() {
  if (!form.value.current_password) { alert('Current password required'); return }
  if (!form.value.new_username) { alert('New username required'); return }
  if (!form.value.new_password) { alert('New password required'); return }
  if (form.value.new_password !== form.value.confirm_password) { alert('Passwords do not match'); return }

  await apiPut('/api/admin/credentials', {
    current_password: form.value.current_password,
    new_username: form.value.new_username,
    new_password: form.value.new_password,
    rotate_jwt: form.value.rotate_jwt
  })

  toast('Credentials updated. Please login again.', '#d29922')
  clearToken()
  router.push('/login')
}
</script>
