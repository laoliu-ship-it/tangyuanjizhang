<script setup lang="ts">
import { onLaunch, onError } from '@dcloudio/uni-app'
import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'

onLaunch(() => {
  // 同步恢复登录态（storage 读取是同步 API，首屏即可用）
  useAuthStore().restore()
  useTenantStore().restore()
})

onError((err) => {
  console.error('[全局错误]', err)
})
</script>

<style lang="scss">
@import '@/styles/common.scss';

page {
  background-color: $c-bg;
  color: $c-text;
  font-size: $fs;
  font-family: -apple-system, BlinkMacSystemFont, 'Helvetica Neue', 'PingFang SC',
    'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
  -webkit-font-smoothing: antialiased;
}

view,
text,
scroll-view,
input,
textarea,
button {
  box-sizing: border-box;
}

/* 去掉 uni button 默认样式 */
button {
  padding: 0;
  margin: 0;
  background: transparent;
  line-height: normal;
  font-size: inherit;
  color: inherit;
}

button::after {
  border: none;
}
</style>
