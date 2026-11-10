import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
import ts from 'typescript'

const sourcePath = new URL('../src/utils/billingCapability.ts', import.meta.url)
const source = fs.readFileSync(sourcePath, 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: {
    module: ts.ModuleKind.ESNext,
    target: ts.ScriptTarget.ES2020,
  },
}).outputText
const tempPath = path.join(os.tmpdir(), `billing-capability-${process.pid}-${Date.now()}.mjs`)
fs.writeFileSync(tempPath, compiled)
const { canPurchaseBilling, isIOSWeChatSupported } = await import(pathToFileURL(tempPath).href)
fs.unlinkSync(tempPath)

const base = {
  enabled: true,
  platform: 'android',
  iapEnabled: false,
  wechatVersion: '8.0.70',
  supportsVirtualPayment: true,
  isAdmin: true,
  isDemo: false,
  packageCount: 1,
}

test('Android + IAP 未开启也可以购买', () => {
  assert.equal(canPurchaseBilling({ ...base, platform: 'android', iapEnabled: false }), true)
})

test('iOS + IAP 未开启不可购买', () => {
  assert.equal(canPurchaseBilling({ ...base, platform: 'ios', iapEnabled: false }), false)
})

test('iOS 已开 IAP 且微信版本 >= 8.0.68 可购买', () => {
  assert.equal(canPurchaseBilling({ ...base, platform: 'ios', iapEnabled: true, wechatVersion: '8.0.68' }), true)
  assert.equal(isIOSWeChatSupported('8.0.68'), true)
  assert.equal(isIOSWeChatSupported('8.0.67'), false)
})

test('iOS 已开 IAP 但微信版本过低不可购买', () => {
  assert.equal(canPurchaseBilling({ ...base, platform: 'ios', iapEnabled: true, wechatVersion: '8.0.60' }), false)
})

test('viewer / demo / disabled / 无支付能力 / 无套餐不可购买', () => {
  assert.equal(canPurchaseBilling({ ...base, isAdmin: false }), false)
  assert.equal(canPurchaseBilling({ ...base, isDemo: true }), false)
  assert.equal(canPurchaseBilling({ ...base, enabled: false }), false)
  assert.equal(canPurchaseBilling({ ...base, supportsVirtualPayment: false }), false)
  assert.equal(canPurchaseBilling({ ...base, packageCount: 0 }), false)
})
