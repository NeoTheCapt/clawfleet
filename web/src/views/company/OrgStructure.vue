<template>
  <div>
    <!-- Goals section -->
    <div class="flex justify-between mb-4">
      <h3 class="font-semibold">Company Goals 公司目标</h3>
      <button @click="openGoalModal()" class="px-2.5 py-1 bg-btn-green border border-btn-greenHover rounded-md text-xs text-text hover:bg-btn-greenHover transition">+ Add Goal</button>
    </div>
    <EmptyState v-if="goals.length === 0" emoji="🎯" message="No goals yet" />
    <DataTable v-else :columns="['Title','Priority','Status','Due','Actions']">
      <tr v-for="g in goals" :key="g.id" class="hover:bg-bg-hover">
        <td class="px-4 py-3 border-t border-border-light"><strong>{{ g.title || g.name }}</strong><div v-if="g.description" class="text-xs text-text-muted">{{ g.description.substring(0,80) }}</div></td>
        <td class="px-4 py-3 border-t border-border-light"><span :class="'badge badge-'+(g.priority||'medium')">{{ g.priority || 'medium' }}</span></td>
        <td class="px-4 py-3 border-t border-border-light"><span :class="'badge badge-'+(g.status||'pending').replace(/ /g,'_')">{{ g.status || 'pending' }}</span></td>
        <td class="px-4 py-3 border-t border-border-light text-xs text-text-muted">{{ g.due_date || g.deadline || '-' }}</td>
        <td class="px-4 py-3 border-t border-border-light">
          <button @click="openGoalModal(g)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition mr-1">✏️</button>
          <button @click="delGoal(g.id)" class="px-2 py-0.5 bg-btn-red border border-accent-red rounded text-xs hover:bg-btn-redHover transition">🗑️</button>
        </td>
      </tr>
    </DataTable>

    <!-- Departments section -->
    <div class="flex justify-between mt-8 mb-4">
      <h3 class="font-semibold">Organization Structure 组织架构</h3>
      <button @click="openDeptModal()" class="px-2.5 py-1 bg-btn-green border border-btn-greenHover rounded-md text-xs text-text hover:bg-btn-greenHover transition">+ Add Department</button>
    </div>
    <EmptyState v-if="depts.length === 0" emoji="🏗️" message="No departments yet" />
    <div v-for="d in depts" :key="d.id" class="bg-bg-card border border-border rounded-lg mb-3 overflow-hidden">
      <div class="px-[18px] py-3.5 flex justify-between items-center cursor-pointer hover:bg-bg-hover">
        <h4 class="text-[15px] font-semibold">🏢 {{ d.name }} <span v-if="d.description" class="font-normal text-xs text-text-muted">— {{ d.description }}</span></h4>
        <div class="flex gap-1.5">
          <button @click="openPosModal(d.id)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition">+ Position</button>
          <button @click="openDeptModal(d)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition">✏️</button>
          <button @click="delDept(d.id)" class="px-2 py-0.5 bg-btn-red border border-accent-red rounded text-xs hover:bg-btn-redHover transition">🗑️</button>
        </div>
      </div>
      <div class="border-t border-border-light px-[18px] py-3">
        <div v-if="(deptPositions[d.id]||[]).length === 0" class="text-text-dim text-sm py-1">No positions</div>
        <div v-for="p in deptPositions[d.id]||[]" :key="p.id" class="flex items-center justify-between px-3 py-2 my-1 bg-bg-input rounded-md text-sm">
          <div class="flex items-center gap-2">
            <span>👤</span>
            <strong>{{ p.title || p.name }}</strong>
            <span v-if="p.level" :class="'badge badge-' + ({'c-suite':'running','director':'running','manager':'creating','staff':'stopped'}[p.level]||'stopped')">{{ p.level }}</span>
            <span class="text-text-dim text-[10px] font-mono opacity-60" :title="p.id">{{ p.id }}</span>
            <span v-if="p.reports_to" class="text-text-muted text-[10px]">→ {{ p.reports_to }}</span>
            <span v-if="p.description" class="text-text-muted text-xs">— {{ p.description.substring(0,60) }}</span>
          </div>
          <div class="flex gap-1">
            <button @click="syncPersona(p.id)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition" title="Sync persona to assigned agents">🔄</button>
            <button @click="openPosModal(d.id, p)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition">✏️</button>
            <button @click="delPos(d.id, p.id)" class="px-2 py-0.5 bg-btn-red border border-accent-red rounded text-xs hover:bg-btn-redHover transition">🗑️</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Assignments section -->
    <div class="flex justify-between mt-8 mb-4">
      <h3 class="font-semibold">Agent Assignments 智能体分配</h3>
      <button @click="openAssignModal()" class="px-2.5 py-1 bg-btn-green border border-btn-greenHover rounded-md text-xs text-text hover:bg-btn-greenHover transition">+ Assign Agent</button>
    </div>
    <EmptyState v-if="assignments.length === 0" emoji="👥" message="No assignments yet" />
    <DataTable v-else :columns="['Agent','Position','Status','Actions']">
      <tr v-for="a in assignments" :key="a.id" class="hover:bg-bg-hover">
        <td class="px-4 py-3 border-t border-border-light font-semibold">{{ a.agent_name || a.agent_id }}</td>
        <td class="px-4 py-3 border-t border-border-light">{{ a.position_title || a.position_id }}</td>
        <td class="px-4 py-3 border-t border-border-light"><span :class="'badge badge-'+(a.status||'active')">{{ a.status || 'active' }}</span></td>
        <td class="px-4 py-3 border-t border-border-light">
          <button @click="openAssignModal(a)" class="px-2 py-0.5 border border-border rounded bg-btn text-xs hover:bg-btn-hover transition mr-1">✏️</button>
          <button @click="delAssign(a.id)" class="px-2 py-0.5 bg-btn-red border border-accent-red rounded text-xs hover:bg-btn-redHover transition">🗑️</button>
        </td>
      </tr>
    </DataTable>

    <!-- Goal Modal -->
    <Modal :show="showGoalModal" @close="showGoalModal = false">
      <h3 class="text-lg font-semibold mb-4">{{ editGoal.id ? '✏️ Edit' : '➕ New' }} Goal</h3>
      <FieldLabel>Title 标题</FieldLabel>
      <input v-model="editGoal.title" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Description 描述</FieldLabel>
      <textarea v-model="editGoal.description" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Priority 优先级</FieldLabel>
      <select v-model="editGoal.priority" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3">
        <option v-for="p in ['low','medium','high','critical']" :key="p" :value="p">{{ p }}</option>
      </select>
      <FieldLabel>Status 状态</FieldLabel>
      <select v-model="editGoal.status" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3">
        <option v-for="s in ['pending','in_progress','completed']" :key="s" :value="s">{{ s }}</option>
      </select>
      <FieldLabel>Due Date 截止日期</FieldLabel>
      <input v-model="editGoal.due_date" type="date" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <ModalActions :onCancel="() => (showGoalModal = false)" :onSave="saveGoal" />
    </Modal>

    <!-- Dept Modal -->
    <Modal :show="showDeptModal" @close="showDeptModal = false">
      <h3 class="text-lg font-semibold mb-4">{{ editDept.id ? '✏️ Edit' : '➕ New' }} Department</h3>
      <FieldLabel>Name 名称</FieldLabel>
      <input v-model="editDept.name" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Description 描述</FieldLabel>
      <textarea v-model="editDept.description" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Parent Department ID (optional)</FieldLabel>
      <input v-model="editDept.parent_id" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <ModalActions :onCancel="() => (showDeptModal = false)" :onSave="saveDept" />
    </Modal>

    <!-- Position Modal -->
    <Modal :show="showPosModal" @close="showPosModal = false">
      <h3 class="text-lg font-semibold mb-4">{{ editPos.id ? '✏️ Edit' : '➕ New' }} Position</h3>
      <FieldLabel>Title 职位</FieldLabel>
      <input v-model="editPos.title" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Description 描述</FieldLabel>
      <textarea v-model="editPos.description" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Level 级别</FieldLabel>
      <input v-model="editPos.level" placeholder="e.g. senior, junior, lead" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Reports To (Position ID)</FieldLabel>
      <input v-model="editPos.reports_to" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>System Prompt (Persona) 人设</FieldLabel>
      <textarea v-model="editPos.system_prompt" placeholder="Hey! Looks like I just came online. Who am I? Who are you? Let's figure this out together..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[120px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <div class="text-xs text-text-dim mb-2">Include: My name, nature, vibe, emoji, and info about the user (name, timezone). This prompt will be used when creating an agent for this position.</div>
      <FieldLabel>Background Context 背景</FieldLabel>
      <textarea v-model="editPos.background" placeholder="Additional context about this position's role, responsibilities, knowledge base..." class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Telegram Bot</FieldLabel>
      <select v-model="editPos.telegram_bot_id" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3">
        <option value="">-- None --</option>
        <option v-for="b in bots" :key="b.id" :value="b.id">{{ b.name }} ({{ b.bot_id }})</option>
      </select>
      <ModalActions :onCancel="() => (showPosModal = false)" :onSave="savePos" />
    </Modal>

    <!-- Assignment Modal -->
    <Modal :show="showAssignModal" @close="showAssignModal = false">
      <h3 class="text-lg font-semibold mb-4">{{ editAssign.id ? '✏️ Edit' : '➕ New' }} Assignment</h3>
      <FieldLabel>Agent 智能体</FieldLabel>
      <select v-model="editAssign.agent_id" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3">
        <option value="">-- Select --</option>
        <option v-for="a in allAgents" :key="a.id" :value="a.id">{{ a.name }} ({{ a.agent_type }})</option>
      </select>
      <FieldLabel>Position 岗位</FieldLabel>
      <select v-model="editAssign.position_id" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3">
        <option value="">-- Select --</option>
        <option v-for="p in allPositions" :key="p.id" :value="p.id">{{ p.title || p.name }} {{ p.department_name ? '('+p.department_name+')' : '' }}</option>
      </select>
