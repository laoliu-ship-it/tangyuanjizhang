<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { tenantApi } from '@/api/tenant'
import { rbacApi } from '@/api/rbac'
import { useAuthStore } from '@/store/auth'
import { useMetaStore } from '@/store/meta'
import { useTenantStore } from '@/store/tenant'
import type { TenantMember, TenantRole } from '@/api/types'
import { humanDate } from '@/utils/datetime'
import { confirm, toast, toastError } from '@/utils/ui'

const ROLE_LABELS: Record<string, string> = {
  owner: '所有者',
  admin: '管理员',
  finance: '财务',
  editor: '编辑',
  partner: '合伙人',
  member: '成员',
}

const FALLBACK_ROLES: TenantRole[] = [
  { id: 0, name: 'admin', is_system: true, permissions: [] },
  { id: 0, name: 'finance', is_system: true, permissions: [] },
  { id: 0, name: 'partner', is_system: true, permissions: [] },
]

const authStore = useAuthStore()
const tenantStore = useTenantStore()
const metaStore = useMetaStore()

const members = ref<TenantMember[]>([])
const roles = ref<TenantRole[]>([])
const loading = ref(true)
const errorText = ref('')
const inviteUsername = ref('')
const inviteRoleIndex = ref(0)
const inviting = ref(false)
const removingId = ref(0)
const updatingId = ref(0)

const canManage = computed(() => metaStore.role === 'owner' || metaStore.role === 'admin')
const isOwner = computed(
  () => tenantStore.currentTenant?.owner_id === authStore.userId || metaStore.role === 'owner',
)
const roleOptions = computed(() =>
  (roles.value.length ? roles.value : FALLBACK_ROLES).map((role) => ({
    name: role.name,
    label: ROLE_LABELS[role.name] ?? role.name,
  })),
)

function roleLabel(name: string): string {
  return ROLE_LABELS[name] ?? name
}

function roleTagClass(name: string): string {
  if (name === 'admin') return 'role-tag--admin'
  if (name === 'finance') return 'role-tag--finance'
  if (name === 'partner') return 'role-tag--partner'
  return ''
}

async function load(): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  if (!tenantId) return
  loading.value = true
  errorText.value = ''
  try {
    const [memberList, roleList] = await Promise.all([
      tenantApi.getMembers(tenantId),
      rbacApi.listRoles(tenantId).catch(() => [] as TenantRole[]),
    ])
    members.value = Array.isArray(memberList) ? memberList : []
    roles.value = Array.isArray(roleList) ? roleList : []
    const financeIndex = roleOptions.value.findIndex((role) => role.name === 'finance')
    inviteRoleIndex.value = financeIndex >= 0 ? financeIndex : 0
    await metaStore.ensureRole()
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '加载成员失败'
  } finally {
    loading.value = false
  }
}

function onInviteRoleChange(e: Event): void {
  const detail = (e as Event & { detail?: { value?: string | number } }).detail
  inviteRoleIndex.value = Number(detail?.value ?? 0)
}

async function invite(): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  const username = inviteUsername.value.trim()
  if (!tenantId) return
  if (!username) {
    toast('请输入用户名')
    return
  }
  if (inviting.value) return
  inviting.value = true
  try {
    const role = roleOptions.value[inviteRoleIndex.value]?.name ?? 'finance'
    await tenantApi.addMember(tenantId, { username, role })
    inviteUsername.value = ''
    toast('邀请成功', 'success')
    await load()
  } catch (err) {
    toastError(err, '邀请失败')
  } finally {
    inviting.value = false
  }
}

function canChangeRole(member: TenantMember): boolean {
  return (
    isOwner.value &&
    member.user_id !== authStore.userId &&
    member.user_id !== tenantStore.currentTenant?.owner_id
  )
}

async function changeRole(member: TenantMember, index: number): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  const target = roleOptions.value[index]
  if (!tenantId || !target || updatingId.value) return
  updatingId.value = member.user_id
  try {
    await tenantApi.updateMemberRole(tenantId, member.user_id, target.name)
    member.role = target.name
    toast('角色已更新', 'success')
  } catch (err) {
    toastError(err, '修改角色失败')
  } finally {
    updatingId.value = 0
  }
}

function onRolePick(member: TenantMember, e: Event): void {
  const detail = (e as Event & { detail?: { value?: string | number } }).detail
  const index = Number(detail?.value ?? 0)
  if (roleOptions.value[index]?.name === member.role) return
  void changeRole(member, index)
}

