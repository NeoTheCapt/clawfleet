<template>
  <div>
    <div class="flex justify-end mb-4">
      <button @click="showCreate = true" class="px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">+ New Company</button>
    </div>

    <EmptyState v-if="companies.length === 0" emoji="🏢" message="No companies yet">
      <button @click="showCreate = true" class="mt-3 px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">Create Company</button>
    </EmptyState>

    <div v-else class="grid gap-4">
      <router-link v-for="c in companies" :key="c.id" :to="`/company/${c.id}`"
        class="bg-bg-card border border-border rounded-lg p-5 hover:bg-bg-hover transition block">
        <h3 class="text-base font-semibold">{{ c.name }}</h3>
        <p v-if="c.description" class="text-sm text-text-muted mt-1">{{ c.description }}</p>
      </router-link>
    </div>

    <Modal :show="showCreate" @close="showCreate = false">
      <h3 class="text-lg font-semibold mb-4">🏢 Create Company</h3>
      <FieldLabel>Name 名称</FieldLabel>
      <input v-model="form.name" type="text" placeholder="My Company" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Vision 愿景</FieldLabel>
      <textarea v-model="form.vision" placeholder="Company vision..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Mission 使命</FieldLabel>
      <textarea v-model="form.mission" placeholder="Company mission..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Description 描述</FieldLabel>
      <textarea v-model="form.description" placeholder="Brief description..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <div class="flex justify-end gap-2 mt-5">
        <button @click="showCreate = false" class="px-4 py-2 border border-border rounded-md bg-btn text-text text-sm hover:bg-btn-hover transition">Cancel</button>
        <button @click="create" class="px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">Create</button>
      </div>
    </Modal>
  </div>
</template>

<script setup>
import { ref, onMounted, inject } from 'vue'
import { useRouter } from 'vue-router'
import { api, apiPost } from '../api'
import { useLoader } from '../composables/useLoader'
import FieldLabel from '../components/FieldLabel.vue'
import Modal from '../components/Modal.vue'
import EmptyState from '../components/EmptyState.vue'

const toast = inject('toast')
const router = useRouter()
const companies = ref([])
const showCreate = ref(false)
const form = ref({ name: '', vision: '', mission: '', description: '' })

const { load } = useLoader(async () => {
  companies.value = await api('/api/companies') || []
})

onMounted(load)

async function create() {
  if (!form.value.name.trim()) { alert('Name required'); return }
  const result = await apiPost('/api/companies', form.value)
  showCreate.value = false
  toast('Company created')
  if (result?.id) router.push(`/company/${result.id}`)
  else { companies.value = await api('/api/companies') || [] }
}
</script>
