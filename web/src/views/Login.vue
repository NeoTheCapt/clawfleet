<template>
  <div class="fixed inset-0 bg-bg flex justify-center items-center">
    <div class="bg-bg-card border border-border rounded-xl p-6 w-[500px] text-center">
      <div class="text-5xl mb-3">🚢</div>
      <h3 class="text-lg font-semibold mb-1">Clawfleet</h3>
      <p class="text-text-muted text-sm mb-5">Sign in to continue</p>
      <div v-if="error" class="text-accent-red text-sm mb-2">{{ error }}</div>
      <label class="block text-left text-sm text-text-muted mb-1">Username</label>
      <input v-model="username" type="text" placeholder="admin" autofocus
        class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <label class="block text-left text-sm text-text-muted mb-1">Password</label>
      <input v-model="password" type="password" placeholder="••••••••" @keydown.enter="doLogin"
        class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-4 focus:outline-none focus:border-accent-blue">
      <button @click="doLogin" class="w-full px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">Sign In</button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login as apiLogin } from '../api'
import { setToken } from '../composables/useAuth'

const router = useRouter()
const username = ref('')
const password = ref('')
const error = ref('')

async function doLogin() {
  error.value = ''
  if (!username.value || !password.value) { error.value = 'Please fill in both fields'; return }
  try {
    const data = await apiLogin(username.value, password.value)
    setToken(data.token)
    router.push('/')
  } catch (e) { error.value = e.message || 'Connection error' }
}
</script>