<div class="text-text-dim text-xs mb-3 bg-bg-card p-3 rounded-md border border-border">
        <p>📝 <strong>Agent persona/background</strong> is configured in the agent's settings (Deploy/Edit modal).</p>
        <p class="mt-1">📡 <strong>IM configuration</strong> is also part of the agent's channel settings.</p>
        <p class="mt-1">This assignment only links the agent to a position.</p>
      </div>
      <div class="mt-4">
        <label class="flex items-center gap-2 text-sm mb-2">
          <input type="checkbox" v-model="createNewAgent" class="rounded border-border accent-accent-blue">
          Create new agent for this position
        </label>
        <div v-if="createNewAgent" class="bg-bg-input border border-border rounded-md p-3 mb-3">
          <FieldLabel>Agent Type</FieldLabel>
          <select v-model="newAgentType" class="w-full px-3 py-2 bg-bg-card border border-border rounded-md text-text text-sm mb-2">
            <option value="openclaw">OpenClaw</option>
            <option value="nanobot">NanoBot</option>
            <option value="zeroclaw">ZeroClaw</option>
          </select>
          <div class="text-xs text-text-dim">
            Agent will be created with the position's system prompt and background.
            Channel and API keys can be configured later via Agent Config.
          </div>
        </div>
      </div>
      <ModalActions :onCancel="() => (showAssignModal = false)" :onSave="saveAssign" />
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

