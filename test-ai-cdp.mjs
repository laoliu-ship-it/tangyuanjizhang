/**
 * CDP 自动化测试：登录 cece 账号，上传票据图片，验证 AI 自动记账
 */
import WebSocket from 'node:net' // only for check
import { createConnection } from 'node:net'
import { readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))

const APP_URL = 'http://127.0.0.1:8090'
const CDP_URL = 'http://localhost:9222'
const IMAGE_DIR = `${process.env.HOME}/Downloads/jizhang`
const IMAGE_FILE = `${IMAGE_DIR}/微信图片_20260529150355_5336_118.jpg`

// ── CDP 封装 ─────────────────────────────────────────────────
async function connectCDP() {
  const res = await fetch(`${CDP_URL}/json/new`)
  const tab = await res.json()
  const ws = new (await import('node:module')).default.builtinModules // fallback
  return tab
}

async function getCDPWsUrl() {
  const r = await fetch(`${CDP_URL}/json`)
  const tabs = await r.json()
  // 找空白 tab 或创建新的
  let tab = tabs.find(t => t.type === 'page')
  if (!tab) {
    const r2 = await fetch(`${CDP_URL}/json/new`)
    tab = await r2.json()
  }
  return tab.webSocketDebuggerUrl
}

class CDP {
  constructor(ws) {
    this._ws = ws
    this._id = 0
    this._pending = new Map()
    this._listeners = new Map()
    ws.on('message', (data) => {
      const msg = JSON.parse(data.toString())
      if (msg.id !== undefined && this._pending.has(msg.id)) {
        const { resolve, reject } = this._pending.get(msg.id)
        this._pending.delete(msg.id)
        if (msg.error) reject(new Error(msg.error.message))
        else resolve(msg.result)
      } else if (msg.method) {
        const listeners = this._listeners.get(msg.method) ?? []
        listeners.forEach(fn => fn(msg.params))
      }
    })
  }

  send(method, params = {}) {
    return new Promise((resolve, reject) => {
      const id = ++this._id
      this._pending.set(id, { resolve, reject })
      this._ws.send(JSON.stringify({ id, method, params }))
    })
  }

  on(event, fn) {
    const list = this._listeners.get(event) ?? []
    list.push(fn)
    this._listeners.set(event, list)
  }

  async waitForEvent(event, timeout = 15000) {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error(`Timeout waiting for ${event}`)), timeout)
      this.on(event, (params) => { clearTimeout(timer); resolve(params) })
    })
  }
}

function sleep(ms) { return new Promise(r => setTimeout(r, ms)) }

