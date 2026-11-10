<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { categoryApi } from '@/api/category'
import { useMetaStore } from '@/store/meta'
import type { Category, TxType } from '@/api/types'
import { confirm, toast, toastError } from '@/utils/ui'

const COMMON_ICONS = [
  '🍽️', '🛒', '🚗', '🏠', '💊', '🎮', '👕', '✈️',
  '📚', '💡', '🎵', '💰', '💳', '🎁', '🏋️', '📌',
]

const meta = useMetaStore()

const categories = ref<Category[]>([])
const loading = ref(true)
const errorText = ref('')

const addType = ref<TxType>('expense')
const addName = ref('')
const addIcon = ref('📌')
const adding = ref(false)

const editingId = ref(0)
const editName = ref('')
const editIcon = ref('📌')
const editType = ref<TxType>('expense')
const updating = ref(false)
const deletingId = ref(0)

const expenseCategories = computed(() => categories.value.filter((item) => item.type === 'expense'))
const incomeCategories = computed(() => categories.value.filter((item) => item.type === 'income'))
const currentList = computed(() =>
  addType.value === 'expense' ? expenseCategories.value : incomeCategories.value,
)
const currentTypeLabel = computed(() => (addType.value === 'expense' ? '支出' : '收入'))

async function load(): Promise<void> {
  loading.value = true
  errorText.value = ''
  try {
    const list = await categoryApi.list()
    categories.value = Array.isArray(list) ? list : []
    meta.setCategories(categories.value)
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '加载分类失败'
  } finally {
    loading.value = false
  }
}

function setAddType(type: TxType): void {
  addType.value = type
}

async function add(): Promise<void> {
  const name = addName.value.trim()
  if (!name) {
    toast('请输入分类名称')
    return
  }
  if (adding.value) return
  adding.value = true
  try {
    await categoryApi.create({ name, type: addType.value, icon: addIcon.value || '📌' })
    addName.value = ''
    addIcon.value = '📌'
    toast('添加成功', 'success')
    await load()
  } catch (err) {
    toastError(err, '创建分类失败')
  } finally {
    adding.value = false
  }
}

function startEdit(category: Category): void {
  editingId.value = category.id
  editName.value = category.name
  editIcon.value = category.icon || '📌'
  editType.value = category.type
}

function cancelEdit(): void {
  editingId.value = 0
  editName.value = ''
  editIcon.value = '📌'
  editType.value = 'expense'
}

async function saveEdit(): Promise<void> {
  const name = editName.value.trim()
  if (!name) {
    toast('请输入分类名称')
    return
  }
  if (updating.value) return
  updating.value = true
  try {
    await categoryApi.update(editingId.value, {
      name,
      type: editType.value,
      icon: editIcon.value || '📌',
    })
    cancelEdit()
    toast('已保存', 'success')
    await load()
  } catch (err) {
    toastError(err, '更新分类失败')
  } finally {
    updating.value = false
  }
}

