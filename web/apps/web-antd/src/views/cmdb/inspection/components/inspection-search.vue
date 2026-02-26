<script setup lang="ts">
import { Form, FormItem, Select, Button, Space } from 'ant-design-vue';

const props = defineProps<{
  status: string;
  type: string;
}>();

const emit = defineEmits<{
  (e: 'update:status', value: string): void;
  (e: 'update:type', value: string): void;
  (e: 'search'): void;
  (e: 'reset'): void;
}>();

const statusOptions = [
  { value: '', label: '全部状态' },
  { value: 'pending', label: '等待中' },
  { value: 'running', label: '运行中' },
  { value: 'success', label: '成功' },
  { value: 'failed', label: '失败' },
];

const typeOptions = [
  { value: '', label: '全部类型' },
  { value: 'host', label: '主机巡检' },
  { value: 'mysql', label: 'MySQL 巡检' },
  { value: 'redis', label: 'Redis 巡检' },
  { value: 'nginx', label: 'Nginx 巡检' },
  { value: 'tomcat', label: 'Tomcat 巡检' },
  { value: 'elasticsearch', label: 'Elasticsearch 巡检' },
];
</script>

<template>
  <Form layout="inline">
    <FormItem label="状态">
      <Select
        :value="props.status"
        :options="statusOptions"
        style="width: 150px"
        placeholder="全部状态"
        @change="(v) => emit('update:status', (v as string) || '')"
      />
    </FormItem>
    <FormItem label="类型">
      <Select
        :value="props.type"
        :options="typeOptions"
        style="width: 180px"
        placeholder="全部类型"
        @change="(v) => emit('update:type', (v as string) || '')"
      />
    </FormItem>
    <FormItem>
      <Space>
        <Button type="primary" @click="emit('search')">查询</Button>
        <Button @click="emit('reset')">重置</Button>
      </Space>
    </FormItem>
  </Form>
</template>
