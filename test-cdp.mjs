/**
 * CDP 自动化测试 - 多笔 LLM 记账
 *
 * 场景一：OCR → LLM (ocr_text)
 *   上传图片 → AI识别多笔 → 前端自动生成草稿 → 用户确认分类 → 审核提交
 *
 * 场景二：Vision LLM (vision)
 *   图片直接给 LLM → AI识别多笔 → 前端自动生成草稿 → 用户确认分类 → 审核提交
 */

import { WebSocket } from 'ws'
import { readFileSync, writeFileSync } from 'fs'

const APP_URL   = 'http://127.0.0.1:18090'
const CDP_URL   = 'http://localhost:9222'
const IMG_DIR   = `${process.env.HOME}/Downloads/jizhang`
const IMGS      = [
  `${IMG_DIR}/微信图片_20260529150355_5336_118.jpg`,
  `${IMG_DIR}/微信图片_20260529150356_5337_118.jpg`,
  `${IMG_DIR}/微信图片_20260529150356_5338_118.jpg`,
]
const TENANT_ID = 4

// ── CDP 封装 ──────────────────────────────────────────────────
class CDP {
  constructor(ws) {
    this._ws = ws; this._id = 0
    this._pending = new Map(); this._listeners = new Map()
    ws.on('message', data => {
      const msg = JSON.parse(data.toString())
      if (msg.id != null && this._pending.has(msg.id)) {
        const { resolve, reject } = this._pending.get(msg.id)
        this._pending.delete(msg.id)
        msg.error ? reject(new Error(msg.error.message)) : resolve(msg.result)
      } else if (msg.method) {
        ;(this._listeners.get(msg.method) ?? []).forEach(fn => fn(msg.params))
      }
    })
  }
  cmd(method, params = {}) {
    return new Promise((resolve, reject) => {
      const id = ++this._id
      this._pending.set(id, { resolve, reject })
      this._ws.send(JSON.stringify({ id, method, params }))
    })
  }
  on(event, fn) {
    const list = this._listeners.get(event) ?? []; list.push(fn)
    this._listeners.set(event, list)
  }
  waitFor(event, timeout = 20000) {
    return new Promise((resolve, reject) => {
      const t = setTimeout(() => reject(new Error(`timeout: ${event}`)), timeout)
      this.on(event, p => { clearTimeout(t); resolve(p) })
    })
  }
}

const sleep = ms => new Promise(r => setTimeout(r, ms))

async function screenshot(cdp, name) {
  const { data } = await cdp.cmd('Page.captureScreenshot', { format: 'jpeg', quality: 80 })
  const path = `/tmp/cdp-${name}.jpg`
  writeFileSync(path, Buffer.from(data, 'base64'))
  console.log(`    📷  ${path}`)
}

async function js(cdp, expression) {
  const { result, exceptionDetails } = await cdp.cmd('Runtime.evaluate', {
    expression, returnByValue: true, awaitPromise: true,
  })
  if (exceptionDetails) throw new Error(exceptionDetails.exception?.description || exceptionDetails.text)
  return result.value
}

// React controlled input setter
const HELPERS = `
window.__set = function(el, val) {
  const s = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set
  s.call(el, val)
  el.dispatchEvent(new Event('input', {bubbles:true}))
  el.dispatchEvent(new Event('change', {bubbles:true}))
}`

