<script setup lang="ts">
import { computed } from 'vue'
import { stripTenantSuffix } from '@/config'
import { useMetaStore } from '@/store/meta'
import { useTenantStore } from '@/store/tenant'
import { routes } from '@/utils/nav'
import { toast } from '@/utils/ui'

withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })

const tenantStore = useTenantStore()
const metaStore = useMetaStore()

const currentName = computed(() => stripTenantSuffix(tenantStore.currentTenantName))

function currentPageUrl(): string {
  try {
    const pages = getCurrentPages()
    const page = pages[pages.length - 1] as unknown as {
      route?: string
      options?: Record<string, string>
      $page?: { fullPath?: string }
    }
    if (page?.route) {
      let url = `/${page.route}`
      const options = page.options ?? {}
      const query = Object.keys(options)
        .filter((key) => options[key] !== undefined && options[key] !== '')
        .map((key) => `${key}=${encodeURIComponent(String(options[key]))}`)
        .join('&')
      if (query) url += `?${query}`
      return url
    }
    if (page?.$page?.fullPath) return page.$page.fullPath
  } catch (e) {
    // 忽略
  }
  return routes.home
}

function onSwitch(): void {
  const tenants = tenantStore.tenants
  if (!tenants.length) return
  uni.showActionSheet({
    itemList: tenants.map((t) => stripTenantSuffix(t.name) || t.name),
    success: (res) => {
      const target = tenants[res.tapIndex]
      if (!target || target.id === tenantStore.currentTenantId) return
      tenantStore.switchTenant(target.id)
      metaStore.reset()
      const url = currentPageUrl()
      uni.reLaunch({
        url,
        fail: () => uni.reLaunch({ url: routes.home }),
      })
    },
    fail: () => {
      // 用户取消
    },
  })
}
</script>

<template>
  <view v-if="tenantStore.tenants.length" class="tenant" :class="{ 'tenant--compact': compact }" @click="onSwitch">
    <text class="tenant__name">{{ currentName || '选择账本' }}</text>
    <text class="tenant__arrow">⌄</text>
  </view>
</template>

<style lang="scss" scoped>
.tenant {
  display: flex;
  align-items: center;
  gap: 4rpx;
  max-width: 240rpx;
  padding: 8rpx 14rpx;
  border-radius: 999rpx;
  background-color: $c-primary-bg;
  color: $c-primary;
}

.tenant--compact {
  max-width: 190rpx;
  padding: 6rpx 12rpx;
}

.tenant__name {
  font-size: $fs-xs;
  font-weight: 500;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.tenant__arrow {
  font-size: 22rpx;
  line-height: 1;
}

.tenant:active {
  opacity: 0.75;
}
</style>
