<template>
  <div>
    <div class="flex justify-between mb-4">
      <h3 class="font-semibold">IM Groups 即时通讯群组</h3>
      <button @click="openModal()" class="px-2.5 py-1 bg-btn-green border border-btn-greenHover rounded-md text-xs text-text hover:bg-btn-greenHover transition">+ New Group</button>
    </div>
    <EmptyState v-if="groups.length === 0" emoji="💬" message="No IM groups yet" />
    <DataTable v-else :columns="['Name','Platform','Members','Status','Actions']">
      <tr v-for="g in groups" :key="g.id" class="hover:bg-bg-hover">
        <td class="px-4 py-3 border-t border-border-light font-semibold">{{ g.name }}</td>
        <td class="px-4 py-3 border-t border-border-light">{{ g.platform || g.type || '-' }}</td>
        <td class="px-4 py-3 border-t border-border-light text-xs text-text-muted">{{ Array.isArray(g.members) ? g.members.length : (g.member_count || '-') }}</td>
        <td class="px-4 py-3 border-t border-border-light"><span :class="'badge badge-'+(g.status||'active')">{{ g.status || 'active' }}</span></td>
        <td class="px-4 py-3 border-t border-border-light">
          <button @click="openModal(g)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition mr-1">✏️</button>
          <button @click="del(g.id)" class="px-2 py-0.5 bg-btn-red border border-accent-red rounded text-xs hover:bg-btn-redHover transition">🗑️</button>
        </td>
      </tr>
    </DataTable>

    <Modal :show="showModal" @close="showModal = false">
      <h3 class="text-lg font-semibold mb-4">{{ form.id ? '✏️ Edit' : '➕ New' }} IM Group</h3>
      <FieldLabel>Name 名称</FieldLabel>
      <input v-model="form.name" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Platform 平台</FieldLabel>
      <select v-model="form.platform" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3">
        <option v-for="p in ['telegram','discord','slack','wechat','custom']" :key="p" :value="p">{{ p }}</option>
      </select>
      <FieldLabel>Description</FieldLabel>
      <textarea v-model="form.description" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Members (comma-separated agent IDs)</FieldLabel>
      <input v-model="form.members_str" placeholder="agent-id-1, agent-id-2" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Config (JSON)</FieldLabel>
      <textarea v-model="form.config_str" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y font-mono text-xs focus:outline-none focus:border-accent-blue"></textarea>
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
const groups = ref([])
const showModal = ref(false)
const form = ref({})

const { load } = useLoader(async () => {
  groups.value = await api(`/api/companies/${cid()}/im-groups`).catch(() => []) || []
})

onMounted(load)

function openModal(g) {
  form.value = g ? { ...g, members_str: Array.isArray(g.members) ? g.members.join(', ') : '', config_str: g.config ? JSON.stringify(g.config, null, 2) : '' }
    : { name: '', platform: 'telegram', description: '', members_str: '', config_str: '' }
  showModal.value = true
}

async function save() {
  let config = null
  if (form.value.config_str?.trim()) { try { config = JSON.parse(form.value.config_str) } catch { alert('Invalid config JSON'); return } }
  const b = { name: form.value.name, platform: form.value.platform, description: form.value.description, members: form.value.members_str.split(',').map(s => s.trim()).filter(Boolean), config }
  if (!b.name) { alert('Name required'); return }
  if (form.value.id) await apiPut(`/api/companies/${cid()}/im-groups/${form.value.id}`, b)
  else await apiPost(`/api/companies/${cid()}/im-groups`, b)
  showModal.value = false; toast('Saved'); load()
}

async function del(id) { await confirmAndDelete('Delete this IM group?', `/api/companies/${cid()}/im-groups/${id}`, toast, 'Deleted', '#da3633', load) }
</script>