async function remove(member: TenantMember): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  if (!tenantId || removingId.value) return
  const ok = await confirm(`确认将「${member.username}」移出本账本？`, '移除成员')
  if (!ok) return
  removingId.value = member.user_id
  try {
    await tenantApi.removeMember(tenantId, member.user_id)
    toast('已移除', 'success')
    await load()
  } catch (err) {
    toastError(err, '移除失败')
  } finally {
    removingId.value = 0
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <view class="members">
    <view v-if="canManage" class="card invite">
      <text class="invite__title">邀请成员</text>
      <view class="invite__row">
        <input
          v-model="inviteUsername"
          class="input invite__input"
          placeholder="输入用户名"
          confirm-type="done"
          @confirm="invite"
        />
        <picker
          mode="selector"
          :range="roleOptions"
          range-key="label"
          :value="inviteRoleIndex"
          @change="onInviteRoleChange"
        >
          <view class="invite__picker">
            {{ roleOptions[inviteRoleIndex]?.label ?? '选择角色' }}
          </view>
        </picker>
        <view
          class="btn btn-primary btn-sm invite__btn"
          :class="{ 'btn-disabled': inviting }"
          @click="invite"
        >
          {{ inviting ? '邀请中…' : '邀请' }}
        </view>
      </view>
    </view>

    <view class="card list-card">
      <view class="list-card__head">
        <text class="list-card__title">共 {{ members.length }} 位成员</text>
      </view>

      <view v-if="loading" class="list-state">加载中…</view>
      <view v-else-if="errorText" class="list-state">
        <text>{{ errorText }}</text>
        <text class="list-state__retry" @click="load">重试</text>
      </view>
      <view v-else-if="members.length === 0" class="list-state">暂无成员</view>

      <view v-else class="member-list">
        <view v-for="member in members" :key="member.user_id" class="member">
          <view class="member__avatar">{{ member.username?.charAt(0)?.toUpperCase() }}</view>
          <view class="member__body">
            <view class="member__name-line">
              <text class="member__name">{{ member.username }}</text>
              <text v-if="member.user_id === authStore.userId" class="member__badge">（我）</text>
              <text
                v-if="member.user_id === tenantStore.currentTenant?.owner_id"
                class="member__owner"
              >
                所有者
              </text>
            </view>
            <text class="member__joined">加入于 {{ humanDate(member.joined_at) }}</text>
          </view>

          <picker
            v-if="canChangeRole(member)"
            mode="selector"
            :range="roleOptions"
            range-key="label"
            :value="Math.max(0, roleOptions.findIndex((role) => role.name === member.role))"
            @change="onRolePick(member, $event)"
          >
            <view class="role-tag role-tag--editable" :class="roleTagClass(member.role)">
              {{ roleLabel(member.role) }} ⌄
            </view>
          </picker>
          <text v-else class="role-tag" :class="roleTagClass(member.role)">
            {{ roleLabel(member.role) }}
          </text>

          <text
            v-if="
              canManage &&
              member.user_id !== authStore.userId &&
              member.user_id !== tenantStore.currentTenant?.owner_id
            "
            class="member__remove"
            @click="remove(member)"
          >
            {{ removingId === member.user_id ? '移除中' : '移除' }}
          </text>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.members {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.invite {
  display: flex;
  flex-direction: column;
  gap: 18rpx;
}

.invite__title {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.invite__row {
  display: flex;
  align-items: center;
  gap: 12rpx;
  flex-wrap: wrap;
}

.invite__input {
  flex: 1;
  min-width: 240rpx;
}

.invite__picker {
  height: 80rpx;
  min-width: 160rpx;
  padding: 0 20rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2rpx solid $c-border-2;
  border-radius: $radius-sm;
  background-color: #fff;
  font-size: $fs-sm;
  color: $c-text;
}

.invite__btn {
  min-width: 140rpx;
}

.list-card {
  padding: 0;
  overflow: hidden;
}

.list-card__head {
  padding: 22rpx 28rpx;
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

.member-list {
  display: flex;
  flex-direction: column;
}

.member {
  display: flex;
  align-items: center;
  gap: 16rpx;
  padding: 24rpx 28rpx;
  border-bottom: 2rpx solid $c-border;
}

.member:last-child {
  border-bottom: none;
}

.member__avatar {
  width: 72rpx;
  height: 72rpx;
  flex-shrink: 0;
  border-radius: 50%;
  background-color: $c-primary-bg;
  color: $c-primary;
  font-size: $fs;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}

.member__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.member__name-line {
  display: flex;
  align-items: center;
  gap: 6rpx;
}

.member__name {
  font-size: $fs-sm;
  font-weight: 500;
  color: $c-text;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.member__badge {
  font-size: $fs-xs;
  color: $c-text-3;
}

.member__owner {
  font-size: 20rpx;
  color: $c-warn;
}

.member__joined {
  font-size: 20rpx;
  color: $c-text-3;
}

.role-tag {
  flex-shrink: 0;
  padding: 6rpx 16rpx;
  border-radius: 999rpx;
  font-size: $fs-xs;
  color: $c-text-2;
  background-color: #f3f4f6;
}

.role-tag--admin {
  background-color: $c-primary-bg;
  color: $c-primary;
}

.role-tag--finance {
  background-color: $c-purple-bg;
  color: $c-purple;
}

.role-tag--partner {
  background-color: $c-warn-bg;
  color: $c-warn;
}

.role-tag--editable {
  border: 2rpx solid currentColor;
  background-color: transparent;
}

.member__remove {
  flex-shrink: 0;
  font-size: $fs-xs;
  color: $c-expense;
  padding: 6rpx;
}
</style>
