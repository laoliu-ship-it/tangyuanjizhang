<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppModal from '@/components/AppModal.vue'
import { rbacApi } from '@/api/rbac'
import { useTenantStore } from '@/store/tenant'
import type { PermissionEntry, TenantRole } from '@/api/types'
import { confirm, toast, toastError } from '@/utils/ui'

const ROLE_LABELS: Record<string, string> = {
  owner: '所有者',
  admin: '管理员',
  finance: '财务',
  editor: '编辑',
  partner: '合伙人',
  member: '成员',
}

const RESOURCE_LABELS: Record<string, string> = {
  transaction: '账目',
  category: '分类',
  merchant: '商户',
  statistics: '统计',
  export: '导出',
  tenant: '租户',
}

const ACTION_LABELS: Record<string, Record<string, string>> = {
  transaction: { read: '查看账目', write: '增删改账目' },
  category: { read: '查看分类', write: '管理分类' },
  merchant: { read: '查看商户', write: '管理商户' },
  statistics: { read: '查看统计' },
  export: { read: '导出数据' },
  tenant: { read: '查看租户', write: '管理租户' },
}

const FALLBACK_PERMISSIONS: PermissionEntry[] = [
  { resource: 'transaction', action: 'read' },
  { resource: 'transaction', action: 'write' },
  { resource: 'category', action: 'read' },
  { resource: 'category', action: 'write' },
  { resource: 'merchant', action: 'read' },
  { resource: 'merchant', action: 'write' },
  { resource: 'statistics', action: 'read' },
  { resource: 'export', action: 'read' },
  { resource: 'tenant', action: 'read' },
  { resource: 'tenant', action: 'write' },
]

const tenantStore = useTenantStore()

const roles = ref<TenantRole[]>([])
const permissions = ref<PermissionEntry[]>([])
const loading = ref(true)
const errorText = ref('')

const editorVisible = ref(false)
const editingRole = ref<TenantRole | null>(null)
const roleName = ref('')
const checkedKeys = ref<Set<string>>(new Set())
const saving = ref(false)
const deletingId = ref(0)

const isEditing = computed(() => !!editingRole.value)

const permissionGroups = computed(() => {
  const map = new Map<string, PermissionEntry[]>()
  permissions.value.forEach((permission) => {
    const list = map.get(permission.resource) ?? []
    list.push(permission)
    map.set(permission.resource, list)
  })
  return Array.from(map.entries()).map(([resource, actions]) => ({
    resource,
    label: RESOURCE_LABELS[resource] ?? resource,
    actions: actions.map((permission) => ({
      key: `${resource}:${permission.action}`,
      label: ACTION_LABELS[resource]?.[permission.action] ?? permission.action,
    })),
  }))
})

function roleLabel(name: string): string {
  return ROLE_LABELS[name] ?? name
}

function permissionLabel(permission: PermissionEntry): string {
  if (permission.resource === '*' && permission.action === '*') return '全部权限'
  return (
    ACTION_LABELS[permission.resource]?.[permission.action] ??
    `${permission.resource}:${permission.action}`
  )
}

function permissionSummary(role: TenantRole): string {
  const list = role.permissions ?? []
  if (!list.length) return '暂无权限'
  if (list.some((permission) => permission.resource === '*' && permission.action === '*')) {
    return '全部权限'
  }
  return Array.from(new Set(list.map(permissionLabel))).join('、')
}

async function load(): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  if (!tenantId) return
  loading.value = true
  errorText.value = ''
  try {
    const [roleList, permissionList] = await Promise.all([
      rbacApi.listRoles(tenantId),
      rbacApi.listPermissions().catch(() => [] as PermissionEntry[]),
    ])
    roles.value = Array.isArray(roleList) ? roleList : []
    permissions.value =
      Array.isArray(permissionList) && permissionList.length
        ? permissionList
        : FALLBACK_PERMISSIONS
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '加载角色权限失败'
  } finally {
    loading.value = false
  }
}

