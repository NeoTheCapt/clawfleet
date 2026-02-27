<template>
  <div class="flex justify-end gap-2 mt-5">
    <button @click="onCancel" class="px-4 py-2 border border-border rounded-md bg-btn text-text text-sm hover:bg-btn-hover transition">
      {{ cancelText }}
    </button>
    <button @click="onSave" class="px-4 py-2 bg-btn-green border border-btn-greenHover rounded-md text-text text-sm hover:bg-btn-greenHover transition">
      {{ saveText }}
    </button>
  </div>
</template>

<script setup>
defineProps({
  onCancel: { type: Function, required: true },
  onSave: { type: Function, required: true },
  saveText: { type: String, default: 'Save' },
  cancelText: { type: String, default: 'Cancel' },
})
</script>
