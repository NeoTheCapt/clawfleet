<template>
  <div class="bg-bg-card border border-border rounded-lg overflow-x-auto">
    <table class="w-full text-sm border-collapse">
      <thead>
        <tr>
          <th v-for="col in columns" :key="col" class="text-left px-4 py-3 bg-bg-hover text-text-muted font-semibold text-xs uppercase tracking-wide">{{ col }}</th>
        </tr>
      </thead>
      <tbody>
        <slot />
      </tbody>
    </table>
  </div>
</template>

<script setup>
defineProps({ columns: Array })
</script>