function openCreate(): void {
  editingRole.value = null
  roleName.value = ''
  checkedKeys.value = new Set()
  editorVisible.value = true
}

function openEdit(role: TenantRole): void {
  editingRole.value = role
  roleName.value = role.name
  checkedKeys.value = new Set(
    (role.permissions ?? []).map((permission) => `${permission.resource}:${permission.action}`),
  )
  editorVisible.value = true
}

function togglePermission(key: string): void {
  const next = new Set(checkedKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  checkedKeys.value = next
}

async function save(): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  if (!tenantId || saving.value) return
  if (!isEditing.value && !roleName.value.trim()) {
    toast('请输入角色名称')
    return
  }
  if (checkedKeys.value.size === 0) {
    toast('请至少选择一个权限')
    return
  }
  const payload = Array.from(checkedKeys.value).map((key) => {
    const [resource, action] = key.split(':')
    return { resource, action }
  })
  saving.value = true
  try {
    if (editingRole.value) {
      await rbacApi.updateRole(tenantId, editingRole.value.id, { permissions: payload })
      toast('角色权限已更新', 'success')
    } else {
      await rbacApi.createRole(tenantId, { name: roleName.value.trim(), permissions: payload })
      toast('角色创建成功', 'success')
    }
    editorVisible.value = false
    await load()
  } catch (err) {
    toastError(err, isEditing.value ? '更新失败' : '创建失败')
  } finally {
    saving.value = false
  }
}

async function remove(role: TenantRole): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  if (!tenantId || role.is_system || deletingId.value) return
  const ok = await confirm(`确认删除角色「${roleLabel(role.name)}」？`, '删除角色')
  if (!ok) return
  deletingId.value = role.id
  try {
    await rbacApi.deleteRole(tenantId, role.id)
    toast('角色已删除', 'success')
    await load()
  } catch (err) {
    toastError(err, '删除失败')
  } finally {
    deletingId.value = 0
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <view class="roles">
    <view class="card list-card">
      <view class="list-card__head">
        <text class="list-card__title">共 {{ roles.length }} 个角色</text>
        <view class="btn btn-primary btn-sm" @click="openCreate">创建角色</view>
      </view>

      <view v-if="loading" class="list-state">加载中…</view>
      <view v-else-if="errorText" class="list-state">
        <text>{{ errorText }}</text>
        <text class="list-state__retry" @click="load">重试</text>
      </view>
      <view v-else-if="roles.length === 0" class="list-state">暂无角色</view>

      <view v-else class="role-list">
        <view v-for="role in roles" :key="role.id" class="role-item">
          <view class="role-item__body">
            <view class="role-item__name-line">
              <text class="role-item__name">{{ roleLabel(role.name) }}</text>
              <text v-if="role.is_system" class="role-item__system">系统</text>
            </view>
            <text class="role-item__perms">{{ permissionSummary(role) }}</text>
          </view>
          <view v-if="!role.is_system" class="role-item__actions">
            <text class="role-item__action" @click="openEdit(role)">编辑</text>
            <text class="role-item__action role-item__action--danger" @click="remove(role)">
              {{ deletingId === role.id ? '删除中' : '删除' }}
            </text>
          </view>
        </view>
      </view>
    </view>

    <AppModal
      v-model:visible="editorVisible"
      :title="isEditing ? '编辑角色权限' : '创建角色'"
      position="center"
      :close-on-mask="false"
    >
      <view class="editor">
        <view class="editor__field">
          <text class="editor__label">角色名称</text>
          <input
            v-model="roleName"
            class="input"
            :disabled="isEditing"
            placeholder="例如：店长"
            maxlength="20"
          />
        </view>

        <view class="editor__field">
          <text class="editor__label">权限配置</text>
          <view class="perm-groups">
            <view v-for="group in permissionGroups" :key="group.resource" class="perm-group">
              <text class="perm-group__title">{{ group.label }}</text>
              <view class="perm-group__actions">
                <view
                  v-for="action in group.actions"
                  :key="action.key"
                  class="perm-check"
                  :class="{ 'perm-check--on': checkedKeys.has(action.key) }"
                  @click="togglePermission(action.key)"
                >
                  <text class="perm-check__box">{{ checkedKeys.has(action.key) ? '✓' : '' }}</text>
                  <text class="perm-check__label">{{ action.label }}</text>
                </view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <template #footer>
        <view class="editor__footer">
          <view class="btn editor__btn" @click="editorVisible = false">取消</view>
          <view class="btn btn-primary editor__btn" :class="{ 'btn-disabled': saving }" @click="save">
            {{ saving ? '保存中…' : isEditing ? '保存' : '创建' }}
          </view>
        </view>
      </template>
    </AppModal>
  </view>
</template>

<style lang="scss" scoped>
.roles {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.list-card {
  padding: 0;
  overflow: hidden;
}

.list-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20rpx 28rpx;
  background-color: $c-bg;
  border-bottom: 2rpx solid $c-border;
}

.list-card__title {
  font-size: $fs-sm;
  color: $c-text-2;
}

.list-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16rpx;
  padding: 60rpx 0;
  font-size: $fs-sm;
  color: $c-text-3;
}

