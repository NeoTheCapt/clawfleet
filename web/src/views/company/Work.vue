<template>
  <div>
    <div class="bg-bg-card border border-border rounded-lg p-4 mb-5">
      <h3 class="font-semibold mb-3">CEO Execution Loop</h3>
      <label class="block text-xs text-text-muted mb-1">Goal</label>
      <textarea
        v-model="goalText"
        class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm min-h-[90px] resize-y focus:outline-none focus:border-accent-blue"
        placeholder="Describe the target outcome..."
      />
      <div class="mt-3 flex items-center gap-2">
        <button
          @click="generatePlan"
          class="px-3 py-1.5 bg-accent-blue border border-accent-blue rounded-md text-xs text-white hover:opacity-90 transition"
        >
          Generate Plan
        </button>
        <button
          @click="approvePlan"
          :disabled="!plan.items.length"
          class="px-3 py-1.5 bg-btn-green border border-btn-greenHover rounded-md text-xs text-text hover:bg-btn-greenHover transition disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Approve & Dispatch
        </button>
        <span v-if="warning" class="text-xs text-accent-red">{{ warning }}</span>
      </div>
    </div>

    <div v-if="plan.items.length" class="bg-bg-card border border-border rounded-lg p-4 mb-5">
      <h4 class="font-medium mb-3">Plan Items</h4>
      <div v-for="(item, i) in plan.items" :key="i" class="border border-border rounded-md p-3 mb-3 bg-bg-input">
        <label class="block text-xs text-text-muted mb-1">Title</label>
        <input v-model="item.title" class="w-full px-3 py-2 bg-bg-card border border-border rounded-md text-sm mb-2" />
        <label class="block text-xs text-text-muted mb-1">Description</label>
        <textarea v-model="item.description" class="w-full px-3 py-2 bg-bg-card border border-border rounded-md text-sm min-h-[60px] mb-2 resize-y" />
        <div class="grid grid-cols-1 md:grid-cols-3 gap-2">
          <div>
            <label class="block text-xs text-text-muted mb-1">Owner Position</label>
            <select v-model="item.owner_position_id" class="w-full px-3 py-2 bg-bg-card border border-border rounded-md text-sm">
              <option value="">-- Unassigned --</option>
              <option v-for="p in positions" :key="p.id" :value="p.id">{{ p.title }} ({{ p.id }})</option>
            </select>
          </div>
          <div>
            <label class="block text-xs text-text-muted mb-1">Priority</label>
            <select v-model="item.priority" class="w-full px-3 py-2 bg-bg-card border border-border rounded-md text-sm">
              <option value="low">low</option>
              <option value="medium">medium</option>
              <option value="high">high</option>
            </select>
          </div>
          <div>
            <label class="block text-xs text-text-muted mb-1">Due</label>
            <input v-model="item.due_at" type="datetime-local" class="w-full px-3 py-2 bg-bg-card border border-border rounded-md text-sm" />
          </div>
        </div>
      </div>
    </div>

    <h4 class="font-medium mb-2">Active Work Items</h4>
    <DataTable :columns="['Title','Owner','Status','Priority','Due','Actions']">
      <tr v-for="w in workItems" :key="w.id" class="hover:bg-bg-hover">
        <td class="px-4 py-3 border-t border-border-light">
          <div class="font-semibold">{{ w.title }}</div>
          <div v-if="w.description" class="text-xs text-text-muted">{{ w.description }}</div>
        </td>
        <td class="px-4 py-3 border-t border-border-light text-xs">{{ agentName(w.owner_agent_id) }}</td>
        <td class="px-4 py-3 border-t border-border-light">
          <span :class="'badge badge-'+(w.status || 'approved')">{{ w.status || 'approved' }}</span>
        </td>
        <td class="px-4 py-3 border-t border-border-light text-xs">{{ w.priority || 'medium' }}</td>
        <td class="px-4 py-3 border-t border-border-light text-xs">{{ fmtDue(w.due_at) }}</td>
        <td class="px-4 py-3 border-t border-border-light">
          <select
            :value="w.status"
            @change="setStatus(w.id, $event.target.value)"
            class="px-2 py-1 bg-bg-input border border-border rounded-md text-xs"
          >
            <option v-for="s in statuses" :key="s" :value="s">{{ s }}</option>
          </select>
        </td>
      </tr>
    </DataTable>
  </div>
</template>

<script setup>
import { inject, onMounted, ref } from 'vue'
import { api, apiPost } from '../../api'
import DataTable from '../../components/DataTable.vue'

const props = defineProps({ companyId: String })
const toast = inject('toast', () => {})
const cid = () => props.companyId

const goalText = ref('')
const warning = ref('')
const positions = ref([])
const agents = ref([])
const workItems = ref([])
const statuses = ['draft', 'proposed', 'approved', 'in_progress', 'done', 'cancelled']
const plan = ref({ items: [] })

async function load() {
  positions.value = await api(`/api/companies/${cid()}/positions`).catch(() => []) || []
  agents.value = await api('/api/agents').catch(() => []) || []
  workItems.value = await api(`/api/work?company_id=${cid()}&status=`).catch(() => []) || []
  workItems.value = workItems.value.filter(w => ['approved', 'in_progress', 'proposed', 'draft'].includes(w.status))
}

async function generatePlan() {
  if (!goalText.value.trim()) {
    alert('Goal is required')
    return
  }
  const res = await apiPost('/api/work/plan', { company_id: cid(), goal_text: goalText.value.trim() })
  warning.value = res.warning || ''
  plan.value = res.plan || { items: [] }
  for (const item of plan.value.items || []) {
    item.priority = item.priority || 'medium'
    item.owner_position_id = item.owner_position_id || ''
    item.due_at = normalizeDueInput(item.due_at || '')
  }
}

async function approvePlan() {
  const items = (plan.value.items || []).map(i => ({
    title: (i.title || '').trim(),
    description: i.description || '',
    owner_position_id: i.owner_position_id || '',
    priority: i.priority || 'medium',
    due_at: toRFC3339(i.due_at),
  }))
  if (!items.length || !items.every(i => i.title)) {
    alert('All plan items need title')
    return
  }
  await apiPost('/api/work/approve', {
    company_id: cid(),
    goal_text: goalText.value.trim(),
    plan: { items },
  })
  toast('Plan approved and dispatched')
  plan.value = { items: [] }
  await load()
}

async function setStatus(id, status) {
  await apiPost(`/api/work/${id}/status`, { status })
  toast('Status updated')
  await load()
}

function agentName(id) {
  if (!id) return '-'
  return agents.value.find(a => a.id === id)?.name || id
}

function fmtDue(v) {
  if (!v) return '-'
  return new Date(v).toLocaleString()
}

function toRFC3339(v) {
  if (!v) return ''
  const d = new Date(v)
  return Number.isNaN(d.getTime()) ? '' : d.toISOString()
}

function normalizeDueInput(v) {
  if (!v) return ''
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return ''
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  return `${y}-${m}-${day}T${hh}:${mm}`
}

onMounted(load)
</script>
