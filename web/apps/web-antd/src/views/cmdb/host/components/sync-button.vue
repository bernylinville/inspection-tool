<template>
  <a-button type="primary" :loading="loading" @click="emit('sync')">
    <template #icon>
      <RotateCw class="h-4 w-4" />
    </template>
    同步主机
  </a-button>
</template>

<script lang="ts" setup>
import { RotateCw } from '@vben/icons';

defineProps<{
  loading: boolean;
}>();

const emit = defineEmits<{
  (event: 'sync'): void;
}>();
</script>
