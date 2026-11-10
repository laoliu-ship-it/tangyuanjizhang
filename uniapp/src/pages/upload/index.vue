<script setup lang="ts">
import AppPage from '@/components/AppPage.vue'
import EmptyState from '@/components/EmptyState.vue'

// P2 占位页：独立拍照识别页暂无入口，真实识别链路在记账表单内
</script>

<template>
  <AppPage title="拍照识别" back>
    <EmptyState
      icon="📷"
      text="功能建设中"
      hint="请返回首页，点击「记一笔」上传票据，即可自动 OCR 识别"
    />
  </AppPage>
</template>
