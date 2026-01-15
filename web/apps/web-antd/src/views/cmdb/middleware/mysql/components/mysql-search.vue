<template>
  <a-form layout="inline" class="flex flex-wrap items-end gap-4">
    <a-form-item label="状态">
      <a-select
        v-model:value="formState.status"
        class="w-32"
        placeholder="全部"
        allow-clear
      >
        <a-select-option :value="undefined">全部</a-select-option>
        <a-select-option value="online">在线</a-select-option>
        <a-select-option value="offline">离线</a-select-option>
      </a-select>
    </a-form-item>

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

import type { MySQLFilters } from '../composables/use-mysql-list';

const emit = defineEmits<{
  (event: 'search', filters: MySQLFilters): void;
  (event: 'reset'): void;
}>();

const formState = reactive<MySQLFilters>({
  status: undefined,
});

const handleSearch = () => {
  emit('search', { ...formState });
};

const handleReset = () => {
  formState.status = undefined;
  emit('reset');
};
</script>
