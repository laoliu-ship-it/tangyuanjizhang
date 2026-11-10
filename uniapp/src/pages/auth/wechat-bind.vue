<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { authApi } from '@/api/auth'
import { STORAGE_KEYS } from '@/config'
import { routes } from '@/utils/nav'
import { applyAuthResult } from '@/utils/session'
import { toastError } from '@/utils/ui'

type BookType = 'restaurant' | 'family' | 'startup'
type Mode = 'bind' | 'create'

const BOOK_TYPES: { value: BookType; emoji: string; label: string; desc: string }[] = [
  { value: 'restaurant', emoji: '🍽️', label: '饭店', desc: '餐厅经营记账' },
  { value: 'family', emoji: '🏠', label: '家庭', desc: '家庭收支管理' },
  { value: 'startup', emoji: '🚀', label: '创业', desc: '初创企业财务' },
]

const mode = ref<Mode>('bind')
const bindToken = ref('')
const email = ref('')
const password = ref('')
const username = ref('')
const bookType = ref<BookType>('restaurant')
const loading = ref(false)

onLoad(() => {
  let raw = ''
  try {
    const stored = uni.getStorageSync(STORAGE_KEYS.wechatBind)
    raw = typeof stored === 'string' ? stored : ''
    uni.removeStorageSync(STORAGE_KEYS.wechatBind)
  } catch (e) {
    raw = ''
  }
  bindToken.value = raw
  if (!bindToken.value) {
    toastError(new Error('登录会话已失效，请重新登录'))
    setTimeout(() => backToLogin(), 800)
  }
})

function backToLogin(): void {
  uni.reLaunch({ url: routes.login })
}

async function submitBind(): Promise<void> {
  if (loading.value) return
  if (!bindToken.value) {
    toastError(new Error('登录会话已失效，请重新登录'))
    return
  }
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
    const result = await authApi.wechatBind({
      bind_token: bindToken.value,
      email: email.value.trim(),
      password: password.value,
    })
    await applyAuthResult(result)
  } catch (err) {
    toastError(err, '关联失败，请检查邮箱和密码')
  } finally {
    loading.value = false
  }
}

async function submitCreate(): Promise<void> {
  if (loading.value) return
  if (!bindToken.value) {
    toastError(new Error('登录会话已失效，请重新登录'))
    return
  }
  if (!username.value.trim()) {
    toastError(new Error('请输入用户名'))
    return
  }

  loading.value = true
  try {
    const result = await authApi.wechatRegister({
      bind_token: bindToken.value,
      username: username.value.trim(),
      book_type: bookType.value,
    })
    await applyAuthResult(result)
  } catch (err) {
    toastError(err, '创建账号失败，请重试')
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
          <text class="auth-title">微信登录</text>
          <text class="auth-subtitle">首次使用，请选择关联方式</text>
        </view>

        <view class="tip">
          <text class="tip__text">
            你的微信还没有绑定汤圆记账。已有账号请关联之前的邮箱；没有账号可直接用微信创建新账本。
          </text>
        </view>

        <view class="seg">
          <view
            class="seg__item"
            :class="{ 'seg__item--active': mode === 'bind' }"
            @click="mode = 'bind'"
          >
            关联已有账号
          </view>
          <view
            class="seg__item"
            :class="{ 'seg__item--active': mode === 'create' }"
            @click="mode = 'create'"
          >
            创建新账号
          </view>
        </view>

        <template v-if="mode === 'bind'">
          <view class="field">
            <text class="field__label">邮箱</text>
            <input
              v-model="email"
              class="input"
              type="text"
              placeholder="请输入注册时的邮箱"
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
              @confirm="submitBind"
            />
          </view>

          <view
            class="btn btn-primary btn-block auth-submit"
            :class="{ 'btn-disabled': loading }"
            @click="submitBind"
          >
            {{ loading ? '关联中…' : '关联并登录' }}
          </view>
        </template>

        <template v-else>
          <view class="field">
            <text class="field__label">用户名</text>
            <input v-model="username" class="input" placeholder="请输入用户名" maxlength="20" />
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

          <view
            class="btn btn-primary btn-block auth-submit"
            :class="{ 'btn-disabled': loading }"
            @click="submitCreate"
          >
            {{ loading ? '创建中…' : '创建账号并登录' }}
          </view>
        </template>

        <view class="auth-switch">
          <text class="auth-switch__text">不想关联？</text>
          <text class="auth-switch__link" @click="backToLogin">返回邮箱登录</text>
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
  font-size: 48rpx;
  font-weight: 700;
  color: $c-primary;
}

.auth-subtitle {
  font-size: $fs-sm;
  color: $c-text-2;
}

.tip {
  padding: 18rpx 20rpx;
  border-radius: $radius-sm;
  background-color: $c-primary-bg;
}

.tip__text {
  font-size: $fs-xs;
  color: $c-text-2;
  line-height: 1.7;
}

.seg {
  display: flex;
  padding: 4rpx;
  border-radius: 10rpx;
  background-color: #f3f4f6;
}

.seg__item {
  flex: 1;
  padding: 16rpx 0;
  border-radius: 8rpx;
  text-align: center;
  font-size: $fs-sm;
  color: $c-text-2;
}

.seg__item--active {
  background-color: #fff;
  color: $c-text;
  font-weight: 600;
  box-shadow: 0 2rpx 6rpx rgba(17, 24, 39, 0.08);
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
