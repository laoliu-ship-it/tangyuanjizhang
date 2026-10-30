import { defineStore } from 'pinia'
import { STORAGE_KEYS } from '@/config'
import { getJSON, setJSON, remove } from '@/utils/storage'

interface AuthSnapshot {
  token: string | null
  userId: number | null
  username: string | null
}

const EMPTY: AuthSnapshot = { token: null, userId: null, username: null }

export const useAuthStore = defineStore('auth', {
  state: (): AuthSnapshot => ({ ...EMPTY }),

  getters: {
    isLoggedIn: (state) => !!state.token,
  },

  actions: {
    /** 从本地缓存恢复（App onLaunch 调用一次） */
    restore() {
      const saved = getJSON<AuthSnapshot>(STORAGE_KEYS.auth, EMPTY)
      this.token = saved.token ?? null
      this.userId = saved.userId ?? null
      this.username = saved.username ?? null
    },

    setAuth(token: string, userId: number, username: string) {
      this.token = token
      this.userId = userId
      this.username = username
      this.persist()
    },

    /** 仅更新用户展示信息，不动 token */
    patchUser(patch: Partial<Pick<AuthSnapshot, 'userId' | 'username'>>) {
      if (patch.userId !== undefined) this.userId = patch.userId
      if (patch.username !== undefined) this.username = patch.username
      this.persist()
    },

    logout() {
      this.token = null
      this.userId = null
      this.username = null
      remove(STORAGE_KEYS.auth)
    },

    /** 把当前状态写回缓存（request 层改 token 后需调用） */
    persist() {
      if (!this.token) return
      setJSON(STORAGE_KEYS.auth, {
        token: this.token,
        userId: this.userId,
        username: this.username,
      } satisfies AuthSnapshot)
    },
  },
})
