<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import AppModal from '@/components/AppModal.vue'
import { llmApi } from '@/api/llm'
import type { TenantPrompt } from '@/api/types'
import { confirm, toast, toastError } from '@/utils/ui'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
}>()

const visibleModel = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

const prompt = ref<TenantPrompt | null>(null)
const text = ref('')
const loading = ref(false)
const loadError = ref('')
const saving = ref(false)

const isCustom = computed(() => !!prompt.value?.custom_prompt)
const placeholder = computed(() => prompt.value?.placeholder ?? '{{分类列表}}')

async function load(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    const data = await llmApi.getPrompt()
    prompt.value = data
    text.value = data?.custom_prompt || data?.default_prompt || ''
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : '加载提示词失败'
  } finally {
    loading.value = false
  }
}

watch(
  () => props.visible,
  (value) => {
    if (value) void load()
  },
)

function insertPlaceholder(): void {
  text.value = `${text.value}${text.value ? '\n' : ''}${placeholder.value}`
}

async function save(): Promise<void> {
  const value = text.value.trim()
  if (!value) {
    toast('提示词不能为空，如需恢复平台默认请点击「恢复平台默认」')
    return
  }
  if (saving.value) return
  saving.value = true
  try {
    const data = await llmApi.savePrompt({ custom_prompt: value })
    prompt.value = data
    text.value = data?.custom_prompt || data?.default_prompt || ''
    toast(data?.custom_prompt ? '已保存自定义提示词' : '继续使用平台默认提示词', 'success')
  } catch (err) {
    toastError(err, '保存失败')
  } finally {
    saving.value = false
  }
}

async function reset(): Promise<void> {
  const ok = await confirm('确定恢复平台默认提示词吗？自定义内容将被清除。', '恢复默认')
  if (!ok) return
  saving.value = true
  try {
    const data = await llmApi.savePrompt({ custom_prompt: '' })
    prompt.value = data
    text.value = data?.default_prompt || ''
    toast('已恢复平台默认提示词', 'success')
  } catch (err) {
    toastError(err, '操作失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <AppModal v-model:visible="visibleModel" title="提示词设置" position="bottom">
    <view class="prompt">
      <view v-if="loading" class="prompt__state">加载中…</view>
      <view v-else-if="loadError" class="prompt__state">
        <text>{{ loadError }}</text>
        <text class="prompt__retry" @click="load">重试</text>
      </view>

      <template v-else>
        <view class="prompt__status">
          <text class="prompt__status-label">当前生效</text>
          <text class="prompt__status-tag" :class="{ 'prompt__status-tag--custom': isCustom }">
            {{ isCustom ? '自定义提示词' : '平台默认' }}
          </text>
        </view>

        <text class="prompt__desc">
          截图识别时发给 AI 的指令，决定如何从票据中提取金额、商户、时间、分类等字段。可在平台默认提示词基础上补充你常用的商户与分类规则，保存后即用你的版本。
        </text>

        <view class="prompt__placeholder">
          <text class="prompt__placeholder-title">分类列表占位符</text>
          <text class="prompt__placeholder-text">
            提示词中的 {{ placeholder }} 会在每次识别时自动替换为本账本的分类列表（id / 名称 / 收支类型）。
            若提示词中没有该占位符，分类列表会自动追加到提示词末尾。
          </text>
        </view>

        <view class="prompt__editor-head">
          <text class="prompt__editor-label">提示词内容</text>
          <text class="prompt__insert" @click="insertPlaceholder">插入 {{ placeholder }}</text>
        </view>
        <textarea
          v-model="text"
          class="prompt__textarea"
          :maxlength="-1"
          placeholder="输入提示词内容"
        />
        <text class="prompt__count">{{ text.length }} 字</text>
      </template>
    </view>

    <template #footer>
      <view class="prompt__footer">
        <view class="btn prompt__btn" :class="{ 'btn-disabled': saving }" @click="reset">
          恢复平台默认
        </view>
        <view
          class="btn btn-primary prompt__btn"
          :class="{ 'btn-disabled': saving || loading || !!loadError }"
          @click="save"
        >
          {{ saving ? '保存中…' : '保存' }}
        </view>
      </view>
    </template>
  </AppModal>
</template>

<style lang="scss" scoped>
.prompt {
  display: flex;
  flex-direction: column;
  gap: 18rpx;
}

.prompt__state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16rpx;
  padding: 60rpx 0;
  font-size: $fs-sm;
  color: $c-text-3;
}

.prompt__retry {
  color: $c-primary;
}

.prompt__status {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.prompt__status-label {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.prompt__status-tag {
  padding: 4rpx 16rpx;
  border-radius: 999rpx;
  background-color: #f3f4f6;
  color: $c-text-2;
  font-size: $fs-xs;
}

.prompt__status-tag--custom {
  background-color: $c-primary-bg;
  color: $c-primary;
}

.prompt__desc {
  font-size: $fs-xs;
  color: $c-text-3;
  line-height: 1.7;
}

.prompt__placeholder {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  padding: 18rpx 20rpx;
  border-radius: $radius-sm;
  background-color: $c-primary-bg;
}

.prompt__placeholder-title {
  font-size: $fs-xs;
  font-weight: 600;
  color: $c-primary;
}

.prompt__placeholder-text {
  font-size: 20rpx;
  color: #1d4ed8;
  line-height: 1.6;
}

.prompt__editor-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.prompt__editor-label {
  font-size: $fs-xs;
  color: $c-text-2;
}

.prompt__insert {
  font-size: $fs-xs;
  color: $c-primary;
  padding: 4rpx 8rpx;
}

.prompt__textarea {
  width: 100%;
  height: 520rpx;
  padding: 20rpx;
  border: 2rpx solid $c-border-2;
  border-radius: $radius-sm;
  background-color: #fff;
  font-size: $fs-xs;
  line-height: 1.7;
  color: $c-text;
}

.prompt__count {
  text-align: right;
  font-size: 20rpx;
  color: $c-text-3;
}

.prompt__footer {
  display: flex;
  gap: 16rpx;
}

.prompt__btn {
  flex: 1;
  height: 84rpx;
}
</style>
