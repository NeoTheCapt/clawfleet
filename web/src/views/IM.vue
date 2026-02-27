<template>
  <div class="p-4 md:p-0">
    <div class="flex flex-wrap items-center justify-between gap-2 mb-4">
      <h3 class="font-semibold">Web IM</h3>
      <div class="flex items-center gap-2">
        <select
          v-model="selectedCompanyId"
          class="px-3 py-1.5 border border-border rounded-md bg-bg-input text-xs focus:outline-none focus:border-accent-blue"
          @change="onCompanyChange"
        >
          <option value="">Select company</option>
          <option v-for="co in companies" :key="co.id" :value="co.id">{{ co.name }}</option>
        </select>
        <button @click="syncOrg" class="px-3 py-1.5 border border-border rounded-md bg-btn text-xs hover:bg-btn-hover transition">Sync Org</button>
        <button @click="bootstrapCEO" class="px-3 py-1.5 border border-border rounded-md bg-btn text-xs hover:bg-btn-hover transition">Bootstrap CEO</button>
      </div>
    </div>

    <div class="border border-border rounded-xl overflow-hidden bg-bg-card grid grid-cols-1 md:grid-cols-[260px_1fr_300px] h-[72vh]">
      <aside class="border-r border-border overflow-y-auto">
        <div v-if="error" class="p-3 text-xs text-accent-red border-b border-border">{{ error }}</div>
        <div
          v-for="c in conversations"
          :key="c.id"
          @click="selectConversation(c.id)"
          class="w-full text-left px-4 py-3 border-b border-border-light hover:bg-bg-hover transition"
          :class="activeConversationId === c.id ? 'bg-bg-hover' : ''"
        >
          <div class="text-sm font-medium flex items-center justify-between gap-2">
            <span class="truncate">{{ c.title }}</span>
            <button
              v-if="isJWTUser"
              @click.stop="deleteConversation(c)"
              class="shrink-0 text-xs px-1 py-0.5 border border-border rounded hover:bg-bg-main"
              title="Delete conversation"
              aria-label="Delete conversation"
            >🗑️</button>
          </div>
          <div class="text-xs text-text-muted mt-0.5">{{ c.type }} · {{ c.member_count || 0 }} members</div>
        </div>
        <div v-if="!conversations.length" class="p-4 text-sm text-text-muted">No conversations yet.</div>
      </aside>

      <section class="flex flex-col min-h-0">
        <div class="px-4 py-3 border-b border-border">
          <div class="font-medium text-sm">{{ activeConversation ? activeConversation.title : 'Select a conversation' }}</div>
          <div v-if="activeConversation" class="text-xs text-text-muted">{{ activeConversation.type }} · ref {{ activeConversation.ref_id || '-' }}</div>
        </div>

        <div class="flex-1 overflow-y-auto p-4 space-y-3 bg-bg-main">
          <div v-for="m in messages" :key="m.id" class="rounded-lg border border-border bg-bg-card px-3 py-2">
            <div class="text-xs text-text-muted mb-1">{{ m.sender_type }}:{{ m.sender_id }} · {{ fmtTime(m.created_at) }}</div>
            <div class="text-sm whitespace-pre-wrap">{{ m.body }}</div>
          </div>
          <div v-if="activeConversation && !messages.length" class="text-sm text-text-muted">No messages yet.</div>
        </div>

        <form class="p-3 border-t border-border space-y-2" @submit.prevent="sendMessage">
          <div v-if="selectedMentionAgents.length" class="flex flex-wrap gap-1">
            <span v-for="a in selectedMentionAgents" :key="a.member_id" class="inline-flex items-center gap-1 px-2 py-0.5 border border-border rounded text-xs bg-bg-card">
              @{{ agentName(a) }}
              <button type="button" @click="removeMention(a.member_id)" class="text-text-muted hover:text-text">x</button>
            </span>
          </div>
          <div class="relative flex gap-2">
            <input
              v-model="draft"
              :disabled="!activeConversation"
              placeholder="Type @ to mention agents..."
              class="flex-1 px-3 py-2 bg-bg-input border border-border rounded-md text-sm focus:outline-none focus:border-accent-blue"
              @input="onDraftInput"
              @keydown.down.prevent="moveMentionSelection(1)"
              @keydown.up.prevent="moveMentionSelection(-1)"
              @keydown.enter="onDraftEnter"
            >
            <button :disabled="!activeConversation || !draft.trim()" class="px-3 py-2 bg-btn-green border border-btn-greenHover rounded-md text-xs text-text hover:bg-btn-greenHover transition disabled:opacity-50 disabled:cursor-not-allowed">Send</button>

            <div v-if="showMentionMenu && filteredMentionAgents.length" class="absolute left-0 right-16 bottom-11 border border-border rounded-md bg-bg-card shadow-md max-h-40 overflow-y-auto z-20">
              <button
                v-for="(a, idx) in filteredMentionAgents"
                :key="a.member_id"
                type="button"
                class="w-full text-left px-3 py-2 text-xs border-b border-border-light last:border-b-0 hover:bg-bg-hover"
                :class="idx === mentionSelection ? 'bg-bg-hover' : ''"
                @click="insertMention(a)"
              >
                <span class="font-medium">{{ agentName(a) }}</span>
                <span class="text-text-muted"> · {{ a.member_id }} · {{ a.agent_status || '-' }}</span>
              </button>
            </div>
          </div>
        </form>
      </section>

      <aside class="border-l border-border flex flex-col min-h-0">
        <div class="px-4 py-3 border-b border-border">
          <div class="font-medium text-sm">Agents</div>
          <div class="text-xs text-text-muted">Current conversation members</div>
        </div>
        <div class="flex-1 overflow-y-auto">
          <div v-for="m in agentMembers" :key="m.member_id" class="px-4 py-2 border-b border-border-light text-xs flex items-center justify-between gap-2">
            <div class="min-w-0">
              <div class="font-medium truncate">{{ agentName(m) }}</div>
              <div class="text-text-muted truncate">{{ m.member_id }} · {{ m.role || 'member' }}</div>
            </div>
            <div class="flex items-center gap-1 shrink-0">
              <span class="px-1.5 py-0.5 border border-border rounded">{{ m.agent_status || '-' }}</span>
              <button
                v-if="isChairmanUser"
                @click="kickAgent(m.member_id)"
                class="px-1.5 py-0.5 border border-border rounded hover:bg-bg-main"
                title="Remove agent"
              >Kick</button>
            </div>
          </div>
          <div v-if="activeConversation && !agentMembers.length" class="p-4 text-xs text-text-muted">No agent members.</div>
          <div v-if="!activeConversation" class="p-4 text-xs text-text-muted">Select a conversation.</div>
        </div>
        <div v-if="isChairmanUser && activeConversation" class="p-3 border-t border-border">
          <div class="text-xs font-medium mb-1">Invite Agent</div>
          <div class="flex gap-2">
            <select v-model="inviteAgentID" class="flex-1 px-2 py-1.5 border border-border rounded bg-bg-input text-xs focus:outline-none focus:border-accent-blue">
              <option value="">Select agent</option>
              <option v-for="a in inviteableAgents" :key="a.id" :value="a.id">{{ a.name || a.id }} ({{ a.status || '-' }})</option>
            </select>
            <button
              :disabled="!inviteAgentID"
              class="px-2 py-1.5 border border-border rounded text-xs hover:bg-bg-main disabled:opacity-50"
              @click="inviteAgent"
            >Invite</button>
          </div>
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, ref } from 'vue'
import { api, apiDelete, apiPost } from '../api'
import { getToken } from '../composables/useAuth'

