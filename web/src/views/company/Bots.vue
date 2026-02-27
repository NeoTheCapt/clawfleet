<template>
  <div>
    <div class="flex justify-between mb-4">
      <h3 class="font-semibold">🤖 Telegram Bots</h3>
      <button @click="openModal()" class="px-2.5 py-1 bg-btn-green border border-btn-greenHover rounded-md text-xs text-text hover:bg-btn-greenHover transition">+ Add Telegram Bot</button>
    </div>

    <EmptyState v-if="bots.length === 0" emoji="🤖" message="No Telegram bots yet" />
    <DataTable v-else :columns="['Name','Telegram Bot ID','Actions']">
      <tr v-for="b in bots" :key="b.id" class="hover:bg-bg-hover">
        <td class="px-4 py-3 border-t border-border-light font-semibold">{{ b.name }}</td>
        <td class="px-4 py-3 border-t border-border-light font-mono text-xs">{{ b.bot_id }}</td>
        <td class="px-4 py-3 border-t border-border-light">
          <button @click="openModal(b)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition mr-1">✏️</button>
          <button @click="delBot(b.id)" class="px-2 py-0.5 bg-btn-red border border-accent-red rounded text-xs hover:bg-btn-redHover transition">🗑️</button>
        </td>
      </tr>
    </DataTable>

    <Modal :show="showModal" @close="showModal = false">
      <h3 class="text-lg font-semibold mb-4">{{ form.id ? '✏️ Edit' : '➕ New' }} Telegram Bot</h3>
      <FieldLabel>Name 名称</FieldLabel>
      <input v-model="form.name" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Telegram Bot ID</FieldLabel>
      <input v-model="form.bot_id" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 font-mono focus:outline-none focus:border-accent-blue">
      <FieldLabel>Telegram Bot Token</FieldLabel>
      <input v-model="form.token" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 font-mono focus:outline-none focus:border-accent-blue">
      <ModalActions :onCancel="() => (showModal = false)" :onSave="save" />
    </Modal>
  </div>
</template>

<script setup>
import { ref, onMounted, inject } from 'vue'
import { api, apiPost, apiPut, confirmAndDelete } from '../../api'
import { useLoader } from '../../composables/useLoader'
import EmptyState from '../../components/EmptyState.vue'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import ModalActions from '../../components/ModalActions.vue'
import FieldLabel from '../../components/FieldLabel.vue'

const props = defineProps({ companyId: String })
const toast = inject('toast')
const bots = ref([])
const showModal = ref(false)
const form = ref({ id: '', name: '', bot_id: '', token: '' })

const { load } = useLoader(async () => {
  bots.value = await api(`/api/companies/${props.companyId}/bots`).catch(() => []) || []
})

function openModal(b) {
  form.value = b ? { ...b } : { id: '', name: '', bot_id: '', token: '' }
  showModal.value = true
}

async function save() {
  const payload = { name: form.value.name, bot_id: form.value.bot_id, token: form.value.token }
  if (form.value.id) await apiPut(`/api/companies/${props.companyId}/bots/${form.value.id}`, payload)
  else await apiPost(`/api/companies/${props.companyId}/bots`, payload)
  showModal.value = false
  toast('Saved')
  load()
}

async function delBot(id) {
  await confirmAndDelete('Delete this Telegram bot?', `/api/companies/${props.companyId}/bots/${id}`, toast, 'Deleted', '#da3633', load)
}

onMounted(load)
</script>
