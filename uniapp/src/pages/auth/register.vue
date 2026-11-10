<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { authApi } from '@/api/auth'
import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import { goHome } from '@/utils/nav'
import { applyAuthResult } from '@/utils/session'
import { toastError } from '@/utils/ui'

type BookType = 'restaurant' | 'family' | 'startup'

const BOOK_TYPES: { value: BookType; emoji: string; label: string; desc: string }[] = [
  { value: 'restaurant', emoji: '🍽️', label: '饭店', desc: '餐厅经营记账' },
  { value: 'family', emoji: '🏠', label: '家庭', desc: '家庭收支管理' },
  { value: 'startup', emoji: '🚀', label: '创业', desc: '初创企业财务' },
]

const authStore = useAuthStore()
const tenantStore = useTenantStore()

const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const bookType = ref<BookType>('restaurant')
const loading = ref(false)

onShow(() => {
  if (authStore.token && tenantStore.currentTenantId) {
    goHome()
  }
})

function backToLogin(): void {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
    return
  }
  uni.reLaunch({ url: '/pages/auth/login' })
}

async function submit(): Promise<void> {
  if (loading.value) return
  if (!username.value.trim()) {
    toastError(new Error('请输入用户名'))
    return
  }
  if (!email.value.trim()) {
    toastError(new Error('请输入邮箱'))
    return
  }
  if (password.value.length < 6) {
    toastError(new Error('密码至少需要 6 个字符'))
    return
  }
  if (password.value !== confirmPassword.value) {
    toastError(new Error('两次输入的密码不一致'))
    return
  }

  loading.value = true
  try {
    const result = await authApi.register({
      username: username.value.trim(),
      email: email.value.trim(),
      password: password.value,
      book_type: bookType.value,
    })
    await applyAuthResult(result)
  } catch (err) {
    toastError(err, '注册失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <scroll-view scroll-y class="auth-page">
    <view class="auth-inner">
      <view class="auth-card">
      <view class="auth-head">
        <text class="auth-title">汤圆记账</text>
        <text class="auth-subtitle">创建您的账号</text>
      </view>

      <view class="field">
        <text class="field__label">用户名</text>
        <input v-model="username" class="input" placeholder="请输入用户名" maxlength="20" />
      </view>

      <view class="field">
        <text class="field__label">邮箱</text>
        <input v-model="email" class="input" placeholder="请输入邮箱" />
      </view>

      <view class="field">
        <text class="field__label">密码</text>
        <input v-model="password" class="input" password placeholder="至少 6 个字符" />
      </view>

      <view class="field">
        <text class="field__label">确认密码</text>
        <input v-model="confirmPassword" class="input" password placeholder="再次输入密码" />
      </view>

      <view class="field">
        <text class="field__label">账本类型</text>
        <view class="book-types">
          <view
            v-for="option in BOOK_TYPES"
            :key="option.value"
            class="book-type"
            :class="{ 'book-type--active': bookType === option.value }"
            @click="bookType = option.value"
          >
            <text class="book-type__emoji">{{ option.emoji }}</text>
            <text class="book-type__label">{{ option.label }}</text>
            <text class="book-type__desc">{{ option.desc }}</text>
          </view>
        </view>
      </view>

      <view class="btn btn-primary btn-block auth-submit" :class="{ 'btn-disabled': loading }" @click="submit">
        {{ loading ? '注册中…' : '注册' }}
      </view>

      <view class="auth-switch">
        <text class="auth-switch__text">已有账号？</text>
        <text class="auth-switch__link" @click="backToLogin">立即登录</text>
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
  gap: 22rpx;
  padding: 52rpx 44rpx;
  border-radius: $radius-lg;
  background-color: #fff;
  box-shadow: 0 20rpx 60rpx rgba(30, 64, 175, 0.12);
}

.auth-head {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10rpx;
  margin-bottom: 8rpx;
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

.book-types {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12rpx;
}

.book-type {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
  padding: 18rpx 6rpx;
  border: 3rpx solid $c-border-2;
  border-radius: $radius-sm;
  background-color: #fff;
}

.book-type--active {
  border-color: $c-primary;
  background-color: $c-primary-bg;
}

.book-type__emoji {
  font-size: 44rpx;
  line-height: 1.2;
}

.book-type__label {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.book-type--active .book-type__label {
  color: $c-primary;
}

.book-type__desc {
  font-size: 18rpx;
  color: $c-text-3;
  text-align: center;
}

.auth-submit {
  height: 92rpx;
  font-weight: 600;
  margin-top: 8rpx;
}

.auth-switch {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8rpx;
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
