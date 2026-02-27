<template>
  <div>
    <div class="flex justify-between mb-4">
      <h3 class="font-semibold">KPI Reviews 绩效考核</h3>
      <div class="flex gap-2">
        <button @click="triggerDailyNow()" class="px-2.5 py-1 bg-accent-blue border border-accent-blue rounded-md text-xs text-white hover:opacity-90 transition">Trigger Daily Review Now</button>
        <button @click="openCycleModal()" class="px-2.5 py-1 bg-btn-green border border-btn-greenHover rounded-md text-xs text-text hover:bg-btn-greenHover transition">+ New Review Cycle</button>
      </div>
    </div>
    <div class="bg-bg-card border border-border rounded-lg p-5 mb-4">
      <div class="flex items-center justify-between mb-3">
        <h4 class="font-medium text-sm">Today's Reviews ({{ todayPeriod() }})</h4>
        <button @click="loadTodayReviews()" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition">Refresh</button>
      </div>
      <div v-if="todayReviews.length === 0" class="text-text-dim text-sm">No reviews generated for today yet.</div>
      <table v-else class="w-full text-sm">
        <tr>
          <th class="text-left px-4 py-2 text-text-muted text-xs uppercase">Reviewee</th>
          <th class="text-left px-4 py-2 text-text-muted text-xs uppercase">Reviewer</th>
          <th class="text-left px-4 py-2 text-text-muted text-xs uppercase">Status</th>
          <th class="text-left px-4 py-2 text-text-muted text-xs uppercase">Feedback</th>
        </tr>
        <tr v-for="r in todayReviews" :key="r.id">
          <td class="px-4 py-2">{{ agentName(r.agent_id) }}</td>
          <td class="px-4 py-2">{{ agentName(r.reviewer_id) }}</td>
          <td class="px-4 py-2"><span :class="'badge badge-'+(r.status||'pending')">{{ r.status || 'pending' }}</span></td>
          <td class="px-4 py-2 text-xs text-text-muted max-w-[350px] truncate">{{ r.feedback || '-' }}</td>
        </tr>
      </table>
    </div>
    <EmptyState v-if="cycles.length === 0" emoji="📊" message="No review cycles yet" />
    <div v-for="c in cycles" :key="c.id" class="bg-bg-card border border-border rounded-lg p-5 mb-4">
      <div class="flex justify-between items-center mb-3">
        <div>
          <strong>{{ c.name || c.title || 'Cycle' }}</strong>
          <span :class="'badge badge-'+(c.status||'active')" class="ml-2">{{ c.status || 'active' }}</span>
          <span v-if="c.frequency" class="badge badge-active ml-2 text-xs">{{ c.frequency }}</span>
          <span v-if="c.auto_review" class="text-xs ml-1" title="Auto Review Enabled">🤖</span>
          <span class="text-xs text-text-muted ml-2">{{ c.metrics && c.metrics.length ? c.metrics.join(' · ') : '' }}</span>
        </div>
        <div class="flex gap-1.5">
          <button @click="loadReviews(c.id)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition">📋 Reviews</button>
          <button @click="openCycleModal(c)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition">✏️</button>
          <button @click="delCycle(c.id)" class="px-2 py-0.5 bg-btn-red border border-accent-red rounded text-xs hover:bg-btn-redHover transition">🗑️</button>
        </div>
      </div>
      <div v-if="cycleReviews[c.id]">
        <div v-if="cycleReviews[c.id].length === 0" class="text-text-dim text-sm">
          No reviews. <button @click="openReviewModal(c.id)" class="px-2 py-0.5 bg-btn-green border border-btn-greenHover rounded text-xs hover:bg-btn-greenHover transition">+ Add Review</button>
        </div>
        <div v-else>
          <button @click="openReviewModal(c.id)" class="px-2 py-0.5 bg-btn-green border border-btn-greenHover rounded text-xs hover:bg-btn-greenHover transition mb-2">+ Add Review</button>
          <table class="w-full text-sm">
            <tr><th class="text-left px-4 py-2 text-text-muted text-xs uppercase">Agent</th><th class="text-left px-4 py-2 text-text-muted text-xs uppercase">Score</th><th class="text-left px-4 py-2 text-text-muted text-xs uppercase">Summary</th><th class="text-left px-4 py-2 text-text-muted text-xs uppercase">Period</th></tr>
            <tr v-for="r in cycleReviews[c.id]" :key="r.id">
              <td class="px-4 py-2">{{ r.agent_name || r.agent_id }}</td>
              <td class="px-4 py-2"><strong :style="{ color: r.score >= 80 ? '#3fb950' : r.score >= 60 ? '#d29922' : '#f85149' }">{{ r.score ?? '-' }}</strong></td>
              <td class="px-4 py-2 text-xs text-text-muted max-w-[250px] truncate">{{ r.summary || r.comment || '-' }}</td>
              <td class="px-4 py-2 text-xs text-text-muted">{{ r.period || '-' }}</td>
            </tr>
          </table>
        </div>
      </div>
    </div>

    <!-- Cycle Modal -->
    <Modal :show="showCycleModal" @close="showCycleModal = false">
      <h3 class="text-lg font-semibold mb-4">{{ editCycle.id ? '✏️ Edit' : '➕ New' }} Review Cycle</h3>
      <FieldLabel>Name 名称</FieldLabel>
      <input v-model="editCycle.name" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <div class="grid grid-cols-2 gap-3 mb-3">
        <div>
          <FieldLabel>Frequency 频率</FieldLabel>
          <select v-model="editCycle.frequency" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
            <option value="weekly">Weekly 每周</option>
            <option value="biweekly">Biweekly 双周</option>
            <option value="monthly">Monthly 每月</option>
            <option value="quarterly">Quarterly 每季</option>
          </select>
        </div>
        <div>
          <FieldLabel>Status</FieldLabel>
          <select v-model="editCycle.status" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm focus:outline-none focus:border-accent-blue">
            <option v-for="s in ['active','completed','pending']" :key="s" :value="s">{{ s }}</option>
          </select>
        </div>
      </div>
      <FieldLabel>Metrics 考核指标（逗号分隔）</FieldLabel>
      <input v-model="editCycle.metrics_str" placeholder="e.g. 代码质量,任务完成率,响应速度" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Report Template 报告模板 (optional)</FieldLabel>
      <textarea v-model="editCycle.report_template" placeholder="请从以下维度自评：&#10;1. 本周完成的工作&#10;2. 遇到的困难&#10;3. 下周计划" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <div class="flex items-center gap-2 mb-3">
        <input v-model="editCycle.auto_review" type="checkbox" id="auto-review" class="accent-accent-blue">
        <label for="auto-review" class="text-sm text-text-muted">🤖 Auto Review — 到期时自动触发 agent 自评</label>
      </div>
      <ModalActions :onCancel="() => (showCycleModal = false)" :onSave="saveCycle" />
    </Modal>

    <!-- Review Modal -->
    <Modal :show="showReviewModal" @close="showReviewModal = false">
      <h3 class="text-lg font-semibold mb-4">➕ New KPI Review</h3>
      <FieldLabel>Agent</FieldLabel>
      <select v-model="editReview.agent_id" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3">
        <option v-for="a in agents" :key="a.id" :value="a.id">{{ a.name }}</option>
      </select>
      <FieldLabel>Score (0-100)</FieldLabel>
      <input v-model.number="editReview.score" type="number" min="0" max="100" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Period</FieldLabel>
      <input v-model="editReview.period" placeholder="e.g. 2025-01" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Summary 总结</FieldLabel>
      <textarea v-model="editReview.summary" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <ModalActions :onCancel="() => (showReviewModal = false)" :onSave="saveReview" />
    </Modal>
  </div>