const toast = inject('toast', () => {})
const companies = ref([])
const selectedCompanyId = ref('')
const conversations = ref([])
const messages = ref([])
const activeConversationId = ref('')
const draft = ref('')
const error = ref('')
const members = ref([])
const companyAgents = ref([])
const inviteAgentID = ref('')
const selectedMentionAgentIDs = ref([])
const showMentionMenu = ref(false)
const mentionQuery = ref('')
const mentionSelection = ref(0)

const activeConversation = computed(() => conversations.value.find(c => c.id === activeConversationId.value))
const isJWTUser = computed(() => !!getToken())
const currentUser = computed(() => decodeJWTSub(getToken()))
const isChairmanUser = computed(() => {
  const id = (currentUser.value || '').toLowerCase()
  return id === 'chairman' || id === 'brian'
})
const agentMembers = computed(() => members.value.filter(m => m.member_type === 'agent'))
const selectedMentionAgents = computed(() => {
  const pick = new Set(selectedMentionAgentIDs.value)
  return agentMembers.value.filter(m => pick.has(m.member_id))
})
const filteredMentionAgents = computed(() => {
  const q = mentionQuery.value.trim().toLowerCase()
  const list = agentMembers.value.filter(m => !selectedMentionAgentIDs.value.includes(m.member_id))
  if (!q) return list
  return list.filter(m =>
    (m.agent_display_name || '').toLowerCase().includes(q) ||
    (m.member_id || '').toLowerCase().includes(q)
  )
})
const inviteableAgents = computed(() => {
  const existing = new Set(agentMembers.value.map(m => m.member_id))
  return companyAgents.value.filter(a => !existing.has(a.id))
})

