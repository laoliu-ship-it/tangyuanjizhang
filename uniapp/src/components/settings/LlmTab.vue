<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import PromptSheet from './PromptSheet.vue'
import { llmApi } from '@/api/llm'
import type { TenantLLMConfig } from '@/api/types'
import { toast, toastError } from '@/utils/ui'

const PROVIDERS = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'deepseek', label: 'DeepSeek' },
  { value: 'azure', label: 'Azure OpenAI' },
  { value: 'ollama', label: 'Ollama（本地）' },
]

const MODES: { value: 'vision' | 'ocr_text'; label: string; desc: string }[] = [
  { value: 'ocr_text', label: '文字模式', desc: '将 OCR 文本发给 AI 分析' },
  { value: 'vision', label: '视觉模式', desc: '将图片直接发给 AI 分析' },
]

const config = ref<TenantLLMConfig | null>(null)
const loading = ref(true)
const loadError = ref('')
const saving = ref(false)
const promptVisible = ref(false)

const enabled = ref(false)
const usePlatform = ref(true)
const providerIndex = ref(0)
const baseURL = ref('')
const apiKey = ref('')
const model = ref('gpt-4o')
const mode = ref<'vision' | 'ocr_text'>('ocr_text')

const platformUnavailable = computed(
  () => enabled.value && usePlatform.value && !!config.value && !config.value.platform_enabled,
)

const providerLabel = computed(() => PROVIDERS[providerIndex.value]?.label ?? 'OpenAI')

async function load(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    const data = await llmApi.getConfig()
    config.value = data
    enabled.value = !!data?.enabled
    usePlatform.value = data?.use_platform ?? true
    const index = PROVIDERS.findIndex((item) => item.value === data?.provider)
    providerIndex.value = index >= 0 ? index : 0
    baseURL.value = data?.base_url ?? ''
    model.value = data?.model ?? 'gpt-4o'
    mode.value = data?.mode === 'vision' ? 'vision' : 'ocr_text'
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : '加载 AI 配置失败（仅管理员可用）'
  } finally {
    loading.value = false
  }
}

function onProviderChange(e: Event): void {
  const detail = (e as Event & { detail?: { value?: string | number } }).detail
  providerIndex.value = Number(detail?.value ?? 0)
}

function onEnabledChange(e: Event): void {
  const detail = (e as Event & { detail?: { value?: boolean } }).detail
  enabled.value = !!detail?.value
}

