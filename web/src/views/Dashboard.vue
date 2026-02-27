<template>
  <div>
    <div class="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-4 mb-6">
      <StatCard label="Nodes Online" :value="`${onlineNodes}/${nodes.length}`" color="green" />
      <StatCard label="Agents Running" :value="`${runningAgents}/${agents.length}`" color="blue" />
      <StatCard label="Companies" :value="companies.length" color="yellow" />
      <StatCard label="Departments" :value="totalDepts" color="purple" />
      <StatCard label="Positions" :value="totalPositions" color="blue" />
      <StatCard label="Assignments" :value="totalAssignments" color="green" />
    </div>
    <h3 class="mb-3 font-semibold">Recent Agents</h3>
    <DataTable :columns="['Name','Type','Role','Node','Status']">
      <tr v-for="a in agents.slice(0,10)" :key="a.id" class="hover:bg-bg-hover">
        <td class="px-4 py-3 border-t border-border-light">{{ a.name }}</td>
        <td class="px-4 py-3 border-t border-border-light"><span :class="'agent-type agent-type-'+a.agent_type">{{ a.agent_type }}</span></td>
        <td class="px-4 py-3 border-t border-border-light">{{ a.role }}</td>
        <td class="px-4 py-3 border-t border-border-light">{{ a.node_id ? a.node_id.substring(0,20)+'...' : '-' }}</td>
        <td class="px-4 py-3 border-t border-border-light"><span :class="'badge badge-'+a.status">{{ a.status }}</span></td>
      </tr>
    </DataTable>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import StatCard from '../components/StatCard.vue'
import DataTable from '../components/DataTable.vue'

const nodes = ref([])
const agents = ref([])
const companies = ref([])
const onlineNodes = ref(0)
const runningAgents = ref(0)
const totalDepts = ref(0)
const totalPositions = ref(0)
const totalAssignments = ref(0)

onMounted(async () => {
  try {
    const [n, a] = await Promise.all([api('/api/nodes'), api('/api/agents')])
    nodes.value = Array.isArray(n) ? n : []
    agents.value = Array.isArray(a) ? a : []
    onlineNodes.value = nodes.value.filter(x => x.status === 'online').length
    runningAgents.value = agents.value.filter(x => x.status === 'running').length

    try { companies.value = await api('/api/companies') || [] } catch {}

    for (const c of companies.value) {
      try {
        const chart = await api(`/api/companies/${c.id}/org-chart`)
        if (chart) {
          totalDepts.value += (chart.departments || []).length
          ;(chart.departments || []).forEach(d => totalPositions.value += (d.positions || []).length)
        }
      } catch {}
      try {
        const assignments = await api(`/api/companies/${c.id}/assignments`)
        totalAssignments.value += (assignments || []).length
      } catch {}
    }
  } catch {}
})
</script>
