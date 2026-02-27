<template>
  <div>
    <DataTable :columns="['Name','Type','Status','Model','Actions']">
      <tr v-for="a in agents" :key="a.id" class="hover:bg-bg-hover">
        <td class="px-4 py-3 border-t border-border-light font-semibold">
          {{ a.name }}
          <div v-if="a.role" class="text-xs text-text-muted">{{ a.role }}</div>
        </td>
        <td class="px-4 py-3 border-t border-border-light"><span :class="'agent-type agent-type-'+a.agent_type">{{ a.agent_type }}</span></td>
        <td class="px-4 py-3 border-t border-border-light"><span :class="'badge badge-'+a.status">{{ a.status }}</span></td>
        <td class="px-4 py-3 border-t border-border-light text-xs text-text-muted">{{ a.config?.model || '-' }}</td>
        <td class="px-4 py-3 border-t border-border-light">
          <div class="flex gap-1">
            <button :disabled="a.status === 'deleting'" @click="openEdit(a)" title="Config" class="px-2 py-1 bg-btn border border-border rounded text-xs hover:bg-btn-hover transition" :class="a.status === 'deleting' ? 'opacity-50 cursor-not-allowed' : ''">⚙️</button>
            <button :disabled="a.status === 'deleting'" @click="restartAgent(a)" title="Restart" class="px-2 py-1 bg-accent-blue border border-accent-blue rounded text-xs text-white hover:opacity-90 transition" :class="a.status === 'deleting' ? 'opacity-50 cursor-not-allowed' : ''">🔄</button>
            <button :disabled="a.status === 'deleting'" @click="deleteAgent(a)" title="Delete" class="px-2 py-1 bg-btn-red border border-accent-red rounded text-xs hover:bg-btn-redHover transition" :class="a.status === 'deleting' ? 'opacity-50 cursor-not-allowed' : ''">🗑️</button>
          </div>
        </td>
      </tr>
    </DataTable>
    <p v-if="agents.length === 0" class="text-text-muted text-center py-10">No agents deployed yet. Go to Nodes to deploy one.</p>

    <EditAgentModal :show="showEdit" :agent="editAgent" @close="showEdit = false" @saved="load" />
  </div>
</template>

<script setup>
import { ref, onMounted, inject } from 'vue'
import { api, apiPost, apiDelete } from '../api'
import { useLoader } from '../composables/useLoader'
import DataTable from '../components/DataTable.vue'
import EditAgentModal from '../components/EditAgentModal.vue'

const toast = inject('toast')
const agents = ref([])
const showEdit = ref(false)
const editAgent = ref({})

const { load } = useLoader(async () => {
  agents.value = await api('/api/agents') || []
})

onMounted(load)

function openEdit(a) { editAgent.value = a; showEdit.value = true }

async function restartAgent(a) {
  if (!confirm(`Restart agent "${a.name}"?`)) return
  try {
    await apiPost(`/api/agents/${a.id}/restart`, {})
    toast('Agent restarting...')
    load()
  } catch (e) { alert('Restart failed: ' + e.message) }
}

async function deleteAgent(a) {
  if (!confirm(`Delete agent "${a.name}"? This will stop and remove the container.`)) return
  try {
    await apiDelete(`/api/agents/${a.id}`)
    toast('Agent deleting...')
    load()
    // Best-effort: refresh a few times so it disappears when purge completes
    setTimeout(load, 1200)
    setTimeout(load, 2500)
    setTimeout(load, 4000)
  } catch (e) { alert('Delete failed: ' + e.message) }
}
</script>
