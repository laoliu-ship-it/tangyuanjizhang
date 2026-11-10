<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { authApi } from '@/api/auth'
import { STORAGE_KEYS } from '@/config'
import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import { getJSON, remove, setJSON } from '@/utils/storage'
import { goHome, routes } from '@/utils/nav'
import { applyAuthResult } from '@/utils/session'
import { toastError } from '@/utils/ui'

interface Remembered {
  email: string
  password: string
}

const authStore = useAuthStore()
const tenantStore = useTenantStore()

const remembered = getJSON<Remembered | null>(STORAGE_KEYS.remember, null)
const email = ref(remembered?.email ?? '')
const password = ref(remembered?.password ?? '')
const remember = ref(!!remembered)
const loading = ref(false)
const demoEnabled = ref(false)
const demoLoading = ref(false)

onMounted(async () => {
  try {
    const cfg = await authApi.demoConfig()
    demoEnabled.value = !!cfg?.enabled
  } catch (e) {
    demoEnabled.value = false
  }
})

onShow(() => {
  if (authStore.token && tenantStore.currentTenantId) {
    goHome()
  }
})

function goRegister(): void {
  uni.navigateTo({ url: routes.register })
}

function onRememberChange(e: Event): void {
  const detail = (e as Event & { detail?: { value?: boolean } }).detail
  remember.value = !!detail?.value
}

async function submit(): Promise<void> {
  if (loading.value) return
  if (!email.value.trim()) {
    toastError(new Error('请输入邮箱'))
    return
  }
  if (!password.value) {
    toastError(new Error('请输入密码'))
    return
  }
  loading.value = true
  try {
    const result = await authApi.login({
      email: email.value.trim(),
      password: password.value,
    })
    if (remember.value) {
      setJSON(STORAGE_KEYS.remember, { email: email.value.trim(), password: password.value })
    } else {
      remove(STORAGE_KEYS.remember)
    }
    await applyAuthResult(result)
  } catch (err) {
    toastError(err, '登录失败，请检查邮箱和密码')
  } finally {
    loading.value = false
  }
}

/**
 * 小程序微信一键登录：
 * - 已绑定 → 直接登录
 * - 未绑定 → 带上短期 bind_token 进入「关联已有邮箱 / 创建新账号」页
 */
async function demoLogin(): Promise<void> {
  if (demoLoading.value) return
  demoLoading.value = true
  try {
    const result = await authApi.demoLogin()
    await applyAuthResult(result)
  } catch (err) {
    toastError(err, '体验登录失败')
  } finally {
    demoLoading.value = false
  }
}

function wechatLogin(): void {
  if (loading.value) return
  loading.value = true
  uni.login({
    provider: 'weixin',
    success: async (res) => {
      const code = res.code
      if (!code) {
        loading.value = false
        toastError(new Error('微信登录凭证获取失败，请重试'))
        return
      }
      try {
        const result = await authApi.wechatLogin(code)
        if (result.bound && result.auth) {
          await applyAuthResult(result.auth)
          return
        }
        if (!result.bind_token) {
          throw new Error('微信登录失败，请重试')
        }
        // 通过本地缓存传递短期凭证，避免出现在页面 URL / 日志中
        uni.setStorageSync(STORAGE_KEYS.wechatBind, result.bind_token)
        uni.navigateTo({ url: routes.wechatBind })
      } catch (err) {
        toastError(err, '微信登录失败，请重试')
      } finally {
        loading.value = false
      }
    },
    fail: () => {
      loading.value = false
      toastError(new Error('微信登录失败，请重试'))
    },
  })
}
</script>

