import { request } from './request'
import type { AuthResult } from './types'

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
}
