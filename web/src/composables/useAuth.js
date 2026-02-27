import { ref } from 'vue'

const TOKEN_KEY = 'clawfleet_token'
const token = ref(localStorage.getItem(TOKEN_KEY) || '')

export function getToken() { return token.value }

export function setToken(t) {
  token.value = t
  localStorage.setItem(TOKEN_KEY, t)
}

export function clearToken() {
  token.value = ''
  localStorage.removeItem(TOKEN_KEY)
}

export function isAuthenticated() { return !!token.value }

export function useAuth() {
  return { token, getToken, setToken, clearToken, isAuthenticated }
}
