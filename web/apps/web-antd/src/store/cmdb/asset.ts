import type { Application, Host, Project } from '#/api/cmdb/types';
import { ref } from 'vue';
import { defineStore } from 'pinia';
import {
  listProjectsApi,
  listApplicationsApi,
  listHostsApi,
  syncHostsApi,
} from '#/api/cmdb';

export const useCmdbAssetStore = defineStore('cmdbAsset', () => {
  const projects = ref<Project[]>([]);
  const projectsTotal = ref(0);
  const projectsLoading = ref(false);

  const applications = ref<Application[]>([]);
  const applicationsTotal = ref(0);
  const applicationsLoading = ref(false);

  const hosts = ref<Host[]>([]);
  const hostsTotal = ref(0);
  const hostsLoading = ref(false);
  const syncLoading = ref(false);

  async function fetchProjects(params?: { page?: number; page_size?: number; status?: string }) {
    projectsLoading.value = true;
    try {
      const data = await listProjectsApi(params);
      projects.value = data.items;
      projectsTotal.value = data.total;
    } finally {
      projectsLoading.value = false;
    }
  }

  async function fetchApplications(params?: { page?: number; page_size?: number; project_id?: number }) {
    applicationsLoading.value = true;
    try {
      const data = await listApplicationsApi(params);
      applications.value = data.items;
      applicationsTotal.value = data.total;
    } finally {
      applicationsLoading.value = false;
    }
  }

  async function fetchHosts(params?: { page?: number; page_size?: number; business_group?: string }) {
    hostsLoading.value = true;
    try {
      const data = await listHostsApi(params);
      hosts.value = data.items;
      hostsTotal.value = data.total;
    } finally {
      hostsLoading.value = false;
    }
  }

  async function syncHosts() {
    syncLoading.value = true;
    try {
      return await syncHostsApi();
    } finally {
      syncLoading.value = false;
    }
  }

  function $reset() {
    projects.value = [];
    projectsTotal.value = 0;
    applications.value = [];
    applicationsTotal.value = 0;
    hosts.value = [];
    hostsTotal.value = 0;
  }

  return {
    projects, projectsTotal, projectsLoading,
    applications, applicationsTotal, applicationsLoading,
    hosts, hostsTotal, hostsLoading, syncLoading,
    fetchProjects, fetchApplications, fetchHosts, syncHosts,
    $reset,
  };
});
