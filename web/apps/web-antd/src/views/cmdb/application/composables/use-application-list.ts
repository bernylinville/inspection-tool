import { ref, reactive } from 'vue';
import { message } from 'ant-design-vue';
import { listApplicationsApi, deleteApplicationApi } from '#/api/cmdb/asset';
import type { Application, ApplicationStatus } from '#/api/cmdb/types';

export function useApplicationList() {
  const loading = ref(false);
  const applications = ref<Application[]>([]);
  const pagination = reactive({
    current: 1,
    pageSize: 20,
    total: 0,
  });
  const searchParams = reactive({
    page: 1,
    page_size: 20,
    name: undefined as string | undefined,
    project_id: undefined as number | undefined,
    status: undefined as ApplicationStatus | undefined,
  });

  const fetchApplications = async () => {
    loading.value = true;
    try {
      const res = (await listApplicationsApi(searchParams)) as { items: Application[]; total: number; page: number; page_size: number } | undefined;
      if (res) {
        applications.value = res.items;
        pagination.total = res.total;
        pagination.current = res.page;
        pagination.pageSize = res.page_size;
      }
    } catch (error) {
      message.error('加载应用列表失败');
    } finally {
      loading.value = false;
    }
  };

  async function deleteApplication(id: number) {
    try {
      await deleteApplicationApi(id);
      message.success('应用删除成功');
      await fetchApplications();
    } catch (error) {
      message.error('应用删除失败');
    }
  }

  const handlePageChange = (page: number, pageSize: number) => {
    searchParams.page = page;
    searchParams.page_size = pageSize;
    pagination.current = page;
    pagination.pageSize = pageSize;
    fetchApplications();
  };

  const handleSearch = () => {
    searchParams.page = 1;
    pagination.current = 1;
    fetchApplications();
  };

  const handleReset = () => {
    searchParams.name = undefined;
    searchParams.project_id = undefined;
    searchParams.status = undefined;
    searchParams.page = 1;
    pagination.current = 1;
    fetchApplications();
  };

  return {
    loading,
    applications,
    pagination,
    searchParams,
    fetchApplications,
    deleteApplication,
    handlePageChange,
    handleSearch,
    handleReset,
  };
}
