<template>
  <div class="text-center py-12 text-text-dim">
    <div class="text-5xl mb-3">{{ emoji }}</div>
    <p>{{ message }}</p>
    <slot />
  </div>
</template>

<script setup>
defineProps({ emoji: String, message: String })
</script>
