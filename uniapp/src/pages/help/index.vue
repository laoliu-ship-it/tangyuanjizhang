<script setup lang="ts">
import { computed, ref } from 'vue'
import AppPage from '@/components/AppPage.vue'
import { useAuthGuard } from '@/composables/useAuthGuard'

useAuthGuard()

type Platform = 'ios' | 'android'

const platform = ref<Platform>('ios')
const openFaq = ref<string[]>([])
const isMp = ref(false)
// #ifdef MP-WEIXIN
isMp.value = true
// #endif

const IOS_STEPS = [
  '在 iPhone 上打开 Safari 浏览器，访问记账网页',
  '点击底部导航栏的「分享」按钮（方框带向上箭头的图标）',
  '在弹出的菜单中向下滑动，找到并点击「添加到主屏幕」',
  '确认后，桌面就会出现记账本的快捷图标，以后点击即可快速打开',
]

const ANDROID_STEPS = [
  '在 Android 手机上打开 Chrome 浏览器，访问记账网页',
  '点击右上角的「菜单」按钮（三个竖点的图标 ⋮）',
  '在弹出的菜单中点击「添加到主屏幕」（部分版本显示「安装应用」）',
  '确认后，桌面就会出现记账本的快捷图标，以后点击即可快速打开',
]

const steps = computed(() => (platform.value === 'ios' ? IOS_STEPS : ANDROID_STEPS))

const FAQS: { id: string; question: string; answer: string }[] = [
  {
    id: 'expense-image',
    question: '为什么新增支出时提示需要上传截图？',
    answer:
      '账本管理员可在「账本设置 - 通用」中开启「支出记录必须上传图片」。开启后，新增支出必须附上票据截图才能保存；如不需要校验，可联系管理员关闭该开关。',
  },
  {
    id: 'ai-draft',
    question: '上传一张票据后出现多笔 AI 建议，是什么原因？',
    answer:
      '当票据中包含多笔消费时，AI 会按行拆分出多笔建议。点击「接受」会把建议填入表单并暂存，确认每笔分类后点击「完成」统一提交即可。',
  },
  {
    id: 'credits',
    question: '积分是做什么用的？',
    answer:
      '上传图片、OCR 识别、AI 分析会消耗积分。积分按批次管理，最近过期的批次优先扣除。点击顶部积分徽标可以查看余额和明细；如显示「无限制」，表示当前账本已加入白名单。',
  },
  {
    id: 'excel',
    question: '如何批量导入 / 导出交易？',
    answer:
      '进入「交易记录」，点击右上角「⋯」可选择下载模板、导入 Excel 或导出当前筛选结果。微信小程序可从聊天记录中选择 Excel 文件；H5 会直接下载到本地。',
  },
  {
    id: 'member',
    question: '如何邀请家人或同事一起记账？',
    answer:
      '账本所有者 / 管理员进入「账本设置 - 成员」，输入对方的用户名并选择角色后点击「邀请」。成员角色为只读，如需编辑权限请选择管理员或财务等角色。',
  },
]

function isOpen(id: string): boolean {
  return openFaq.value.includes(id)
}

function toggleFaq(id: string): void {
  if (openFaq.value.includes(id)) {
    openFaq.value = openFaq.value.filter((item) => item !== id)
  } else {
    openFaq.value = [...openFaq.value, id]
  }
}
</script>

<template>
  <AppPage title="帮助中心" tabbar="help">
    <view class="help">
      <!-- 添加到桌面：H5 / App 展示，微信小程序内隐藏 -->
      <view v-if="!isMp" class="card guide">
        <view class="guide__head">
          <view class="guide__title-wrap">
            <text class="guide__title">添加到手机桌面</text>
            <text class="guide__sub">像原生 App 一样快速打开</text>
          </view>
          <view class="seg">
            <view
              class="seg__item"
              :class="{ 'seg__item--active': platform === 'ios' }"
              @click="platform = 'ios'"
            >
              iOS
            </view>
            <view
              class="seg__item"
              :class="{ 'seg__item--active': platform === 'android' }"
              @click="platform = 'android'"
            >
              Android
            </view>
          </view>
        </view>

        <view class="steps">
          <view v-for="(step, index) in steps" :key="index" class="step">
            <text
              class="step__num"
              :class="platform === 'ios' ? 'step__num--ios' : 'step__num--android'"
            >
              {{ index + 1 }}
            </text>
            <text class="step__text">{{ step }}</text>
          </view>
        </view>

        <view class="phones">
          <view class="phone">
            <text class="phone__icon">{{ platform === 'ios' ? '⬆️' : '⋮' }}</text>
            <text class="phone__label">{{ platform === 'ios' ? '点击分享' : '点击菜单' }}</text>
          </view>
          <view class="phone">
            <text class="phone__icon">📲</text>
            <text class="phone__label">添加到主屏幕</text>
          </view>
        </view>

        <view class="tip">
          <text class="tip__title">提示</text>
          <text class="tip__text">
            添加到桌面后，打开应用就像使用原生 App 一样方便，没有浏览器地址栏干扰。
          </text>
        </view>
      </view>

      <!-- 微信小程序：改为「添加到我的小程序」引导 -->
      <view v-else class="card guide">
        <view class="guide__title-wrap">
          <text class="guide__title">添加到我的小程序</text>
          <text class="guide__sub">以后可在微信首页下拉快速打开</text>
        </view>
        <view class="tip tip--mp">
          <text class="tip__title">操作步骤</text>
          <text class="tip__text">
            点击右上角「···」，选择「添加到我的小程序」，以后可在微信首页下拉快速打开。
          </text>
        </view>
      </view>

      <!-- FAQ -->
      <view class="card faq-card">
        <text class="faq-card__title">常见问题</text>
        <view v-for="faq in FAQS" :key="faq.id" class="faq">
          <view class="faq__question" @click="toggleFaq(faq.id)">
            <text class="faq__question-text">{{ faq.question }}</text>
            <text class="faq__arrow" :class="{ 'faq__arrow--open': isOpen(faq.id) }">⌄</text>
          </view>
          <text v-if="isOpen(faq.id)" class="faq__answer">{{ faq.answer }}</text>
        </view>
      </view>
    </view>
  </AppPage>
