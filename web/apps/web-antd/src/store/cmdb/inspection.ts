import type { InspectionJob, InspectionJobCreateRequest, InspectionJobListParams } from '#/api/cmdb/types';
import { ref } from 'vue';
import { defineStore } from 'pinia';
import { listInspectionJobsApi, createInspectionJobApi, getInspectionJobApi, deleteInspectionJobApi } from '#/api/cmdb';

export const useCmdbInspectionStore = defineStore('cmdbInspection', () => {
  const jobs = ref<InspectionJob[]>([]);
  const jobsTotal = ref(0);
  const jobsLoading = ref(false);
  const currentJob = ref<InspectionJob | null>(null);
  const createLoading = ref(false);

  async function fetchJobs(params?: InspectionJobListParams) {
    jobsLoading.value = true;
    try {
      const data = await listInspectionJobsApi(params);
      jobs.value = data.items;
      jobsTotal.value = data.total;
    } finally {
      jobsLoading.value = false;
    }
  }

  async function createJob(request: InspectionJobCreateRequest) {
    createLoading.value = true;
    try {
      const job = await createInspectionJobApi(request);
      jobs.value.unshift(job);
      return job;
    } finally {
      createLoading.value = false;
    }
  }

  async function fetchJob(id: number) {
    const job = await getInspectionJobApi(id);
    currentJob.value = job;
    return job;
  }

  async function deleteJob(id: number) {
    await deleteInspectionJobApi(id);
    jobs.value = jobs.value.filter(j => j.id !== id);
  }

  function $reset() {
    jobs.value = [];
    jobsTotal.value = 0;
    currentJob.value = null;
  }

  return {
    jobs, jobsTotal, jobsLoading, currentJob, createLoading,
    fetchJobs, createJob, fetchJob, deleteJob, $reset,
  };
});
