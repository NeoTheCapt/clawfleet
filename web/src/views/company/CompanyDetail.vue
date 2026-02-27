<template>
  <div>
    <!-- Company selector / actions -->
    <div class="flex items-center gap-3 mb-4">
      <label class="text-sm text-text-muted">Company:</label>
      <select v-model="companyId" @change="onCompanyChange" class="px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm">
        <option v-for="c in companies" :key="c.id" :value="c.id">{{ c.name }}</option>
      </select>
      <router-link to="/companies" class="px-2.5 py-1 border border-border rounded-md bg-btn text-text text-xs hover:bg-btn-hover transition">+ New</router-link>
      <button @click="deleteCo" class="px-2.5 py-1 bg-btn-red border border-accent-red rounded-md text-xs text-text hover:bg-btn-redHover transition">🗑️</button>
    </div>

    <!-- Sub tabs -->
    <div class="flex border-b border-border mb-5 flex-wrap">
      <button v-for="t in tabs" :key="t.key" @click="tab = t.key"
        class="px-4 py-2.5 text-sm border-b-2 transition whitespace-nowrap"
        :class="tab === t.key ? 'text-accent-blue border-accent-blue' : 'text-text-muted border-transparent hover:text-text'">
        {{ t.label }}
      </button>
    </div>

    <component :is="currentTabComponent" :company-id="companyId" :key="companyId + tab" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, inject } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, apiDelete } from '../../api'
import Overview from './Overview.vue'
import OrgStructure from './OrgStructure.vue'
import KPI from './KPI.vue'
import IMGroups from './IMGroups.vue'
import Workflows from './Workflows.vue'
import Governance from './Governance.vue'
import Bots from './Bots.vue'
import Work from './Work.vue'

const toast = inject('toast')
const route = useRoute()
const router = useRouter()
const companyId = ref(route.params.id)
const companies = ref([])
const tab = ref('overview')

const tabs = [
  { key: 'overview', label: '📋 Overview' },
  { key: 'org', label: '🏗️ Org Structure' },
  { key: 'bots', label: '🤖 Telegram Bots' },
  { key: 'kpi', label: '📊 KPI' },
  { key: 'im', label: '💬 IM Groups' },
  { key: 'work', label: '🧭 Work' },
  { key: 'workflows', label: '⚙️ Workflows' },
  { key: 'governance', label: '📜 Governance' }
]

const tabComponents = { overview: Overview, org: OrgStructure, bots: Bots, kpi: KPI, im: IMGroups, work: Work, workflows: Workflows, governance: Governance }
const currentTabComponent = computed(() => tabComponents[tab.value] || Overview)

onMounted(async () => { companies.value = await api('/api/companies') || [] })

function onCompanyChange() { router.replace(`/company/${companyId.value}`) }

async function deleteCo() {
  const co = companies.value.find(c => c.id === companyId.value)
  if (!confirm(`Delete company "${co?.name || companyId.value}"? This cannot be undone.`)) return
  await apiDelete(`/api/companies/${companyId.value}`)
  toast('Company deleted', '#da3633')
  router.push('/companies')
}
</script>
