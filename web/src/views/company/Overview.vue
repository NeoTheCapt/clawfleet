<template>
  <div>
    <div v-if="company" class="bg-bg-card border border-border rounded-lg p-5 mb-4">
      <div class="flex justify-between items-start">
        <div>
          <h3 class="text-lg font-semibold mb-2">{{ company.name }}</h3>
          <p v-if="company.vision" class="text-text-muted text-sm mb-1"><strong>Vision 愿景:</strong> {{ company.vision }}</p>
          <p v-if="company.mission" class="text-text-muted text-sm mb-1"><strong>Mission 使命:</strong> {{ company.mission }}</p>
          <p v-if="company.description" class="text-text-muted text-sm"><strong>Description:</strong> {{ company.description }}</p>
        </div>
        <button @click="showEdit = true" class="px-2.5 py-1 border border-border rounded-md bg-btn text-text text-xs hover:bg-btn-hover transition">✏️ Edit</button>
      </div>
    </div>
    <div class="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-4">
      <StatCard label="Departments 部门" :value="stats.depts" color="blue" />
      <StatCard label="Positions 岗位" :value="stats.positions" color="yellow" />
      <StatCard label="Assignments 分配" :value="stats.assignments" color="green" />
      <StatCard label="Goals 目标" :value="stats.goals" color="purple" />
    </div>

    <Modal :show="showEdit" @close="showEdit = false">
      <h3 class="text-lg font-semibold mb-4">✏️ Edit Company</h3>
      <FieldLabel>Name 名称</FieldLabel>
      <input v-model="editForm.name" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 focus:outline-none focus:border-accent-blue">
      <FieldLabel>Vision 愿景</FieldLabel>
      <textarea v-model="editForm.vision" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Mission 使命</FieldLabel>
      <textarea v-model="editForm.mission" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Description 描述</FieldLabel>
      <textarea v-model="editForm.description" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 min-h-[80px] resize-y focus:outline-none focus:border-accent-blue"></textarea>
      <FieldLabel>Owner ID (Telegram)</FieldLabel>
      <input v-model="editForm.owner_id" class="w-full px-3 py-2 bg-bg-input border border-border rounded-md text-text text-sm mb-3 font-mono focus:outline-none focus:border-accent-blue">
      <div class="flex justify-end gap-2 mt-5">
        <button @click="showEdit = false" class="px-4 py-2 border border-border rounded-md bg-btn text-text text-sm hover:bg-btn-hover transition">Cancel</button>
        <button @click="saveEdit" class="px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">Save</button>
      </div>
    </Modal>
  </div>
</template>

<script setup>
import { ref, onMounted, inject } from 'vue'
import { api, apiPut } from '../../api'
import StatCard from '../../components/StatCard.vue'
import Modal from '../../components/Modal.vue'
import FieldLabel from '../../components/FieldLabel.vue'

const props = defineProps({ companyId: String })
const toast = inject('toast')
const company = ref(null)
const showEdit = ref(false)
const editForm = ref({ name: '', vision: '', mission: '', description: '', owner_id: '' })
const stats = ref({ depts: 0, positions: 0, assignments: 0, goals: 0 })

onMounted(async () => {
  company.value = await api(`/api/companies/${props.companyId}`)
  editForm.value = { name: company.value?.name || '', vision: company.value?.vision || '', mission: company.value?.mission || '', description: company.value?.description || '', owner_id: company.value?.owner_id || '' }

  try {
    const [chart, assignments, goals] = await Promise.all([
      api(`/api/companies/${props.companyId}/org-chart`).catch(() => null),
      api(`/api/companies/${props.companyId}/assignments`).catch(() => []),
      api(`/api/companies/${props.companyId}/goals`).catch(() => [])
    ])
    if (chart) {
      stats.value.depts = (chart.departments || []).length
      ;(chart.departments || []).forEach(d => stats.value.positions += (d.positions || []).length)
    }
    stats.value.assignments = (assignments || []).length
    stats.value.goals = (goals || []).length
  } catch {}
})

async function saveEdit() {
  if (!editForm.value.name.trim()) { alert('Name required'); return }
  await apiPut(`/api/companies/${props.companyId}`, editForm.value)
  company.value = await api(`/api/companies/${props.companyId}`)
  showEdit.value = false
  toast('Company updated')
}
</script>
