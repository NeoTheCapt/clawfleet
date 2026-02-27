<template>
  <div class="bg-bg-card border border-border rounded-lg p-5">
    <div class="text-xs text-text-muted uppercase tracking-wide">{{ label }}</div>
    <div class="text-[28px] font-bold mt-1" :class="colorClass">{{ value }}</div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({ label: String, value: [String, Number], color: { type: String, default: 'blue' } })
const colorClass = computed(() => ({
  green: 'text-accent-green', blue: 'text-accent-blue', yellow: 'text-accent-yellow',
  red: 'text-accent-red', purple: 'text-accent-purple'
}[props.color] || 'text-accent-blue'))
</script>
