import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'

// 全局错误捕获 — 打印完整堆栈方便定位线上问题
window.addEventListener('error', (e) => {
  console.error('[全局错误]', e.message, e.filename, 'line', e.lineno, '\n', e.error?.stack)
})
window.addEventListener('unhandledrejection', (e) => {
  console.error('[未捕获的 Promise 拒绝]', e.reason?.stack || e.reason)
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