</template>

<style lang="scss" scoped>
.help {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.guide {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.guide__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16rpx;
}

.guide__title-wrap {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.guide__title {
  font-size: $fs;
  font-weight: 600;
  color: $c-text;
}

.guide__sub {
  font-size: 20rpx;
  color: $c-text-3;
}

.seg {
  display: flex;
  flex-shrink: 0;
  padding: 4rpx;
  border-radius: 10rpx;
  background-color: #f3f4f6;
}

.seg__item {
  padding: 8rpx 18rpx;
  border-radius: 8rpx;
  font-size: 20rpx;
  color: $c-text-2;
}

.seg__item--active {
  background-color: #fff;
  color: $c-text;
  font-weight: 600;
  box-shadow: 0 2rpx 6rpx rgba(17, 24, 39, 0.08);
}

.steps {
  display: flex;
  flex-direction: column;
  gap: 18rpx;
}

.step {
  display: flex;
  align-items: flex-start;
  gap: 14rpx;
}

.step__num {
  width: 36rpx;
  height: 36rpx;
  flex-shrink: 0;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20rpx;
  font-weight: 700;
  color: #fff;
  background-color: $c-primary;
}

.step__num--android {
  background-color: $c-income;
}

.step__text {
  flex: 1;
  font-size: $fs-sm;
  color: $c-text-2;
  line-height: 1.6;
}

.phones {
  display: flex;
  justify-content: center;
  gap: 40rpx;
  padding: 24rpx;
  border-radius: $radius-sm;
  background-color: $c-bg;
  border: 2rpx dashed $c-border-2;
}

.phone {
  width: 180rpx;
  height: 300rpx;
  border-radius: 24rpx;
  border: 2rpx solid $c-border-2;
  background-color: #fff;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  box-shadow: 0 6rpx 16rpx rgba(17, 24, 39, 0.06);
}

.phone__icon {
  font-size: 48rpx;
  line-height: 1;
}

.phone__label {
  font-size: 20rpx;
  color: $c-text-3;
}

.tip {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  padding: 18rpx 20rpx;
  border-radius: $radius-sm;
  background-color: $c-primary-bg;
}

.tip--mp {
  background-color: $c-income-bg;
}

.tip__title {
  font-size: $fs-xs;
  font-weight: 600;
  color: $c-primary;
}

.tip--mp .tip__title {
  color: $c-income;
}

.tip__text {
  font-size: 20rpx;
  color: $c-text-2;
  line-height: 1.6;
}

.faq-card {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.faq-card__title {
  font-size: $fs;
  font-weight: 600;
  color: $c-text;
  margin-bottom: 8rpx;
}

.faq {
  border-bottom: 2rpx solid $c-border;
}

.faq:last-child {
  border-bottom: none;
}

.faq__question {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  padding: 24rpx 0;
}

.faq__question-text {
  flex: 1;
  font-size: $fs-sm;
  font-weight: 500;
  color: $c-text;
  line-height: 1.5;
}

.faq__arrow {
  flex-shrink: 0;
  font-size: 28rpx;
  color: $c-text-3;
  transition: transform 0.2s;
}

.faq__arrow--open {
  transform: rotate(180deg);
}

.faq__answer {
  display: block;
  padding: 0 0 24rpx;
  font-size: $fs-xs;
  color: $c-text-2;
  line-height: 1.8;
}
</style>
