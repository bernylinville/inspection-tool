<template>
  <a-form layout="inline" class="flex flex-wrap items-end gap-4">
    <a-form-item class="ml-auto">
      <div class="flex items-center gap-2">
        <a-button type="primary" @click="handleSearch">搜索</a-button>
        <a-button @click="handleReset">重置</a-button>
      </div>
    </a-form-item>
  </a-form>
</template>

<script lang="ts" setup>
import { reactive } from 'vue';

import type { RoleFilters } from '../composables/use-role-list';

const emit = defineEmits<{
  (event: 'search', filters: RoleFilters): void;
  (event: 'reset'): void;
}>();

const formState = reactive<RoleFilters>({});

const handleSearch = () => {
  emit('search', { ...formState });
};

const handleReset = () => {
  emit('reset');
};
</script>
