import { computed, onMounted, ref } from 'vue';
import { message } from 'ant-design-vue';
import { assignUserRolesApi, createUserApi, deleteUserApi, getUserApi, listUsersApi, updateUserApi } from '@/api/cmdb/user';
import type { AssignRolesRequest, User, UserCreateRequest, UserUpdateRequest } from '@/api/cmdb/types';

export interface UserFilters {
  status?: 'active' | 'inactive';
}

export function useUserList() {
  const users = ref<User[]>([]);
  const total = ref(0);
  const page = ref(1);
  const pageSize = ref(20);
  const loading = ref(false);
  const error = ref<Error | null>(null);
  const filters = ref<UserFilters>({});

  const selectedUser = ref<User | null>(null);
  const detailVisible = ref(false);

  const formVisible = ref(false);
  const formMode = ref<'create' | 'edit'>('create');
  const formLoading = ref(false);

  const assignRolesVisible = ref(false);
  const assignRolesLoading = ref(false);
  const assignRolesSelectedUser = ref<User | null>(null);

  const totalPages = computed(() => {
    if (pageSize.value <= 0) {
      return 0;
    }
    return Math.ceil(total.value / pageSize.value);
  });

  const hasUsers = computed(() => users.value.length > 0);

  const fetchUsers = async () => {
    loading.value = true;
    error.value = null;

    try {
      const result = (await listUsersApi({
        page: page.value,
        page_size: pageSize.value,
        ...filters.value,
      })) as { items: User[]; total: number };

      users.value = Array.isArray(result.items) ? result.items : [];
      total.value = Number.isFinite(result.total) ? result.total : 0;
    } catch (err) {
      error.value = err as Error;
      message.error('获取用户列表失败');
    } finally {
      loading.value = false;
    }
  };

  const changePage = async (nextPage: number, nextPageSize?: number) => {
    page.value = nextPage;
    if (typeof nextPageSize === 'number') {
      pageSize.value = nextPageSize;
    }
    await fetchUsers();
  };

  const applyFilters = async (nextFilters: UserFilters) => {
    filters.value = { ...nextFilters };
    page.value = 1;
    await fetchUsers();
  };

  const resetFilters = async () => {
    filters.value = {};
    page.value = 1;
    await fetchUsers();
  };

  const createUser = async (data: UserCreateRequest) => {
    formLoading.value = true;
    try {
      await createUserApi(data);
      message.success('创建用户成功');
      await fetchUsers();
      return true;
    } catch (err) {
      message.error('创建用户失败');
      return false;
    } finally {
      formLoading.value = false;
    }
  };

  const updateUser = async (id: number, data: UserUpdateRequest) => {
    formLoading.value = true;
    try {
      await updateUserApi(id, data);
      message.success('更新用户成功');
      await fetchUsers();
      return true;
    } catch (err) {
      message.error('更新用户失败');
      return false;
    } finally {
      formLoading.value = false;
    }
  };

  const deleteUser = async (userId: number) => {
    try {
      await deleteUserApi(userId);
      message.success('删除成功');
      await fetchUsers();
    } catch (err) {
      message.error('删除用户失败');
    }
  };

  const assignRoles = async (userId: number, data: AssignRolesRequest) => {
    assignRolesLoading.value = true;
    try {
      await assignUserRolesApi(userId, data);
      message.success('分配角色成功');
      await fetchUsers();
      return true;
    } catch (err) {
      message.error('分配角色失败');
      return false;
    } finally {
      assignRolesLoading.value = false;
    }
  };

  const viewDetail = async (userId: number) => {
    try {
      const result = (await getUserApi(userId)) as User;
      selectedUser.value = result ?? null;
      detailVisible.value = true;
    } catch (err) {
      message.error('获取用户详情失败');
    }
  };

  const closeDetail = () => {
    selectedUser.value = null;
    detailVisible.value = false;
  };

  const openForm = (mode: 'create' | 'edit', user?: User) => {
    formMode.value = mode;
    if (mode === 'edit' && user) {
      selectedUser.value = user;
    }
    formVisible.value = true;
  };

  const closeForm = () => {
    formVisible.value = false;
    formMode.value = 'create';
    selectedUser.value = null;
  };

  const openAssignRoles = (user: User) => {
    assignRolesSelectedUser.value = user;
    assignRolesVisible.value = true;
  };

  const closeAssignRoles = () => {
    assignRolesVisible.value = false;
    assignRolesSelectedUser.value = null;
  };

  onMounted(() => {
    void fetchUsers();
  });

  return {
    users,
    total,
    page,
    pageSize,
    loading,
    error,
    filters,
    selectedUser,
    detailVisible,
    formVisible,
    formMode,
    formLoading,
    assignRolesVisible,
    assignRolesLoading,
    assignRolesSelectedUser,
    totalPages,
    hasUsers,
    fetchUsers,
    changePage,
    applyFilters,
    resetFilters,
    createUser,
    updateUser,
    deleteUser,
    assignRoles,
    viewDetail,
    closeDetail,
    openForm,
    closeForm,
    openAssignRoles,
    closeAssignRoles,
  };
}
