import { request } from './request'
import type { AuthResult, DemoConfig, WeChatLoginResult } from './types'

export const authApi = {
  /** POST /auth/register */
  register(data: {
    username: string
    email: string
    password: string
    book_type: string
  }): Promise<AuthResult> {
    return request<AuthResult>({ url: '/auth/register', method: 'POST', data })
  },

  /** POST /auth/login */
  login(data: { email: string; password: string }): Promise<AuthResult> {
    return request<AuthResult>({ url: '/auth/login', method: 'POST', data })
  },

  /** POST /auth/wechat/login 小程序微信登录（code 由 uni.login 获取） */
  wechatLogin(code: string): Promise<WeChatLoginResult> {
    return request<WeChatLoginResult>({ url: '/auth/wechat/login', method: 'POST', data: { code } })
  },

  /** POST /auth/wechat/bind 首次微信登录时关联已有邮箱账号 */
  wechatBind(data: { bind_token: string; email: string; password: string }): Promise<AuthResult> {
    return request<AuthResult>({ url: '/auth/wechat/bind', method: 'POST', data })
  },

  /** POST /auth/wechat/register 首次微信登录时创建新账号 */
  wechatRegister(data: {
    bind_token: string
    username: string
    book_type: string
  }): Promise<AuthResult> {
    return request<AuthResult>({ url: '/auth/wechat/register', method: 'POST', data })
  },

  /** GET /auth/demo/config 体验入口是否开启 */
  demoConfig(): Promise<DemoConfig> {
    return request<DemoConfig>({ url: '/auth/demo/config' })
  },

  /** POST /auth/demo/login 固定演示账号短期登录（只读） */
  demoLogin(): Promise<AuthResult> {
    return request<AuthResult>({ url: '/auth/demo/login', method: 'POST', data: {} })
  },
}