async function loadCompanies() {
  companies.value = await api('/api/companies')
  if (!selectedCompanyId.value && companies.value.length) {
    selectedCompanyId.value = companies.value[0].id
  }
}

async function loadConversations() {
  error.value = ''
  if (!selectedCompanyId.value) {
    conversations.value = []
    activeConversationId.value = ''
    messages.value = []
    return
  }
  try {
    conversations.value = await api(`/api/im/conversations?company_id=${encodeURIComponent(selectedCompanyId.value)}`)
    if (activeConversationId.value && !conversations.value.find(c => c.id === activeConversationId.value)) {
      activeConversationId.value = ''
      messages.value = []
      members.value = []
      selectedMentionAgentIDs.value = []
    }
    if (!activeConversationId.value && conversations.value.length) {
      activeConversationId.value = conversations.value[0].id
      await loadMessages()
      await loadMembers()
    }
  } catch (e) {
    error.value = e.message
  }
}

async function loadMessages() {
  if (!activeConversationId.value) {
    messages.value = []
    return
  }
  try {
    messages.value = await api(`/api/im/conversations/${activeConversationId.value}/messages`)
  } catch (e) {
    error.value = e.message
  }
}

async function loadMembers() {
  if (!activeConversationId.value) {
    members.value = []
    selectedMentionAgentIDs.value = []
    return
  }
  try {
    members.value = await api(`/api/im/conversations/${activeConversationId.value}/members`)
    selectedMentionAgentIDs.value = selectedMentionAgentIDs.value.filter(id => members.value.some(m => m.member_type === 'agent' && m.member_id === id))
  } catch (e) {
    error.value = e.message
  }
}

async function loadCompanyAgents() {
  if (!selectedCompanyId.value) {
    companyAgents.value = []
    return
  }
  try {
    companyAgents.value = await api(`/api/agents?company_id=${encodeURIComponent(selectedCompanyId.value)}`)
  } catch (e) {
    error.value = e.message
  }
}

async function onCompanyChange() {
  activeConversationId.value = ''
  messages.value = []
  members.value = []
  selectedMentionAgentIDs.value = []
  await loadConversations()
  await loadCompanyAgents()
}

async function selectConversation(id) {
  activeConversationId.value = id
  await loadMessages()
  await loadMembers()
}

async function sendMessage() {
  if (!activeConversationId.value || !draft.value.trim()) return
  try {
    const payload = { body: draft.value.trim() }
    if (selectedMentionAgentIDs.value.length) {
      payload.meta = JSON.stringify({ mentions: { agents: selectedMentionAgentIDs.value } })
    }
    await apiPost(`/api/im/conversations/${activeConversationId.value}/messages`, payload)
    draft.value = ''
    selectedMentionAgentIDs.value = []
    showMentionMenu.value = false
    await loadMessages()
    await loadConversations()
  } catch (e) {
    error.value = e.message
  }
}

