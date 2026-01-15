import { ref, reactive } from 'vue';
import { listInspectionJobsApi, createInspectionJobApi, deleteInspectionJobApi, getInspectionJobApi } from '@/api/cmdb/inspection';
import type { InspectionJob, CreateInspectionJobRequest } from '@/api/cmdb/types';

export function useInspectionList() {
  const jobs = ref<InspectionJob[]>([]);
  const total = ref(0);
  const page = ref(1);
  const pageSize = ref(20);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const filters = reactive({
    status: '' as string,
    type: '' as string,
  });

  // Detail state
  const selectedJob = ref<InspectionJob | null>(null);
  const detailVisible = ref(false);

  // Create state
  const createLoading = ref(false);

  async function fetchJobs() {
    loading.value = true;
    error.value = null;
    try {
      const params: Record<string, any> = {
        page: page.value,
        page_size: pageSize.value,
      };
      if (filters.status) params.status = filters.status;
      if (filters.type) params.type = filters.type;

      const res = await listInspectionJobsApi(params);
      jobs.value = res.items || [];
      total.value = res.total || 0;
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch inspection jobs';
    } finally {
      loading.value = false;
    }
  }

  function changePage(newPage: number, newPageSize: number) {
    page.value = newPage;
    pageSize.value = newPageSize;
    fetchJobs();
  }

  function applyFilters() {
    page.value = 1;
    fetchJobs();
  }

  function resetFilters() {
    filters.status = '';
    filters.type = '';
    page.value = 1;
    fetchJobs();
  }

  async function createJob(request: CreateInspectionJobRequest) {
    createLoading.value = true;
    try {
      await createInspectionJobApi(request);
      await fetchJobs();
      return true;
    } catch (e: any) {
      error.value = e.message || 'Failed to create inspection job';
      return false;
    } finally {
      createLoading.value = false;
    }
  }

  async function deleteJob(id: number) {
    try {
      await deleteInspectionJobApi(id);
      await fetchJobs();
      return true;
    } catch (e: any) {
      error.value = e.message || 'Failed to delete inspection job';
      return false;
    }
  }

  async function viewDetail(id: number) {
    try {
      const job = await getInspectionJobApi(id);
      selectedJob.value = job;
      detailVisible.value = true;
    } catch (e: any) {
      error.value = e.message || 'Failed to get job detail';
    }
  }

  function closeDetail() {
    selectedJob.value = null;
    detailVisible.value = false;
  }

  return {
    jobs,
    total,
    page,
    pageSize,
    loading,
    error,
    filters,
    selectedJob,
    detailVisible,
    createLoading,
    fetchJobs,
    changePage,
    applyFilters,
    resetFilters,
    createJob,
    deleteJob,
    viewDetail,
    closeDetail,
  };
}
