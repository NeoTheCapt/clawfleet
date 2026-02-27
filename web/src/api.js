import { getToken, clearToken } from './composables/useAuth'

async function request(path, opts = {}) {
  const token = getToken()
  if (token) {
    opts.headers = opts.headers || {}
    opts.headers['Authorization'] = 'Bearer ' + token
  }
  const r = await fetch(path, opts)
  if (r.status === 401) { clearToken(); throw new Error('unauthorized') }
  if (r.status === 204) return null
  const ct = r.headers.get('content-type') || ''
  if (!ct.includes('application/json')) {
    const text = await r.text()
    throw new Error(r.ok ? 'Unexpected non-JSON response' : `HTTP ${r.status}: ${text.substring(0, 100)}`)
  }
  const data = await r.json()
  if (!r.ok) throw new Error(data.error || `HTTP ${r.status}`)
  return data
}

export async function api(path) {
  return request(path)
}

export async function apiPost(path, body) {
  return request(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
}

export async function apiPut(path, body) {
  return request(path, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
}

export async function apiDelete(path) {
  return request(path, { method: 'DELETE' })
}

export async function confirmAndDelete(message, url, toast, toastMsg = 'Deleted', color = '#da3633', onDone) {
  if (!confirm(message)) return
  await apiDelete(url)
  if (toast) toast(toastMsg, color)
  if (onDone) onDone()
}

export async function login(username, password) {
  return apiPost('/api/login', { username, password })
}
