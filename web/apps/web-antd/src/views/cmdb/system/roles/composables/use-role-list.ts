import { computed, onMounted, ref } from 'vue';
import { message } from 'ant-design-vue';
import { assignRolePermissionsApi, createRoleApi, deleteRoleApi, getRoleApi, listRolesApi, updateRoleApi } from '#/api/cmdb/role';
import type { AssignPermissionsRequest, Permission, Role, RoleCreateRequest, RoleUpdateRequest } from '#/api/cmdb/types';

export interface RoleFilters {}

export function useRoleList() {
  const roles = ref<Role[]>([]);
  const total = ref(0);
  const page = ref(1);
  const pageSize = ref(20);
  const loading = ref(false);
  const error = ref<Error | null>(null);
  const filters = ref<RoleFilters>({});

  const selectedRole = ref<Role | null>(null);
  const detailVisible = ref(false);

  const formVisible = ref(false);
  const formMode = ref<'create' | 'edit'>('create');
  const formLoading = ref(false);

  const assignPermissionsVisible = ref(false);
  const assignPermissionsLoading = ref(false);
  const assignPermissionsSelectedRole = ref<Role | null>(null);

  const totalPages = computed(() => {
    if (pageSize.value <= 0) {
      return 0;
    }
    return Math.ceil(total.value / pageSize.value);
  });

  const hasRoles = computed(() => roles.value.length > 0);

  const fetchRoles = async () => {
    loading.value = true;
    error.value = null;

    try {
      const result = (await listRolesApi({
        page: page.value,
        page_size: pageSize.value,
        ...filters.value,
      })) as { items: Role[]; total: number };

      roles.value = Array.isArray(result.items) ? result.items : [];
      total.value = Number.isFinite(result.total) ? result.total : 0;
    } catch (err) {
      error.value = err as Error;
      message.error('获取角色列表失败');
    } finally {
      loading.value = false;
    }
  };

  const changePage = async (nextPage: number, nextPageSize?: number) => {
    page.value = nextPage;
    if (typeof nextPageSize === 'number') {
      pageSize.value = nextPageSize;
    }
    await fetchRoles();
  };

  const applyFilters = async (nextFilters: RoleFilters) => {
    filters.value = { ...nextFilters };
    page.value = 1;
    await fetchRoles();
  };

  const resetFilters = async () => {
    filters.value = {};
    page.value = 1;
    await fetchRoles();
  };

  const createRole = async (data: RoleCreateRequest) => {
    formLoading.value = true;
    try {
      await createRoleApi(data);
      message.success('创建角色成功');
      await fetchRoles();
      return true;
    } catch (err) {
      message.error('创建角色失败');
      return false;
    } finally {
      formLoading.value = false;
    }
  };

  const updateRole = async (id: number, data: RoleUpdateRequest) => {
    formLoading.value = true;
    try {
      await updateRoleApi(id, data);
      message.success('更新角色成功');
      await fetchRoles();
      return true;
    } catch (err) {
      message.error('更新角色失败');
      return false;
    } finally {
      formLoading.value = false;
    }
  };

  const deleteRole = async (roleId: number) => {
    try {
      await deleteRoleApi(roleId);
      message.success('删除成功');
      await fetchRoles();
    } catch (err) {
      message.error('删除角色失败');
    }
  };

  const assignPermissions = async (roleId: number, data: AssignPermissionsRequest) => {
    assignPermissionsLoading.value = true;
    try {
      await assignRolePermissionsApi(roleId, data);
      message.success('分配权限成功');
      await fetchRoles();
      return true;
    } catch (err) {
      message.error('分配权限失败');
      return false;
    } finally {
      assignPermissionsLoading.value = false;
    }
  };

  const viewDetail = async (roleId: number) => {
    try {
      const result = (await getRoleApi(roleId)) as Role;
      selectedRole.value = result ?? null;
      detailVisible.value = true;
    } catch (err) {
      message.error('获取角色详情失败');
    }
  };

  const closeDetail = () => {
    selectedRole.value = null;
    detailVisible.value = false;
  };

  const openForm = (mode: 'create' | 'edit', role?: Role) => {
    formMode.value = mode;
    if (mode === 'edit' && role) {
      selectedRole.value = role;
    }
    formVisible.value = true;
  };

  const closeForm = () => {
    formVisible.value = false;
    formMode.value = 'create';
    selectedRole.value = null;
  };

  const openAssignPermissions = (role: Role) => {
    assignPermissionsSelectedRole.value = role;
    assignPermissionsVisible.value = true;
  };

  const closeAssignPermissions = () => {
    assignPermissionsVisible.value = false;
    assignPermissionsSelectedRole.value = null;
  };

  onMounted(() => {
    void fetchRoles();
  });

  return {
    roles,
    total,
    page,
    pageSize,
    loading,
    error,
    filters,
    selectedRole,
    detailVisible,
    formVisible,
    formMode,
    formLoading,
    assignPermissionsVisible,
    assignPermissionsLoading,
    assignPermissionsSelectedRole,
    totalPages,
    hasRoles,
    fetchRoles,
    changePage,
    applyFilters,
    resetFilters,
    createRole,
    updateRole,
    deleteRole,
    assignPermissions,
    viewDetail,
    closeDetail,
    openForm,
    closeForm,
    openAssignPermissions,
    closeAssignPermissions,
  };
}
