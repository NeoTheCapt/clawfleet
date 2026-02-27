<template>
  <div>
    <div class="flex justify-between mb-4">
      <h3 class="font-semibold">Workflows 工作流</h3>
      <button @click="openModal()" class="px-2.5 py-1 bg-btn-green border border-btn-greenHover rounded-md text-xs text-text hover:bg-btn-greenHover transition">+ New Workflow</button>
    </div>
    <EmptyState v-if="items.length === 0" emoji="⚙️" message="No workflows yet" />
    <DataTable v-else :columns="['Name','Position','Trigger','Status','Actions']">
      <tr v-for="w in items" :key="w.id" class="hover:bg-bg-hover">
        <td class="px-4 py-3 border-t border-border-light"><strong>{{ w.name }}</strong><div v-if="w.description" class="text-xs text-text-muted">{{ w.description.substring(0,60) }}</div></td>
        <td class="px-4 py-3 border-t border-border-light">{{ w.position_title || w.position_id || '-' }}</td>
        <td class="px-4 py-3 border-t border-border-light text-xs text-text-muted">{{ w.trigger || w.trigger_type || '-' }}</td>
        <td class="px-4 py-3 border-t border-border-light"><span :class="'badge badge-'+(w.status||'active')">{{ w.status || 'active' }}</span></td>
        <td class="px-4 py-3 border-t border-border-light">
          <button @click="openModal(w)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition mr-1">✏️</button>
          <button @click="del(w.id)" class="px-2 py-0.5 bg-btn-red border border-accent-red rounded text-xs hover:bg-btn-redHover transition">🗑️</button>
        </td>
      </tr>
    </DataTable>

    <Modal :show="showModal" @close="showModal = false">
      <h3 class="text-lg font-semibold mb-4">{{ form.id ? '✏️ Edit' : '➕ New' }} Workflow</h3>
      <FieldLabel>Name 名称</FieldLabel>
      <input v-model="form.name" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Position 岗位</FieldLabel>
      <select v-model="form.position_id" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3">
        <option value="">-- Any --</option>
        <option v-for="p in positions" :key="p.id" :value="p.id">{{ p.title || p.name }}</option>
      </select>
      <FieldLabel>Trigger 触发条件</FieldLabel>
      <input v-model="form.trigger" placeholder="e.g. on_message, cron:0 9 * * *" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Description</FieldLabel>
      <textarea v-model="form.description" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Steps (JSON)</FieldLabel>
      <textarea v-model="form.steps_str" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[120px] resize-y font-mono text-xs focus:outline-none focus:border-accent-blue"></textarea>
      <ModalActions :onCancel="() => (showModal = false)" :onSave="save" />
    </Modal>
  </div>
</template>

<script setup>
import { ref, onMounted, inject } from 'vue'
import { api, apiPost, apiPut, confirmAndDelete } from '../../api'
import { useLoader } from '../../composables/useLoader'
import DataTable from '../../components/DataTable.vue'
import EmptyState from '../../components/EmptyState.vue'
import Modal from '../../components/Modal.vue'
import ModalActions from '../../components/ModalActions.vue'
import FieldLabel from '../../components/FieldLabel.vue'

const props = defineProps({ companyId: String })
const toast = inject('toast')
const cid = () => props.companyId
const items = ref([])
const positions = ref([])
const showModal = ref(false)
const form = ref({})

const { load } = useLoader(async () => {
  items.value = await api(`/api/companies/${cid()}/workflows`).catch(() => []) || []
  positions.value = await api(`/api/companies/${cid()}/positions`).catch(() => []) || []
})

onMounted(load)

function openModal(w) {
  form.value = w ? { ...w, steps_str: w.steps ? JSON.stringify(w.steps, null, 2) : '' }
    : { name: '', position_id: '', trigger: '', description: '', steps_str: '[\n  {"action": "example", "params": {}}\n]' }
  showModal.value = true
}

async function save() {
  let steps = null
  if (form.value.steps_str?.trim()) { try { steps = JSON.parse(form.value.steps_str) } catch { alert('Invalid steps JSON'); return } }
  const b = { name: form.value.name, position_id: form.value.position_id || null, trigger: form.value.trigger, description: form.value.description, steps }
  if (!b.name) { alert('Name required'); return }
  if (form.value.id) await apiPut(`/api/companies/${cid()}/workflows/${form.value.id}`, b)
  else await apiPost(`/api/companies/${cid()}/workflows`, b)
  showModal.value = false; toast('Saved'); load()
}

async function del(id) { await confirmAndDelete('Delete this workflow?', `/api/companies/${cid()}/workflows/${id}`, toast, 'Deleted', '#da3633', load) }
</script>