async function syncOrg() {
  try {
    if (!selectedCompanyId.value) {
      toast('Select a company first', '#da3633')
      return
    }
    const result = await apiPost('/api/im/sync-org', { company_id: selectedCompanyId.value })
    toast(`Synced: ${result.conversations || 0} convs`)
    await loadConversations()
  } catch (e) {
    error.value = e.message
  }
}

async function bootstrapCEO() {
  try {
    const result = await apiPost('/api/im/bootstrap', {})
    toast(result.created ? 'CEO agent created' : 'CEO agent already exists')
    await loadConversations()
  } catch (e) {
    error.value = e.message
  }
}

async function inviteAgent() {
  if (!activeConversationId.value || !inviteAgentID.value) return
  try {
    await apiPost(`/api/im/conversations/${activeConversationId.value}/members`, {
      member_type: 'agent',
      member_id: inviteAgentID.value,
      role: 'member'
    })
    inviteAgentID.value = ''
    await loadMembers()
    await loadConversations()
    toast('Agent invited')
  } catch (e) {
    error.value = e.message
  }
}

async function kickAgent(agentID) {
  if (!activeConversationId.value) return
  if (!confirm(`Remove agent "${agentID}" from this conversation?`)) return
  try {
    await apiDelete(`/api/im/conversations/${activeConversationId.value}/members/agent/${encodeURIComponent(agentID)}`)
    selectedMentionAgentIDs.value = selectedMentionAgentIDs.value.filter(id => id !== agentID)
    await loadMembers()
    await loadConversations()
    toast('Agent removed')
  } catch (e) {
    error.value = e.message
  }
}

async function deleteConversation(c) {
  if (!confirm(`Delete conversation "${c.title}"? This will remove all members and messages.`)) return
  try {
    await apiDelete(`/api/im/conversations/${c.id}`)
    if (activeConversationId.value === c.id) {
      activeConversationId.value = ''
      messages.value = []
    }
    toast('Conversation deleted')
    await loadConversations()
  } catch (e) {
    error.value = e.message
  }
}

function fmtTime(ts) {
  if (!ts) return '-'
  return new Date(ts).toLocaleString()
}

function agentName(member) {
  return member.agent_display_name || member.member_id
}

function onDraftInput() {
  const match = draft.value.match(/(?:^|\s)@([a-zA-Z0-9_-]*)$/)
  if (!match || !activeConversationId.value) {
    showMentionMenu.value = false
    mentionQuery.value = ''
    return
  }
  mentionQuery.value = match[1] || ''
  mentionSelection.value = 0
  showMentionMenu.value = true
}

function onDraftEnter(e) {
  if (!showMentionMenu.value || !filteredMentionAgents.value.length) return
  e.preventDefault()
  insertMention(filteredMentionAgents.value[mentionSelection.value] || filteredMentionAgents.value[0])
}

function moveMentionSelection(delta) {
  if (!showMentionMenu.value || !filteredMentionAgents.value.length) return
  const next = mentionSelection.value + delta
  const max = filteredMentionAgents.value.length - 1
  mentionSelection.value = Math.max(0, Math.min(max, next))
}

function insertMention(member) {
  if (!member) return
  const match = draft.value.match(/(?:^|\s)@([a-zA-Z0-9_-]*)$/)
  if (!match) return
  draft.value = draft.value.replace(/(?:^|\s)@([a-zA-Z0-9_-]*)$/, (all) => all.replace(/@([a-zA-Z0-9_-]*)$/, `@${agentName(member)} `))
  if (!selectedMentionAgentIDs.value.includes(member.member_id)) {
    selectedMentionAgentIDs.value.push(member.member_id)
  }
  showMentionMenu.value = false
  mentionQuery.value = ''
}

function removeMention(agentID) {
  selectedMentionAgentIDs.value = selectedMentionAgentIDs.value.filter(id => id !== agentID)
}

function decodeJWTSub(token) {
  if (!token) return ''
  const parts = token.split('.')
  if (parts.length < 2) return ''
  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4)
    return JSON.parse(atob(padded)).sub || ''
  } catch {
    return ''
  }
}

onMounted(async () => {
  await loadCompanies()
  await loadCompanyAgents()
  await loadConversations()
})
</script>
