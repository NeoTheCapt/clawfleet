<template>
  <div>
    <div class="flex justify-between mb-4">
      <h3 class="font-semibold">Governance Logs 审计日志</h3>
      <button @click="load" class="px-2.5 py-1 border border-border rounded-md bg-btn text-xs text-text hover:bg-btn-hover transition">🔄 Refresh</button>
    </div>
    <EmptyState v-if="logs.length === 0" emoji="📜" message="No governance logs yet" />
    <DataTable v-else :columns="['Time','Actor','Action','Resource','Details']">
      <tr v-for="l in logs" :key="l.id" class="hover:bg-bg-hover">
        <td class="px-4 py-3 border-t border-border-light text-xs text-text-muted whitespace-nowrap">{{ formatTime(l) }}</td>
        <td class="px-4 py-3 border-t border-border-light">{{ l.actor || l.user || '-' }}</td>
        <td class="px-4 py-3 border-t border-border-light"><span :class="actionBadge(l.action)">{{ l.action || '-' }}</span></td>
        <td class="px-4 py-3 border-t border-border-light text-sm">{{ l.resource_type || l.resource || '-' }} <span v-if="l.resource_id" class="text-text-dim">#{{ l.resource_id.substring(0,8) }}</span></td>
        <td class="px-4 py-3 border-t border-border-light text-xs text-text-muted max-w-[250px] truncate">{{ l.details || l.description || '-' }}</td>
      </tr>
    </DataTable>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../../api'
import { useLoader } from '../../composables/useLoader'
import DataTable from '../../components/DataTable.vue'
import EmptyState from '../../components/EmptyState.vue'

const props = defineProps({ companyId: String })
const logs = ref([])

const { load } = useLoader(async () => {
  logs.value = await api(`/api/companies/${props.companyId}/governance-logs`).catch(() => []) || []
})

onMounted(load)

function formatTime(l) {
  const t = l.created_at || l.timestamp
  return t ? new Date(t).toLocaleString() : '-'
}

function actionBadge(action) {
  const base = 'badge '
  if (action === 'delete') return base + 'badge-error'
  if (action === 'create') return base + 'badge-online'
  return base + 'badge-pending'
}
</script>
