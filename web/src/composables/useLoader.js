import { ref } from 'vue'

// useLoader wraps an async fetcher with loading + error state.
export function useLoader(fetcher) {
  const loading = ref(false)
  const error = ref('')

  async function load() {
    loading.value = true
    error.value = ''
    try {
      return await fetcher()
    } catch (e) {
      error.value = e?.message || String(e)
      return null
    } finally {
      loading.value = false
    }
  }

  return { loading, error, load }
}