<template>
  <scroll-view scroll-y class="auth-page">
    <view class="auth-inner">
      <view class="auth-card">
      <view class="auth-head">
        <text class="auth-title">汤圆记账</text>
        <text class="auth-subtitle">专业记账管理工具</text>
      </view>

      <view class="field">
        <text class="field__label">邮箱</text>
        <input
          v-model="email"
          class="input"
          type="text"
          placeholder="请输入邮箱"
          confirm-type="next"
        />
      </view>

      <view class="field">
        <text class="field__label">密码</text>
        <input
          v-model="password"
          class="input"
          password
          placeholder="请输入密码"
          confirm-type="done"
          @confirm="submit"
        />
      </view>

      <view class="remember">
        <text class="remember__label">记住密码</text>
        <switch
          :checked="remember"
          color="#2563eb"
          style="transform: scale(0.75)"
          @change="onRememberChange"
        />
      </view>

      <view class="btn btn-primary btn-block auth-submit" :class="{ 'btn-disabled': loading }" @click="submit">
        {{ loading ? '登录中…' : '登录' }}
      </view>

      <!-- #ifdef MP-WEIXIN -->
      <view class="divider">
        <text class="divider__text">或</text>
      </view>
      <view
        class="btn wechat-btn btn-block"
        :class="{ 'btn-disabled': loading }"
        @click="wechatLogin"
      >
        微信一键登录
      </view>
      <text class="wechat-hint">首次使用微信登录，可关联之前的邮箱账号</text>
      <!-- #endif -->

      <view v-if="demoEnabled" class="demo-entry" :class="{ 'btn-disabled': demoLoading }" @click="demoLogin">
        <text class="demo-entry__title">{{ demoLoading ? '进入体验中…' : '一键体验（只读演示）' }}</text>
        <text class="demo-entry__hint">使用演示账本浏览示例数据，不可购买或修改真实账本</text>
      </view>

      <view class="auth-switch">
        <text class="auth-switch__text">还没有账号？</text>
        <text class="auth-switch__link" @click="goRegister">立即注册</text>
      </view>
      </view>
    </view>
  </scroll-view>
</template>

<style lang="scss" scoped>
.auth-page {
  height: 100vh;
  background: linear-gradient(135deg, #eff6ff, #e0e7ff);
}

.auth-inner {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 120rpx 40rpx;
  padding-top: calc(120rpx + env(safe-area-inset-top));
  box-sizing: border-box;
}

.auth-card {
  width: 100%;
  max-width: 640rpx;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 26rpx;
  padding: 56rpx 44rpx;
  border-radius: $radius-lg;
  background-color: #fff;
  box-shadow: 0 20rpx 60rpx rgba(30, 64, 175, 0.12);
}

.auth-head {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10rpx;
  margin-bottom: 12rpx;
}

.auth-title {
  font-size: 56rpx;
  font-weight: 700;
  color: $c-primary;
}

.auth-subtitle {
  font-size: $fs-sm;
  color: $c-text-2;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 10rpx;
}

.field__label {
  font-size: $fs-sm;
  font-weight: 500;
  color: $c-text-2;
}

.remember {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4rpx 0;
}

.remember__label {
  font-size: $fs-sm;
  color: $c-text-2;
}

.auth-submit {
  height: 92rpx;
  font-weight: 600;
  margin-top: 8rpx;
}

.divider {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.divider::before,
.divider::after {
  content: '';
  flex: 1;
  height: 2rpx;
  background-color: $c-border;
}

.divider__text {
  font-size: $fs-xs;
  color: $c-text-3;
}

.demo-entry {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  padding: 22rpx;
  border-radius: $radius-lg;
  background-color: #f5f3ff;
  border: 2rpx dashed #c4b5fd;
  text-align: center;
}
.demo-entry__title {
  font-size: $fs-sm;
  color: #6d28d9;
  font-weight: 600;
}
.demo-entry__hint {
  font-size: $fs-xs;
  color: #7c3aed;
}

.wechat-btn {
  height: 92rpx;
  font-weight: 600;
  background-color: #07c160;
  color: #fff;
}

.wechat-hint {
  text-align: center;
  font-size: 20rpx;
  color: $c-text-3;
}

.auth-switch {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8rpx;
  margin-top: 8rpx;
}

.auth-switch__text {
  font-size: $fs-sm;
  color: $c-text-2;
}

.auth-switch__link {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-primary;
}
</style>
