<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import AppPage from '@/components/AppPage.vue'
import EmptyState from '@/components/EmptyState.vue'
import GeneralTab from '@/components/settings/GeneralTab.vue'
import MembersTab from '@/components/settings/MembersTab.vue'
import RolesTab from '@/components/settings/RolesTab.vue'
import CategoriesTab from '@/components/settings/CategoriesTab.vue'
import LlmTab from '@/components/settings/LlmTab.vue'
import CreditsTab from '@/components/settings/CreditsTab.vue'
import { tenantApi } from '@/api/tenant'
import { TENANT_NAME_SUFFIX, stripTenantSuffix } from '@/config'
import { ensureSessionReady, useAuthGuard } from '@/composables/useAuthGuard'
import { useRole } from '@/composables/useRole'
import { useMetaStore } from '@/store/meta'
import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import { confirm, toast, toastError } from '@/utils/ui'
import { goLogin } from '@/utils/nav'

useAuthGuard()

type TabKey = 'general' | 'members' | 'roles' | 'categories' | 'llm' | 'credits'

const TABS: { key: TabKey; label: string }[] = [
  { key: 'general', label: '通用' },
  { key: 'members', label: '成员' },
  { key: 'roles', label: '角色' },
  { key: 'categories', label: '分类' },
  { key: 'llm', label: 'AI' },
  { key: 'credits', label: '积分' },
]

const tenantStore = useTenantStore()
const metaStore = useMetaStore()
const authStore = useAuthStore()
const { canEdit, load: loadRole } = useRole()

const activeTab = ref<TabKey>('general')
const roleLoaded = ref(false)
const prefix = ref('')
const saving = ref(false)

const currentTenantId = computed(() => tenantStore.currentTenantId)
const currentPrefix = computed(() => stripTenantSuffix(tenantStore.currentTenantName))
const previewName = computed(() => `${prefix.value.trim() || '…'}${TENANT_NAME_SUFFIX}`)
const canSave = computed(
  () => !!prefix.value.trim() && prefix.value.trim() !== currentPrefix.value && !saving.value,
)

watch(
  [currentTenantId, currentPrefix],
  () => {
    prefix.value = currentPrefix.value
  },
  { immediate: true },
)

function switchTab(key: TabKey): void {
  activeTab.value = key
}

async function logout(): Promise<void> {
  const ok = await confirm('确定退出当前账号吗？退出后需要重新登录。', '退出登录')
  if (!ok) return
  authStore.logout()
  tenantStore.reset()
  metaStore.reset()
  goLogin(true)
}

async function saveName(): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  const name = prefix.value.trim()
  if (!tenantId || !name || saving.value) return
  saving.value = true
  try {
    const updated = await tenantApi.update(tenantId, { name: `${name}${TENANT_NAME_SUFFIX}` })
    tenantStore.updateTenantName(tenantId, updated?.name ?? `${name}${TENANT_NAME_SUFFIX}`)
    toast('账本名称已更新', 'success')
  } catch (err) {
    toastError(err, '保存失败')
  } finally {
    saving.value = false
  }
}

onLoad((options) => {
  const tab = options?.tab as TabKey | undefined
  if (tab && TABS.some((item) => item.key === tab)) {
    activeTab.value = tab
  }
})

onShow(() => {
  void (async () => {
    if (!(await ensureSessionReady())) return
    await loadRole()
    roleLoaded.value = true
  })()
})
</script>

<template>
  <AppPage title="账本设置" tabbar="settings">
    <view class="settings">
      <!-- 账本名称 -->
      <view class="card rename">
        <text class="rename__title">账本名称</text>
        <view v-if="canEdit" class="rename__row">
          <input
            v-model="prefix"
            class="input rename__input"
            placeholder="输入账本名称前缀"
            maxlength="20"
          />
          <text class="rename__suffix">{{ TENANT_NAME_SUFFIX }}</text>
          <view
            class="btn btn-primary btn-sm rename__save"
            :class="{ 'btn-disabled': !canSave }"
            @click="saveName"
          >
            {{ saving ? '保存中…' : '保存' }}
          </view>
        </view>
        <text v-else class="rename__readonly">{{ tenantStore.currentTenantName || '未选择账本' }}</text>
        <text v-if="canEdit" class="rename__preview">预览：{{ previewName }}</text>
      </view>

      <!-- Tab 条 -->
      <scroll-view scroll-x class="tabs" :show-scrollbar="false">
        <view class="tabs__inner">
          <view
            v-for="tab in TABS"
            :key="tab.key"
            class="tabs__item"
            :class="{ 'tabs__item--active': activeTab === tab.key }"
            @click="switchTab(tab.key)"
          >
            {{ tab.label }}
          </view>
        </view>
      </scroll-view>

      <view v-if="!roleLoaded" class="state">加载中…</view>

      <EmptyState
        v-else-if="!canEdit && activeTab !== 'credits'"
        icon="🔒"
        text="权限不足"
        hint="仅账本所有者 / 管理员可访问该设置，如需权限请联系管理员"
      />

      <template v-else>
        <GeneralTab v-if="activeTab === 'general'" />
        <MembersTab v-else-if="activeTab === 'members'" />
        <RolesTab v-else-if="activeTab === 'roles'" />
        <CategoriesTab v-else-if="activeTab === 'categories'" />
        <LlmTab v-else-if="activeTab === 'llm'" />
        <CreditsTab v-else-if="activeTab === 'credits'" />
      </template>

      <view class="logout" @click="logout">退出登录</view>
    </view>
  </AppPage>
</template>

<style lang="scss" scoped>
.settings {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.rename {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
}

.rename__title {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.rename__row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.rename__input {
  flex: 1;
  min-width: 0;
}

.rename__suffix {
  flex-shrink: 0;
  font-size: $fs-sm;
  color: $c-text-2;
}

.rename__save {
  flex-shrink: 0;
}

.rename__readonly {
  font-size: $fs;
  color: $c-text;
}

.rename__preview {
  font-size: $fs-xs;
  color: $c-text-3;
}

.tabs {
  width: 100%;
  white-space: nowrap;
}

.tabs__inner {
  display: inline-flex;
  gap: 8rpx;
  padding: 6rpx;
  border-radius: $radius-sm;
  background-color: #eef2f7;
}

.tabs__item {
  flex-shrink: 0;
  padding: 14rpx 28rpx;
  border-radius: 10rpx;
  font-size: $fs-sm;
  color: $c-text-2;
}

.tabs__item--active {
  background-color: #fff;
  color: $c-primary;
  font-weight: 600;
  box-shadow: 0 2rpx 8rpx rgba(17, 24, 39, 0.08);
}

.state {
  padding: 60rpx 0;
  text-align: center;
  font-size: $fs-sm;
  color: $c-text-3;
}

.logout {
  margin-top: 12rpx;
  padding: 26rpx 0;
  text-align: center;
  border-radius: $radius;
  border: 2rpx solid #fecaca;
  background-color: $c-card;
  color: $c-expense;
  font-size: $fs-sm;
}

.logout:active {
  background-color: $c-expense-bg;
}
</style>
