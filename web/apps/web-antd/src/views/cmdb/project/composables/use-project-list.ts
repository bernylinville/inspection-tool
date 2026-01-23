import { ref, reactive } from 'vue';
import { message } from 'ant-design-vue';
import { listProjectsApi, deleteProjectApi, syncProjectsApi } from '#/api/cmdb/asset';
import type { Project, ProjectStatus } from '#/api/cmdb/types';

export function useProjectList() {
  const loading = ref(false);
  const syncLoading = ref(false);
  const projects = ref<Project[]>([]);
  const pagination = reactive({
    current: 1,
    pageSize: 20,
    total: 0,
  });
  const searchParams = reactive({
    page: 1,
    page_size: 20,
    status: undefined as ProjectStatus | undefined,
  });

  const fetchProjects = async () => {
    loading.value = true;
    try {
      const res = (await listProjectsApi(searchParams)) as { items: Project[]; total: number; page: number; page_size: number } | undefined;
      if (res) {
        projects.value = res.items;
        pagination.total = res.total;
        pagination.current = res.page;
        pagination.pageSize = res.page_size;
      }
    } catch (error) {
      message.error('Failed to load projects');
    } finally {
      loading.value = false;
    }
  };

  const syncProjects = async () => {
    syncLoading.value = true;
    try {
      const res = await syncProjectsApi();
      if (res) {
        message.success(
          `Sync completed: ${res.new_projects} new, ${res.updated_projects} updated, ${res.failed_projects} failed in ${res.duration}`
        );
        await fetchProjects();
      }
    } catch (error) {
      message.error('Failed to sync projects');
    } finally {
      syncLoading.value = false;
    }
  };

  async function deleteProject(id: number) {
    try {
      await deleteProjectApi(id);
      message.success('Project deleted successfully');
      await fetchProjects();
    } catch (error) {
      message.error('Failed to delete project');
    }
  }

  const handlePageChange = (page: number, pageSize: number) => {
    searchParams.page = page;
    searchParams.page_size = pageSize;
    pagination.current = page;
    pagination.pageSize = pageSize;
    fetchProjects();
  };

  const handleSearch = () => {
    searchParams.page = 1;
    pagination.current = 1;
    fetchProjects();
  };

  const handleReset = () => {
    searchParams.status = undefined;
    searchParams.page = 1;
    pagination.current = 1;
    fetchProjects();
  };

  return {
    loading,
    syncLoading,
    projects,
    pagination,
    searchParams,
    fetchProjects,
    syncProjects,
    deleteProject,
    handlePageChange,
    handleSearch,
    handleReset,
  };
}