</template>

<script setup>
import { ref, onMounted, inject } from 'vue'
import { api, apiPost, apiPut, confirmAndDelete } from '../../api'
import { useLoader } from '../../composables/useLoader'
import EmptyState from '../../components/EmptyState.vue'
import Modal from '../../components/Modal.vue'
import ModalActions from '../../components/ModalActions.vue'
import FieldLabel from '../../components/FieldLabel.vue'

const props = defineProps({ companyId: String })
const toast = inject('toast')
const cid = () => props.companyId

const cycles = ref([])
const cycleReviews = ref({})
const agents = ref([])
const agentMap = ref({})
const todayReviews = ref([])
const showCycleModal = ref(false)
const editCycle = ref({})
const showReviewModal = ref(false)
const editReview = ref({ cycle_id: '', agent_id: '', score: 80, period: '', summary: '' })

const { load } = useLoader(async () => {
  cycles.value = await api(`/api/companies/${cid()}/review-cycles`).catch(() => []) || []
  await loadTodayReviews()
})

onMounted(load)

function todayPeriod() {
  const d = new Date()
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

async function ensureAgentMap() {
  if (Object.keys(agentMap.value).length > 0) return
  const list = await api('/api/agents').catch(() => []) || []
  const next = {}
  for (const a of list) next[a.id] = a
  agentMap.value = next
}

function agentName(id) {
  if (!id) return '-'
  return agentMap.value[id]?.name || id
}

async function loadTodayReviews() {
  await ensureAgentMap()
  todayReviews.value = await api(`/api/companies/${cid()}/kpi-reviews?period=${todayPeriod()}`).catch(() => []) || []
}

async function triggerDailyNow() {
  const res = await apiPost('/api/kpi/trigger-daily', { company_id: cid(), scope: 'company' })
  toast(`Daily reviews: ${res.created_count || 0} new, ${res.existing_count || 0} existing`)
  await load()
}

async function loadReviews(cycleId) {
  cycleReviews.value[cycleId] = await api(`/api/companies/${cid()}/kpi-reviews?cycle_id=${cycleId}`).catch(() => []) || []
}

function openCycleModal(c) {
  if (c) {
    editCycle.value = { ...c, metrics_str: (c.metrics || []).join(', '), auto_review: c.auto_review || false }
  } else {
    editCycle.value = { name: '', frequency: 'monthly', status: 'active', metrics_str: '', report_template: '', auto_review: false }
  }
  showCycleModal.value = true
}

async function saveCycle() {
  const metrics = editCycle.value.metrics_str ? editCycle.value.metrics_str.split(',').map(s => s.trim()).filter(Boolean) : []
  const b = {
    name: editCycle.value.name,
    frequency: editCycle.value.frequency,
    metrics: metrics,
    report_template: editCycle.value.report_template || '',
    auto_review: editCycle.value.auto_review || false,
    status: editCycle.value.status,
  }
  if (!b.name) { alert('Name required'); return }
  if (editCycle.value.id) await apiPut(`/api/companies/${cid()}/review-cycles/${editCycle.value.id}`, b)
  else await apiPost(`/api/companies/${cid()}/review-cycles`, b)
  showCycleModal.value = false; toast('Saved'); load()
}

async function delCycle(id) { await confirmAndDelete('Delete this review cycle?', `/api/companies/${cid()}/review-cycles/${id}`, toast, 'Deleted', '#da3633', load) }

async function openReviewModal(cycleId) {
  agents.value = await api('/api/agents').catch(() => []) || []
  editReview.value = { cycle_id: cycleId, agent_id: agents.value[0]?.id || '', score: 80, period: '', summary: '' }
  showReviewModal.value = true
}

async function saveReview() {
  await apiPost(`/api/companies/${cid()}/kpi-reviews`, editReview.value)
  showReviewModal.value = false; toast('Review saved'); loadReviews(editReview.value.cycle_id)
}
</script>