// Data
const goals = ref([])
const depts = ref([])
const deptPositions = ref({})
const assignments = ref([])
const allAgents = ref([])
const bots = ref([])
const allPositions = ref([])

// Modals
const showGoalModal = ref(false)
const editGoal = ref({})
const showDeptModal = ref(false)
const editDept = ref({})
const showPosModal = ref(false)
const editPos = ref({ deptId: '' })
const showAssignModal = ref(false)
const editAssign = ref({})
const createNewAgent = ref(false)
const newAgentType = ref('openclaw')

const { load } = useLoader(async () => {
  goals.value = await api(`/api/companies/${cid()}/goals`).catch(() => []) || []
  depts.value = await api(`/api/companies/${cid()}/departments`).catch(() => []) || []
  bots.value = await api(`/api/companies/${cid()}/bots`).catch(() => []) || []
  const dp = {}
  await Promise.all(depts.value.map(async d => {
    try { dp[d.id] = await api(`/api/companies/${cid()}/departments/${d.id}/positions`) || [] } catch { dp[d.id] = [] }
  }))
  deptPositions.value = dp
  assignments.value = await api(`/api/companies/${cid()}/assignments`).catch(() => []) || []
})

onMounted(load)

// Goals
function openGoalModal(g) {
  editGoal.value = g ? { ...g } : { title: '', description: '', priority: 'medium', status: 'pending', due_date: '' }
  showGoalModal.value = true
}
async function saveGoal() {
  const b = { title: editGoal.value.title, description: editGoal.value.description, priority: editGoal.value.priority, status: editGoal.value.status, due_date: editGoal.value.due_date || null }
  if (!b.title) { alert('Title required'); return }
  if (editGoal.value.id) await apiPut(`/api/companies/${cid()}/goals/${editGoal.value.id}`, b)
  else await apiPost(`/api/companies/${cid()}/goals`, b)
  showGoalModal.value = false; toast('Goal saved'); load()
}
async function delGoal(id) { await confirmAndDelete('Delete this goal?', `/api/companies/${cid()}/goals/${id}`, toast, 'Deleted', '#da3633', load) }

