<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { tenantApi } from '@/api/tenant'
import { useTenantStore } from '@/store/tenant'
import { toast, toastError } from '@/utils/ui'

const tenantStore = useTenantStore()

const requireExpenseImage = ref(true)
const loading = ref(true)
const saving = ref(false)

async function load(): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  if (!tenantId) return
  loading.value = true
  try {
    const settings = await tenantApi.getSettings(tenantId)
    requireExpenseImage.value = settings?.require_expense_image ?? true
  } catch (err) {
    toastError(err, '加载配置失败')
  } finally {
    loading.value = false
  }
}

function toggle(): void {
  requireExpenseImage.value = !requireExpenseImage.value
}

async function save(): Promise<void> {
  const tenantId = tenantStore.currentTenantId
  if (!tenantId || saving.value) return
  saving.value = true
  try {
    const result = await tenantApi.updateSettings(tenantId, {
      require_expense_image: requireExpenseImage.value,
    })
    requireExpenseImage.value = result?.require_expense_image ?? requireExpenseImage.value
    toast('保存成功', 'success')
  } catch (err) {
    toastError(err, '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void load()
})

watch(
  () => tenantStore.currentTenantId,
  () => {
    void load()
  },
)
</script>

<template>
  <view class="general">
    <view v-if="loading" class="general__loading">加载中…</view>
    <template v-else>
      <view class="card general__card">
        <view class="general__row">
          <view class="general__text">
            <text class="general__title">支出记录必须上传图片</text>
            <text class="general__desc">开启后，新增支出时若未上传凭证图片将无法保存</text>
          </view>
          <switch
            :checked="requireExpenseImage"
            color="#2563eb"
            style="transform: scale(0.8)"
            @change="toggle"
          />
        </view>
      </view>

      <view class="btn btn-primary btn-block" :class="{ 'btn-disabled': saving }" @click="save">
        {{ saving ? '保存中…' : '保存配置' }}
      </view>
    </template>
  </view>
</template>

<style lang="scss" scoped>
.general {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.general__loading {
  padding: 60rpx 0;
  text-align: center;
  font-size: $fs-sm;
  color: $c-text-3;
}

.general__card {
  padding: 28rpx;
}

.general__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
}

.general__text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.general__title {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.general__desc {
  font-size: $fs-xs;
  color: $c-text-3;
  line-height: 1.5;
}
</style>