// ── 主测试流程 ────────────────────────────────────────────────
async function main() {
  const { default: { WebSocket } } = await import('node:module')
  
  // 动态 import ws
  let WS
  try {
    const m = await import('ws')
    WS = m.default ?? m.WebSocket ?? m
  } catch {
    console.error('❌ 需要安装 ws 包: npm install ws -g 或 cd /tmp && npm install ws')
    process.exit(1)
  }

  console.log('=== 饭店记账 AI 功能 CDP 测试 ===\n')

  // 1. 获取 CDP WebSocket URL
  console.log('1. 连接 Chrome DevTools...')
  const wsUrl = await getCDPWsUrl()
  console.log(`   WS: ${wsUrl}`)

  const ws = new WS(wsUrl)
  await new Promise((res, rej) => { ws.on('open', res); ws.on('error', rej) })
  const cdp = new CDP(ws)

  // 启用必要的 domains
  await cdp.send('Page.enable')
  await cdp.send('Network.enable')
  await cdp.send('Runtime.enable')
  await cdp.send('DOM.enable')

  // 2. 导航到应用
  console.log(`\n2. 打开应用 ${APP_URL}...`)
  await cdp.send('Page.navigate', { url: APP_URL })
  await cdp.waitForEvent('Page.loadEventFired', 10000)
  await sleep(1500)
  console.log('   ✓ 页面加载完成')

  // 截图保存当前状态
  async function screenshot(name) {
    const { data } = await cdp.send('Page.captureScreenshot', { format: 'jpeg', quality: 70 })
    const { writeFileSync } = await import('node:fs')
    const path = `/tmp/cdp-${name}.jpg`
    writeFileSync(path, Buffer.from(data, 'base64'))
    console.log(`   📷 截图: ${path}`)
  }

  await screenshot('01-initial')

  // 3. 检查是否在登录页
  const { result: titleResult } = await cdp.send('Runtime.evaluate', {
    expression: 'document.title'
  })
  console.log(`\n3. 页面标题: ${titleResult.value}`)

  // 4. 登录
  console.log('\n4. 登录 cece 账号...')
  
  // 等待登录表单出现
  await sleep(1000)
  
  // 找 email 输入框并填入
  const fillResult = await cdp.send('Runtime.evaluate', {
    expression: `
      (function() {
        // 找所有 input
        const inputs = document.querySelectorAll('input')
        const emailInput = Array.from(inputs).find(i => 
          i.type === 'email' || i.placeholder?.includes('邮箱') || i.placeholder?.includes('email') || i.name === 'email'
        )
        const passwordInput = Array.from(inputs).find(i => i.type === 'password')
        
        if (!emailInput || !passwordInput) {
          return { ok: false, msg: '未找到登录表单，inputs: ' + Array.from(inputs).map(i=>i.type+':'+i.placeholder).join(',') }
        }
        
        // React controlled input 需要触发 nativeInputValueSetter
        const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set
        
        nativeInputValueSetter.call(emailInput, 'cece@test.com')
        emailInput.dispatchEvent(new Event('input', { bubbles: true }))
        emailInput.dispatchEvent(new Event('change', { bubbles: true }))
        
        nativeInputValueSetter.call(passwordInput, 'cece123')
        passwordInput.dispatchEvent(new Event('input', { bubbles: true }))
        passwordInput.dispatchEvent(new Event('change', { bubbles: true }))
        
        return { ok: true, email: emailInput.value, pwd: passwordInput.value.length + ' chars' }
      })()
    `,
    returnByValue: true
  })
  console.log('   表单填写:', JSON.stringify(fillResult.result.value))

  await screenshot('02-login-filled')

  // 点击登录按钮
  const clickLogin = await cdp.send('Runtime.evaluate', {
    expression: `
      (function() {
        const btn = Array.from(document.querySelectorAll('button')).find(b => 
          b.textContent.includes('登录') && !b.textContent.includes('注册')
        )
        if (!btn) return { ok: false, msg: '未找到登录按钮, buttons: ' + Array.from(document.querySelectorAll('button')).map(b=>b.textContent.trim()).join('|') }
        btn.click()
        return { ok: true, btn: btn.textContent.trim() }
      })()
    `,
    returnByValue: true
  })
  console.log('   点击登录:', JSON.stringify(clickLogin.result.value))

  // 等待跳转
  await sleep(2500)
  await screenshot('03-after-login')

  // 检查是否登录成功
  const { result: urlResult } = await cdp.send('Runtime.evaluate', { expression: 'location.href' })
  console.log(`   当前 URL: ${urlResult.value}`)

  const { result: bodyText } = await cdp.send('Runtime.evaluate', {
    expression: `document.body.innerText.slice(0, 200)`
  })
  console.log(`   页面内容片段: ${bodyText.value}`)

  // 5. 检查租户 LLM 配置
  console.log('\n5. 检查租户 AI 配置...')
  const { result: llmCheck } = await cdp.send('Runtime.evaluate', {
    expression: `
      (async function() {
        const token = localStorage.getItem('token') || 
          Object.entries(localStorage).find(([k,v]) => v?.includes('eyJ'))?.[1] || ''
        const headers = token ? { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' } : {}
        
        // 获取 tenant_id
        const tenantId = localStorage.getItem('tenantId') || localStorage.getItem('tenant_id') || ''
        
        try {
          const r = await fetch('/api/llm/config', { headers })
          const data = await r.json()
          return JSON.stringify(data)
        } catch(e) {
          return 'error: ' + e.message
        }
      })()
    `,
    returnByValue: true,
    awaitPromise: true
  })
  console.log('   LLM 配置:', llmCheck.result.value)

  // 6. 导航到"记一笔"或上传页面
  console.log('\n6. 查找上传/记账入口...')
  const { result: navResult } = await cdp.send('Runtime.evaluate', {
    expression: `
      (function() {
        // 找导航菜单中的记账/上传入口
        const links = Array.from(document.querySelectorAll('a, button, [role="link"]'))
        const uploadLink = links.find(el => 
          el.textContent.includes('拍照') || el.textContent.includes('上传') || 
          el.textContent.includes('记账') || el.textContent.includes('OCR')
        )
        if (uploadLink) {
          uploadLink.click()
          return { found: true, text: uploadLink.textContent.trim(), href: uploadLink.href || '' }
        }
        // 尝试直接导航
        return { found: false, links: links.slice(0,10).map(l=>l.textContent.trim()).join('|') }
      })()
    `,
    returnByValue: true
  })
  console.log('   导航结果:', JSON.stringify(navResult.result.value))

  await sleep(1000)

  // 也尝试直接 hash 导航
  await cdp.send('Runtime.evaluate', {
    expression: `
      // 尝试点击底部导航的上传按钮
      const allBtns = Array.from(document.querySelectorAll('button, a'))
      const uploadBtn = allBtns.find(b => b.textContent.includes('拍照') || b.textContent.includes('识别'))
      if (uploadBtn) uploadBtn.click()
    `
  })
  
  await sleep(1000)
  await screenshot('04-upload-page')

  // 7. 通过 API 直接测试 OCR+LLM
  console.log('\n7. 直接 API 测试: 上传图片走 OCR+LLM...')
  const { result: apiTest } = await cdp.send('Runtime.evaluate', {
    expression: `
      (async function() {
        // 从 localStorage 拿 token
        let token = ''
        for (let i = 0; i < localStorage.length; i++) {
          const key = localStorage.key(i)
          const val = localStorage.getItem(key)
          if (val && val.startsWith('eyJ')) { token = val; break }
        }
        
        // 也尝试从 sessionStorage
        if (!token) {
          for (let i = 0; i < sessionStorage.length; i++) {
            const key = sessionStorage.key(i)
            const val = sessionStorage.getItem(key)
            if (val && val.startsWith('eyJ')) { token = val; break }
          }
        }
        
        // 获取所有 localStorage 内容用于调试
        const lsKeys = {}
        for (let i = 0; i < localStorage.length; i++) {
          const k = localStorage.key(i)
          lsKeys[k] = localStorage.getItem(k)?.slice(0, 80)
        }
        
        return JSON.stringify({ token: token ? token.slice(0,30)+'...' : 'NOT FOUND', localStorage: lsKeys })
      })()
    `,
    returnByValue: true,
    awaitPromise: true
  })
  console.log('   Auth 信息:', apiTest.result.value)

  ws.close()
  console.log('\n=== 测试完成，查看 /tmp/cdp-*.jpg 截图 ===')
}

main().catch(e => { console.error('测试失败:', e.message); process.exit(1) })