async function save(): Promise<void> {
  if (saving.value) return
  if (platformUnavailable.value) {
    toast('平台暂未开放 AI 分析功能，请联系管理员，或切换为「使用自定义配置」')
    return
  }
  saving.value = true
  try {
    const payload: Parameters<typeof llmApi.saveConfig>[0] = {
      enabled: enabled.value,
      use_platform: usePlatform.value,
      provider: PROVIDERS[providerIndex.value]?.value ?? 'openai',
      base_url: baseURL.value,
      model: model.value,
      mode: mode.value,
    }
    if (apiKey.value.trim()) payload.api_key = apiKey.value.trim()
    const data = await llmApi.saveConfig(payload)
    config.value = data
    apiKey.value = ''
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
</script>

<template>
  <view class="llm">
    <view v-if="loading" class="llm__state">加载中…</view>
    <view v-else-if="loadError" class="llm__state llm__state--error">
      <text class="llm__error-title">无法加载 AI 配置</text>
      <text class="llm__error-desc">{{ loadError }}</text>
      <view class="btn btn-primary btn-sm" @click="load">重试</view>
    </view>

    <template v-else>
      <view class="card">
        <view class="row-between">
          <view class="llm__text">
            <text class="llm__title">启用 AI 分析</text>
            <text class="llm__desc">上传票据时自动调用大模型识别金额、商户、分类</text>
          </view>
          <switch :checked="enabled" color="#2563eb" style="transform: scale(0.8)" @change="onEnabledChange" />
        </view>
      </view>

      <template v-if="enabled">
        <view class="card">
          <text class="llm__section-title">API 来源</text>
          <view class="source-list">
            <view class="source" @click="usePlatform = true">
              <view class="source__radio" :class="{ 'source__radio--on': usePlatform }" />
              <view class="source__body">
                <text class="source__title">使用平台提供的 AI</text>
                <text class="source__desc">无需配置，直接使用平台统一服务</text>
              </view>
            </view>
            <view class="source" @click="usePlatform = false">
              <view class="source__radio" :class="{ 'source__radio--on': !usePlatform }" />
              <view class="source__body">
                <text class="source__title">使用自定义 API Key</text>
                <text class="source__desc">使用自己的 OpenAI / DeepSeek 等账号</text>
              </view>
            </view>
          </view>
          <view v-if="platformUnavailable" class="llm__warn">
            平台暂未配置 AI 服务，请切换为「使用自定义 API Key」或联系管理员。
          </view>
        </view>

        <view v-if="!usePlatform" class="card custom">
          <text class="llm__section-title">自定义配置</text>

          <view class="field">
            <text class="field__label">Provider</text>
            <picker mode="selector" :range="PROVIDERS" range-key="label" :value="providerIndex" @change="onProviderChange">
              <view class="field__picker">
                <text>{{ providerLabel }}</text>
                <text class="field__arrow">⌄</text>
              </view>
            </picker>
          </view>

          <view class="field">
            <text class="field__label">API Base URL（OpenAI 兼容接口可留空）</text>
            <input v-model="baseURL" class="input" placeholder="https://api.openai.com/v1" />
          </view>

          <view class="field">
            <text class="field__label">
              API Key
              <text v-if="config?.api_key_mask" class="field__mask">当前：{{ config.api_key_mask }}</text>
            </text>
            <input
              v-model="apiKey"
              class="input"
              password
              :placeholder="config?.api_key_mask ? '不修改请留空' : '输入 API Key'"
            />
          </view>

          <view class="field">
            <text class="field__label">模型名称</text>
            <input v-model="model" class="input" placeholder="gpt-4o" />
          </view>
        </view>

        <view class="card">
          <text class="llm__section-title">识别模式</text>
          <view class="mode-list">
            <view
              v-for="item in MODES"
              :key="item.value"
              class="mode"
              :class="{ 'mode--active': mode === item.value }"
              @click="mode = item.value"
            >
              <view class="source__radio" :class="{ 'source__radio--on': mode === item.value }" />
              <view class="source__body">
                <text class="source__title">{{ item.label }}</text>
                <text class="source__desc">{{ item.desc }}</text>
              </view>
            </view>
          </view>

          <view v-if="mode === 'vision'" class="llm__warn">
            <text class="llm__warn-title">视觉模式需要模型支持图片输入</text>
            <text class="llm__warn-text">
              常用模型：gpt-4o、gpt-4o-mini、claude-3 等；DeepSeek 请使用 deepseek-chat 或切换为文字模式。
            </text>
          </view>
        </view>

        <view class="card prompt-entry" @click="promptVisible = true">
          <view class="prompt-entry__body">
            <text class="llm__title">提示词设置</text>
            <text class="llm__desc">自定义 AI 识别票据时使用的提示词</text>
          </view>
          <text class="prompt-entry__arrow">›</text>
        </view>
      </template>

      <view
        class="btn btn-primary btn-block"
        :class="{ 'btn-disabled': saving || platformUnavailable }"
        @click="save"
      >
        {{ saving ? '保存中…' : '保存配置' }}
      </view>
    </template>

    <PromptSheet v-model:visible="promptVisible" />
  </view>
</template>

<style lang="scss" scoped>
.llm {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.llm__state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14rpx;
  padding: 70rpx 20rpx;
  font-size: $fs-sm;
  color: $c-text-3;
}

.llm__error-title {
  font-size: $fs;
  font-weight: 600;
  color: $c-expense;
}

.llm__error-desc {
  font-size: $fs-xs;
  color: $c-text-3;
  text-align: center;
}

.llm__text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.llm__title {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.llm__desc {
  font-size: $fs-xs;
  color: $c-text-3;
  line-height: 1.5;
}

.llm__section-title {
  display: block;
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
  margin-bottom: 18rpx;
}

.source-list,
.mode-list {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}

.source,
.mode {
  display: flex;
  align-items: flex-start;
  gap: 14rpx;
  padding: 16rpx;
  border: 2rpx solid $c-border-2;
  border-radius: $radius-sm;
  background-color: #fff;
}

.mode--active {
  border-color: $c-primary;
  background-color: $c-primary-bg;
}

.source__radio {
  width: 32rpx;
  height: 32rpx;
  flex-shrink: 0;
  margin-top: 4rpx;
  border-radius: 50%;
  border: 3rpx solid $c-border-2;
  background-color: #fff;
}

.source__radio--on {
  border-color: $c-primary;
  background-color: $c-primary;
  box-shadow: inset 0 0 0 6rpx #fff;
}

.source__body {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.source__title {
  font-size: $fs-sm;
  font-weight: 500;
  color: $c-text;
}

.source__desc {
  font-size: $fs-xs;
  color: $c-text-3;
}

.custom {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 10rpx;
}

.field__label {
  font-size: $fs-xs;
  color: $c-text-2;
}

.field__mask {
  margin-left: 10rpx;
  color: $c-text-3;
}

.field__picker {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 80rpx;
  padding: 0 24rpx;
  border: 2rpx solid $c-border-2;
  border-radius: $radius-sm;
  background-color: #fff;
  font-size: $fs;
  color: $c-text;
}

.field__arrow {
  font-size: 26rpx;
  color: $c-text-3;
}

.llm__warn {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  padding: 18rpx 20rpx;
  border-radius: $radius-sm;
  background-color: $c-warn-bg;
  font-size: $fs-xs;
  color: $c-warn;
  line-height: 1.6;
}

.llm__warn-title {
  font-weight: 600;
}

.llm__warn-text {
  color: #92400e;
}

.prompt-entry {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.prompt-entry__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.prompt-entry__arrow {
  font-size: 40rpx;
  color: $c-text-3;
  line-height: 1;
}
</style>