.list-state__retry {
  color: $c-primary;
}

.role-list {
  display: flex;
  flex-direction: column;
}

.role-item {
  display: flex;
  align-items: center;
  gap: 16rpx;
  padding: 22rpx 28rpx;
  border-bottom: 2rpx solid $c-border;
}

.role-item:last-child {
  border-bottom: none;
}

.role-item__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.role-item__name-line {
  display: flex;
  align-items: center;
  gap: 10rpx;
}

.role-item__name {
  font-size: $fs-sm;
  font-weight: 500;
  color: $c-text;
}

.role-item__system {
  padding: 2rpx 12rpx;
  border-radius: 999rpx;
  background-color: #f3f4f6;
  color: $c-text-3;
  font-size: 18rpx;
}

.role-item__perms {
  font-size: 20rpx;
  color: $c-text-3;
  line-height: 1.5;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.role-item__actions {
  flex-shrink: 0;
  display: flex;
  gap: 20rpx;
}

.role-item__action {
  font-size: $fs-xs;
  color: $c-primary;
}

.role-item__action--danger {
  color: $c-expense;
}

.editor {
  display: flex;
  flex-direction: column;
  gap: 22rpx;
}

.editor__field {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}

.editor__label {
  font-size: $fs-sm;
  color: $c-text-2;
}

.perm-groups {
  display: flex;
  flex-direction: column;
  gap: 18rpx;
  max-height: 420rpx;
  overflow-y: auto;
}

.perm-group {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.perm-group__title {
  font-size: $fs-xs;
  font-weight: 600;
  color: $c-text;
}

.perm-group__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}

.perm-check {
  display: flex;
  align-items: center;
  gap: 8rpx;
  padding: 10rpx 16rpx;
  border: 2rpx solid $c-border-2;
  border-radius: 999rpx;
  background-color: #fff;
}

.perm-check--on {
  border-color: $c-primary;
  background-color: $c-primary-bg;
}

.perm-check__box {
  width: 28rpx;
  height: 28rpx;
  border-radius: 6rpx;
  border: 2rpx solid $c-border-2;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20rpx;
  color: #fff;
  background-color: #fff;
}

.perm-check--on .perm-check__box {
  border-color: $c-primary;
  background-color: $c-primary;
}

.perm-check__label {
  font-size: $fs-xs;
  color: $c-text-2;
}

.perm-check--on .perm-check__label {
  color: $c-primary;
}

.editor__footer {
  display: flex;
  gap: 16rpx;
}

.editor__btn {
  flex: 1;
  height: 84rpx;
}
</style>