async function remove(category: Category): Promise<void> {
  const ok = await confirm('确认删除该分类？删除后相关交易将无法显示分类信息。', '删除分类')
  if (!ok) return
  deletingId.value = category.id
  try {
    await categoryApi.remove(category.id)
    toast('已删除', 'success')
    await load()
  } catch (err) {
    toastError(err, '删除失败')
  } finally {
    deletingId.value = 0
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <view class="cats">
    <!-- 新增分类 -->
    <view class="card add-card">
      <text class="add-card__title">新增分类</text>

      <view class="type-toggle">
        <view
          class="type-toggle__item"
          :class="{ 'type-toggle__item--expense': addType === 'expense' }"
          @click="setAddType('expense')"
        >
          支出
        </view>
        <view
          class="type-toggle__item"
          :class="{ 'type-toggle__item--income': addType === 'income' }"
          @click="setAddType('income')"
        >
          收入
        </view>
      </view>

      <text class="add-card__label">选择图标</text>
      <view class="icon-grid">
        <view
          v-for="icon in COMMON_ICONS"
          :key="icon"
          class="icon-grid__item"
          :class="{ 'icon-grid__item--active': addIcon === icon }"
          @click="addIcon = icon"
        >
          {{ icon }}
        </view>
      </view>
      <input v-model="addIcon" class="input icon-input" placeholder="或输入 emoji" maxlength="4" />

      <view class="add-card__row">
        <input v-model="addName" class="input add-card__name" placeholder="分类名称" maxlength="20" />
        <view
          class="btn btn-primary add-card__btn"
          :class="{ 'btn-disabled': adding }"
          @click="add"
        >
          {{ adding ? '添加中…' : '添加' }}
        </view>
      </view>
    </view>

    <!-- 分类列表 -->
    <view class="card list-card">
      <view class="list-card__head" :class="addType === 'expense' ? 'list-card__head--expense' : 'list-card__head--income'">
        <text class="list-card__title">{{ currentTypeLabel }}分类（{{ currentList.length }}）</text>
      </view>

      <view v-if="loading" class="list-state">加载中…</view>
      <view v-else-if="errorText" class="list-state">
        <text>{{ errorText }}</text>
        <text class="list-state__retry" @click="load">重试</text>
      </view>
      <view v-else-if="currentList.length === 0" class="list-state">
        暂无{{ currentTypeLabel }}分类
      </view>

      <view v-else class="cat-list">
        <view v-for="category in currentList" :key="category.id" class="cat-item">
          <!-- 编辑态 -->
          <view v-if="editingId === category.id" class="cat-edit">
            <view class="type-toggle">
              <view
                class="type-toggle__item"
                :class="{ 'type-toggle__item--expense': editType === 'expense' }"
                @click="editType = 'expense'"
              >
                支出
              </view>
              <view
                class="type-toggle__item"
                :class="{ 'type-toggle__item--income': editType === 'income' }"
                @click="editType = 'income'"
              >
                收入
              </view>
            </view>
            <view class="icon-grid icon-grid--compact">
              <view
                v-for="icon in COMMON_ICONS"
                :key="icon"
                class="icon-grid__item"
                :class="{ 'icon-grid__item--active': editIcon === icon }"
                @click="editIcon = icon"
              >
                {{ icon }}
              </view>
            </view>
            <input v-model="editIcon" class="input icon-input" placeholder="或输入 emoji" maxlength="4" />
            <view class="cat-edit__row">
              <input v-model="editName" class="input cat-edit__name" placeholder="分类名称" maxlength="20" />
              <view class="btn btn-primary btn-sm" :class="{ 'btn-disabled': updating }" @click="saveEdit">
                {{ updating ? '保存中…' : '保存' }}
              </view>
              <view class="btn btn-sm" @click="cancelEdit">取消</view>
            </view>
          </view>

          <!-- 展示态 -->
          <view v-else class="cat-item__row">
            <text class="cat-item__icon">{{ category.icon || '📌' }}</text>
            <text class="cat-item__name">{{ category.name }}</text>
            <view class="cat-item__actions">
              <text class="cat-item__action" @click="startEdit(category)">编辑</text>
              <text
                class="cat-item__action cat-item__action--danger"
                @click="remove(category)"
              >
                {{ deletingId === category.id ? '删除中' : '删除' }}
              </text>
            </view>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.cats {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.add-card {
  display: flex;
  flex-direction: column;
  gap: 18rpx;
}

.add-card__title {
  font-size: $fs-sm;
  font-weight: 600;
  color: $c-text;
}

.add-card__label {
  font-size: $fs-xs;
  color: $c-text-2;
}

.type-toggle {
  display: flex;
  border: 2rpx solid $c-border-2;
  border-radius: $radius-sm;
  overflow: hidden;
}

.type-toggle__item {
  flex: 1;
  padding: 16rpx 0;
  text-align: center;
  font-size: $fs-sm;
  color: $c-text-2;
  background-color: #fff;
}

.type-toggle__item--expense {
  background-color: $c-expense;
  color: #fff;
}

.type-toggle__item--income {
  background-color: $c-income;
  color: #fff;
}

.icon-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 10rpx;
}

.icon-grid__item {
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32rpx;
  border: 2rpx solid $c-border-2;
  border-radius: 12rpx;
  background-color: #fff;
}

.icon-grid__item--active {
  border-color: $c-primary;
  background-color: $c-primary-bg;
}

.icon-grid--compact {
  grid-template-columns: repeat(8, 1fr);
}

.icon-input {
  height: 68rpx;
  width: 220rpx;
}

.add-card__row {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.add-card__name {
  flex: 1;
}

.add-card__btn {
  min-width: 150rpx;
  height: 84rpx;
}

.list-card {
  padding: 0;
  overflow: hidden;
}

.list-card__head {
  padding: 22rpx 28rpx;
  border-bottom: 2rpx solid $c-border;
}

.list-card__head--expense {
  background-color: $c-expense-bg;
}

.list-card__head--income {
  background-color: $c-income-bg;
}

.list-card__title {
  font-size: $fs-sm;
  font-weight: 500;
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

.cat-list {
  display: flex;
  flex-direction: column;
}

.cat-item {
  padding: 20rpx 28rpx;
  border-bottom: 2rpx solid $c-border;
}

.cat-item:last-child {
  border-bottom: none;
}

.cat-item__row {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.cat-item__icon {
  font-size: 40rpx;
  line-height: 1;
}

.cat-item__name {
  flex: 1;
  min-width: 0;
  font-size: $fs-sm;
  color: $c-text;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.cat-item__actions {
  flex-shrink: 0;
  display: flex;
  gap: 24rpx;
}

.cat-item__action {
  font-size: $fs-xs;
  color: $c-primary;
}

.cat-item__action--danger {
  color: $c-expense;
}

.cat-edit {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
}

.cat-edit__row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.cat-edit__name {
  flex: 1;
}
</style>