// Departments
function openDeptModal(d) {
  editDept.value = d ? { ...d } : { name: '', description: '', parent_id: '' }
  showDeptModal.value = true
}
async function saveDept() {
  const b = { name: editDept.value.name, description: editDept.value.description, parent_id: editDept.value.parent_id || null }
  if (!b.name) { alert('Name required'); return }
  if (editDept.value.id) await apiPut(`/api/companies/${cid()}/departments/${editDept.value.id}`, b)
  else await apiPost(`/api/companies/${cid()}/departments`, b)
  showDeptModal.value = false; toast('Department saved'); load()
}
async function delDept(id) { await confirmAndDelete('Delete this department and all its positions?', `/api/companies/${cid()}/departments/${id}`, toast, 'Deleted', '#da3633', load) }

// Positions
function openPosModal(deptId, p) {
  editPos.value = p ? { ...p, deptId, telegram_bot_id: p.telegram_bot_id || '' } : { title: '', description: '', level: '', reports_to: '', telegram_bot_id: '', deptId }
  showPosModal.value = true
}
async function savePos() {
  const b = { 
    title: editPos.value.title, 
    description: editPos.value.description, 
    level: editPos.value.level, 
    reports_to: editPos.value.reports_to || null,
    system_prompt: editPos.value.system_prompt || '',
    background: editPos.value.background || '',
    telegram_bot_id: editPos.value.telegram_bot_id || ''
  }
  if (!b.title) { alert('Title required'); return }
  if (editPos.value.id) await apiPut(`/api/companies/${cid()}/departments/${editPos.value.deptId}/positions/${editPos.value.id}`, b)
  else await apiPost(`/api/companies/${cid()}/departments/${editPos.value.deptId}/positions`, b)
  showPosModal.value = false; toast('Position saved'); load()
}
async function delPos(deptId, posId) { await confirmAndDelete('Delete this position?', `/api/companies/${cid()}/departments/${deptId}/positions/${posId}`, toast, 'Deleted', '#da3633', load) }

async function syncPersona(posId) {
  try {
    const res = await apiPost(`/api/positions/${posId}/sync-persona`, {})
    const total = res?.total_agents ?? 0
    const ok = res?.success_count ?? 0
    const fail = res?.failure_count ?? 0
    toast(`Persona sync: ${ok}/${total} success${fail ? `, ${fail} failed` : ''}`)
  } catch (e) { alert('Sync failed: ' + e.message) }
}

// Assignments
async function openAssignModal(a) {
  allAgents.value = await api('/api/agents').catch(() => []) || []
  allPositions.value = await api(`/api/companies/${cid()}/positions`).catch(() => []) || []
  createNewAgent.value = false
  newAgentType.value = 'openclaw'
  if (a) {
    editAssign.value = { ...a }
  } else {
    editAssign.value = { agent_id: '', position_id: '', status: 'active' }
  }
  showAssignModal.value = true
}
async function saveAssign() {
  if (!editAssign.value.position_id) { alert('Position required'); return }
  
  if (createNewAgent.value) {
    // Create new agent with position's system prompt
    const position = allPositions.value.find(p => p.id === editAssign.value.position_id)
    if (!position) { alert('Position not found'); return }
    
    // Get any node (first one)
    const nodes = await api('/api/nodes').catch(() => [])
    if (nodes.length === 0) { alert('No nodes available'); return }
    const node = nodes[0]
    
    const agentReq = {
      name: `${position.title || 'Agent'} (${position.id.substring(0,8)})`,
      agent_type: newAgentType.value,
      role: position.title || '',
      company_id: cid(),
      deploy_mode: 'docker',
      config: {
        system_prompt: position.system_prompt || '',
        description: position.background || '',
        provider: 'openrouter',
        model: newAgentType.value === 'openclaw' ? 'openrouter/deepseek/deepseek-chat' : '',
        channel: { type: 'telegram', dm_policy: 'open', allow_from: ['*'] }
      }
    }
    
    try {
      const agent = await apiPost(`/api/nodes/${node.id}/deploy`, agentReq)
      editAssign.value.agent_id = agent.id
      toast('Agent created')
    } catch (e) {
      alert('Failed to create agent: ' + e.message)
      return
    }
  }
  
  if (!editAssign.value.agent_id) { alert('Agent required'); return }
  
  const b = {
    agent_id: editAssign.value.agent_id,
    position_id: editAssign.value.position_id,
    status: editAssign.value.status || 'active',
  }
  if (editAssign.value.id) await apiPut(`/api/companies/${cid()}/assignments/${editAssign.value.id}`, b)
  else await apiPost(`/api/companies/${cid()}/assignments`, b)
  showAssignModal.value = false; toast('Assignment saved'); load()
}
async function delAssign(id) { await confirmAndDelete('Remove this assignment?', `/api/companies/${cid()}/assignments/${id}`, toast, 'Removed', '#da3633', load) }
</script>