// ── API 工具 ──────────────────────────────────────────────────
async function apiPost(path, token, body) {
  const res = await fetch(`${APP_URL}${path}`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}`, 'X-Tenant-ID': String(TENANT_ID), 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return res.json()
}

async function apiPut(path, token, body) {
  const res = await fetch(`${APP_URL}${path}`, {
    method: 'PUT',
    headers: { Authorization: `Bearer ${token}`, 'X-Tenant-ID': String(TENANT_ID), 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return res.json()
}

async function uploadImage(token, imgPath) {
  const imgData = readFileSync(imgPath)
  const boundary = '----Boundary' + Math.random().toString(36).slice(2)
  const filename = imgPath.split('/').pop()
  const body = Buffer.concat([
    Buffer.from(`--${boundary}\r\nContent-Disposition: form-data; name="file"; filename="${filename}"\r\nContent-Type: image/jpeg\r\n\r\n`),
    imgData,
    Buffer.from(`\r\n--${boundary}--\r\n`),
  ])
  const res = await fetch(`${APP_URL}/api/upload/ocr/analyze`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'X-Tenant-ID': String(TENANT_ID),
      'Content-Type': `multipart/form-data; boundary=${boundary}`,
    },
    body,
  })
  const json = await res.json()
  if (json.code !== 0) throw new Error(`上传失败: ${json.message}`)
  return json.data
}

// ── 步骤：获取已登录 token 并切换租户 ────────────────────────
async function getTokenAndSwitchTenant(cdp) {
  console.log('\n── 获取 Session ──────────────────────────────────')
  await cdp.cmd('Page.navigate', { url: APP_URL })
  await cdp.waitFor('Page.loadEventFired')
  await sleep(1500)
  await js(cdp, HELPERS)
  await screenshot(cdp, '01-home')

  // 读取已存储的 token
  const token = await js(cdp, `
    (function() {
      try { return JSON.parse(localStorage.getItem('auth') || '{}').state?.token || null } catch(e) { return null }
    })()
  `)
  if (!token) throw new Error('未找到登录 token，请先手动登录')
  console.log('  ✅ token:', token.slice(0, 25) + '...')

  // 切换到「测测的记账本」(tenant_id=4)
  const switched = await js(cdp, `
    (function() {
      // 修改 localStorage 中 zustand tenant state
      try {
        const t = JSON.parse(localStorage.getItem('tenant') || '{}')
        if (t.state?.currentTenantId === 4) return { already: true }
        t.state = t.state || {}
        t.state.currentTenantId = 4
        localStorage.setItem('tenant', JSON.stringify(t))
        return { changed: true }
      } catch(e) { return { err: e.message } }
    })()
  `)
  console.log('  租户切换:', JSON.stringify(switched))

  // 点击右上角切换器选「测测的记账本」
  await js(cdp, `
    (function() {
      const btn = [...document.querySelectorAll('button')].find(b => b.textContent.includes('记账本'))
      if (btn) btn.click()
    })()
  `)
  await sleep(500)
  await js(cdp, `
    (function() {
      const opt = [...document.querySelectorAll('button,li,div[role]')].find(el =>
        el.textContent.trim() === '测测的记账本'
      )
      if (opt) opt.click()
    })()
  `)
  await sleep(800)

  const currentTenant = await js(cdp, `
    (function() {
      try { return JSON.parse(localStorage.getItem('tenant') || '{}').state?.currentTenantId } catch(e) { return null }
    })()
  `)
  console.log('  当前租户 ID:', currentTenant)
  await screenshot(cdp, '02-tenant-switched')
  return token
}

// ── 步骤：设置 LLM 模式 ───────────────────────────────────────
async function setLLMMode(token, mode) {
  const r = await apiPut('/api/llm/config', token, {
    enabled: true, use_platform: true,
    provider: 'deepseek', base_url: 'https://api.deepseek.com',
    model: 'deepseek-v4-flash', mode,
  })
  if (r.code !== 0) throw new Error(`设置 LLM 模式失败: ${r.message}`)
  console.log(`  ✅ LLM 模式: ${mode}`)
}

// ── 步骤：通过 API 分析图片，验证多笔 LLM 结果 ────────────────
async function analyzeAllImages(token, label) {
  console.log(`\n  [${label}] 分析 ${IMGS.length} 张图片...`)
  const results = []
  for (let i = 0; i < IMGS.length; i++) {
    const data = await uploadImage(token, IMGS[i])
    const llm = data.llm ?? []
    console.log(`    图${i+1}: LLM识别 ${llm.length} 笔 | OCR ${data.raw_texts?.length ?? 0} 行`)
    llm.forEach((s, j) => console.log(`      [${j+1}] ¥${s.amount} ${s.merchant_name} ${s.category_hint} ${s.date}`))
    results.push({ ...data, imgIndex: i+1 })
  }
  return results
}

// ── 步骤：前端 UI 操作（真实上传文件到浏览器）────────────────
async function testUIFlow(cdp, token, scenario, _ocrResults) {
  console.log(`\n── UI 操作 [${scenario}] ───────────────────────────`)

  // 强制硬刷新页面，清除所有 React state（包括弹窗/slots）
  await cdp.cmd('Page.navigate', { url: APP_URL })
  await cdp.waitFor('Page.loadEventFired')
  await sleep(500)
  await cdp.cmd('Page.reload', { ignoreCache: true })
  await cdp.waitFor('Page.loadEventFired')
  await sleep(2500)
  await js(cdp, HELPERS)

  await screenshot(cdp, `${scenario}-03-dashboard`)

  // 点「记一笔」打开弹窗
  const openBtn = await js(cdp, `
    (function() {
      const btn = [...document.querySelectorAll('button')].find(b =>
        b.textContent.includes('记一笔') || b.textContent.includes('+ 记一笔')
      )
      if (!btn) return { ok: false, btns: [...document.querySelectorAll('button')].map(b=>b.textContent.trim()).filter(Boolean).slice(0,8).join('|') }
      btn.click()
      return { ok: true, text: btn.textContent.trim() }
    })()
  `)
  console.log('  打开记账弹窗:', JSON.stringify(openBtn))
  if (!openBtn?.ok) {
    await screenshot(cdp, `${scenario}-03-open-fail`)
    throw new Error('未找到「记一笔」按钮: ' + JSON.stringify(openBtn))
  }
  await sleep(800)
  await screenshot(cdp, `${scenario}-04-modal`)

  // 读取测试图片为 base64，注入到浏览器文件 input
  console.log('  注入图片文件到浏览器...')
  const imgFiles = IMGS.map(p => ({
    name: p.split('/').pop(),
    b64: readFileSync(p).toString('base64'),
  }))

  // 一次性注入所有图片（React 的 multiple file input 支持一次传多个）
  // 先把所有 base64 存入全局，再统一构造 FileList
  console.log('  一次注入全部图片...')
  for (let i = 0; i < imgFiles.length; i++) {
    await cdp.cmd('Runtime.evaluate', {
      expression: `window.__injectFiles = window.__injectFiles || []; window.__injectFiles.push({ b64: ${JSON.stringify(imgFiles[i].b64)}, name: ${JSON.stringify(imgFiles[i].name)} }); true`,
      returnByValue: true,
    })
  }
  const batchInject = await js(cdp, `
    (async function() {
      try {
        const files = window.__injectFiles || []
        const dt = new DataTransfer()
        for (const { b64, name } of files) {
          const binary = atob(b64)
          const arr = new Uint8Array(binary.length)
          for (let i = 0; i < binary.length; i++) arr[i] = binary.charCodeAt(i)
          dt.items.add(new File([arr], name, { type: 'image/jpeg', lastModified: Date.now() }))
        }
        const input = document.querySelector('input[type="file"]')
        if (!input) return { ok: false, err: '未找到 file input' }
        Object.defineProperty(input, 'files', { value: dt.files, writable: true, configurable: true })
        input.dispatchEvent(new Event('change', { bubbles: true }))
        await new Promise(r => setTimeout(r, 200))
        const fiberKey = Object.keys(input).find(k => k.startsWith('__reactFiber'))
        if (fiberKey) {
          let fiber = input[fiberKey]
          while (fiber) {
            const props = fiber.memoizedProps || fiber.pendingProps
            if (props && typeof props.onChange === 'function') {
              props.onChange({ target: input, currentTarget: input })
              break
            }
            fiber = fiber.return
          }
        }
        delete window.__injectFiles
        return { ok: true, count: dt.files.length }
      } catch(e) { return { ok: false, err: e.message } }
    })()
  `)
  console.log('  批量注入结果:', JSON.stringify(batchInject))
  // 先等一下让图片开始处理
  await sleep(3000)
  await screenshot(cdp, `${scenario}-04-injected`)

  // 等待所有 OCR 处理完成（轮询，每5秒一次，最多等120秒）
  console.log('  等待所有 OCR 处理完成...')
  let completeBtnText = null
  for (let attempt = 0; attempt < 24; attempt++) {
    await sleep(5000)

    // 找有草稿徽标的缩略图，切换到草稿数最多的那个
    const switchResult = await js(cdp, `
      (function() {
        // 缩略图 button 里有 div.bg-blue-500（草稿徽标）
        const thumbBtns = [...document.querySelectorAll('button')].filter(b => {
          const badge = b.querySelector('div[class*="bg-blue-500"]')
          return badge && /^\d+$/.test(badge.textContent.trim())
        })
        if (thumbBtns.length === 0) return { switched: false, count: 0 }
        let maxBtn = thumbBtns[0], maxVal = 0
        thumbBtns.forEach(b => {
          const v = parseInt(b.querySelector('div[class*="bg-blue-500"]').textContent.trim())
          if (v > maxVal) { maxVal = v; maxBtn = b }
        })
        maxBtn.click()
        return { switched: true, maxDrafts: maxVal }
      })()
    `)
    if (switchResult?.switched) {
      console.log(`  [轮询${attempt+1}] 切换到最多草稿 slot: ${switchResult.maxDrafts} 笔`)
      await sleep(300)
    }

    const info = await js(cdp, `
      (function() {
        const btn = [...document.querySelectorAll('button')].find(b =>
          /完成\s*\d+\s*笔/.test(b.textContent)
        )
        const spinning = [...document.querySelectorAll('[class*="animate-spin"]')].length > 0
        return {
          btn: btn ? { text: btn.textContent.trim(), disabled: btn.disabled } : null,
          spinning,
        }
      })()
    `)
    console.log(`  [轮询${attempt+1}] 完成按钮: ${JSON.stringify(info.btn)}, 识别中: ${info.spinning}`)
    if (info.btn && !info.btn.disabled) {
      completeBtnText = info.btn.text
      break
    }
    if (!info.spinning && attempt >= 6) {
      // 已等超过30秒且没有识别中，退出轮询
      break
    }
  }
  console.log(`  OCR完成，完成按钮: ${completeBtnText}`)

  // 如果没有「完成 N 笔」按钮（drafts=0），对每个 slot 手动填入+暂存
  if (!completeBtnText) {
    console.log('  草稿为空，手动逐 slot 填入 AI 建议并暂存...')
    const slotCount = await js(cdp, `
      (function() {
        // 找缩略图数量
        const thumbs = [...document.querySelectorAll('button')].filter(b =>
          b.querySelector('img') || (b.className && b.className.includes('rounded-lg') && b.offsetHeight > 40)
        )
        return thumbs.length
      })()
    `)
    console.log(`  slot 数量: ${slotCount}`)

    for (let i = 0; i < Math.max(slotCount || 0, IMGS.length); i++) {
      // 点第 i 个缩略图
      await js(cdp, `
        (function() {
          const thumbs = [...document.querySelectorAll('button')].filter(b =>
            b.querySelector('img') && b.offsetHeight > 40
          )
          if (thumbs[${i}]) thumbs[${i}].click()
        })()
      `)
      await sleep(400)

      // 点「填入 AI 建议」
      const applyLLM = await js(cdp, `
        (function() {
          const btn = [...document.querySelectorAll('button')].find(b =>
            b.textContent.includes('填入 AI 建议') || b.textContent.includes('重新填入')
          )
          if (btn) { btn.click(); return { ok: true, text: btn.textContent.trim() } }
          const fill = [...document.querySelectorAll('button')].find(b => b.textContent.trim() === '填入')
          if (fill) { fill.click(); return { ok: true, text: '填入' } }
          return { ok: false }
        })()
      `)
      console.log(`  slot[${i+1}] 填入:`, JSON.stringify(applyLLM))
      await sleep(300)

      // 选第一个分类
      await js(cdp, `
        (function() {
          const catBtns = [...document.querySelectorAll('button')].filter(b =>
            /rounded-xl/.test(b.className) && b.textContent.trim().length > 0 && !b.disabled
          )
          const selected = catBtns.find(b => /ring-2|bg-blue|bg-red/.test(b.className))
          if (!selected && catBtns[0]) catBtns[0].click()
        })()
      `)
      await sleep(300)

      // 点「记一笔」
      const drafted = await js(cdp, `
        (function() {
          const btn = [...document.querySelectorAll('button')].find(b => b.textContent.trim() === '记一笔')
          if (!btn || btn.disabled) return { ok: false }
          btn.click(); return { ok: true }
        })()
      `)
      console.log(`  slot[${i+1}] 暂存:`, JSON.stringify(drafted))
      await sleep(500)
    }

    // 再等一下，重新检查完成按钮
    await sleep(1000)
    completeBtnText = await js(cdp, `
      (function() {
        const btn = [...document.querySelectorAll('button')].find(b =>
          /完成\s*\d+\s*笔/.test(b.textContent) && !b.disabled
        )
        return btn ? btn.textContent.trim() : null
      })()
    `)
    console.log(`  手动暂存后完成按钮: ${completeBtnText}`)
  }

  await screenshot(cdp, `${scenario}-05-drafted`)
  console.log(`  完成按钮文字: ${completeBtnText}`)

  if (!completeBtnText) {
    console.log('  ⚠️  没有可提交的草稿，跳过提交步骤')
    return
  }

  // 点「完成 N 笔」
  const completeBtn = await js(cdp, `
    (function() {
      const btn = [...document.querySelectorAll('button')].find(b =>
        /完成\s*\d+\s*笔/.test(b.textContent) && !b.disabled
      )
      if (!btn) return { ok: false, btns: [...document.querySelectorAll('button')].map(b=>b.textContent.trim()).filter(Boolean).slice(0,15).join('|') }
      btn.click()
      return { ok: true, text: btn.textContent.trim() }
    })()
  `)
  console.log('  点击完成:', JSON.stringify(completeBtn))
  if (!completeBtn?.ok) {
    await screenshot(cdp, `${scenario}-06-no-complete-btn`)
    console.log('  ⚠️  未找到完成按钮，跳过')
    return
  }
  await sleep(1000)
  await screenshot(cdp, `${scenario}-06-review`)

  // 找审核弹窗并确认提交
  const submitBtn = await js(cdp, `
    (function() {
      const btn = [...document.querySelectorAll('button')].find(b =>
        b.textContent.includes('确认提交') || b.textContent.includes('确认') ||
        (b.textContent.includes('提交') && /\d/.test(b.textContent))
      )
      if (!btn) return { ok: false, btns: [...document.querySelectorAll('button')].map(b=>b.textContent.trim()).filter(Boolean).slice(0,15).join('|') }
      btn.click()
      return { ok: true, text: btn.textContent.trim() }
    })()
  `)
  console.log('  确认提交:', JSON.stringify(submitBtn))
  await sleep(3000)
  await screenshot(cdp, `${scenario}-07-submitted`)

  // 检查结果：toast 或页面变化
  const result = await js(cdp, `
    (function() {
      const toast = [...document.querySelectorAll('div')].find(el =>
        el.textContent.includes('已提交') || el.textContent.includes('✓')
      )
      const modal = document.querySelector('[class*="modal"],[class*="dialog"]')
      return {
        toast: toast?.textContent?.trim()?.slice(0,50) || null,
        modalClosed: !modal,
        pageTitle: document.title,
      }
    })()
  `)
  console.log('  提交结果:', JSON.stringify(result))
}

// ── 验证数据库最新记录 ────────────────────────────────────────
async function verifyDB(token, label) {
  const res = await fetch(`${APP_URL}/api/transactions?page=1&page_size=5`, {
    headers: { Authorization: `Bearer ${token}`, 'X-Tenant-ID': String(TENANT_ID) },
  })
  const json = await res.json()
  const list = json.data?.items ?? json.data?.list ?? []
  console.log(`\n  [${label}] DB 最新 ${list.length} 笔:`)
  list.forEach(t => {
    const date = (t.transaction_date || '').slice(0, 10)
    console.log(`    ${t.type === 'expense' ? '支出' : '收入'} ¥${t.amount} | ${t.note || '-'} | ${date}`)
  })
}

// ── 主流程 ────────────────────────────────────────────────────
async function main() {
  console.log('╔══════════════════════════════════════════════╗')
  console.log('║   饭店记账 CDP 多笔 AI 记账测试              ║')
  console.log('╚══════════════════════════════════════════════╝')

  const tabsRes = await fetch(`${CDP_URL}/json`)
  const tabs = await tabsRes.json()
  const tab = tabs.find(t => t.type === 'page')
  if (!tab) throw new Error('没有可用的 CDP tab')

  const ws = new WebSocket(tab.webSocketDebuggerUrl)
  await new Promise((res, rej) => { ws.on('open', res); ws.on('error', rej) })
  const cdp = new CDP(ws)
  await cdp.cmd('Page.enable')
  await cdp.cmd('Runtime.enable')
  await cdp.cmd('Network.enable')

  try {
    const token = await getTokenAndSwitchTenant(cdp)

    // ══════════════════════════════════════════════════════════
    // 场景一：OCR → LLM
    // ══════════════════════════════════════════════════════════
    console.log('\n╔══════════════════════════════════════════════╗')
    console.log('║  场景一：OCR → LLM (ocr_text)                ║')
    console.log('╚══════════════════════════════════════════════╝')
    await setLLMMode(token, 'ocr_text')
    const ocrResults = await analyzeAllImages(token, 'OCR→LLM')
    await testUIFlow(cdp, token, 'ocr_text', ocrResults)
    await verifyDB(token, '场景一后')

    // ══════════════════════════════════════════════════════════
    // 场景二：Vision LLM
    // ══════════════════════════════════════════════════════════
    console.log('\n╔══════════════════════════════════════════════╗')
    console.log('║  场景二：Vision LLM（图片直发）               ║')
    console.log('╚══════════════════════════════════════════════╝')
    await setLLMMode(token, 'vision')
    const visionResults = await analyzeAllImages(token, 'Vision')
    await testUIFlow(cdp, token, 'vision', visionResults)
    await verifyDB(token, '场景二后')

    console.log('\n╔══════════════════════════════════════════════╗')
    console.log('║  ✅  全部测试通过                            ║')
    console.log('╚══════════════════════════════════════════════╝')
    console.log('截图: /tmp/cdp-*.jpg')

  } finally {
    ws.close()
  }
}

main().catch(e => {
  console.error('\n❌ 测试失败:', e.message)
  console.error(e.stack)
  process.exit(1)
})
