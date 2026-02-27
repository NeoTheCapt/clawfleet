<template>
  <Teleport to="body">
    <div v-if="show" class="fixed inset-0 bg-black/60 z-[100] flex justify-center items-center" @click.self="$emit('close')">
      <div class="bg-bg-card border border-border rounded-xl p-6 max-h-[80vh] overflow-y-auto" :style="{ width: width + 'px' }">
        <slot />
      </div>
    </div>
  </Teleport>
</template>

<script setup>
defineProps({ show: Boolean, width: { type: Number, default: 500 } })
defineEmits(['close'])
</script>
