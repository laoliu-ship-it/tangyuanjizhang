import { computed, type ComputedRef } from 'vue'
import { useMetaStore, type TenantRoleKey } from '@/store/meta'

export interface UseRoleReturn {
  role: ComputedRef<TenantRoleKey>
  canEdit: ComputedRef<boolean>
  load: () => Promise<void>
}

/** 当前账本角色：owner / admin 可编辑，member 只读 */
export function useRole(): UseRoleReturn {
  const meta = useMetaStore()
  const role = computed<TenantRoleKey>(() => meta.role)
  const canEdit = computed(() => meta.role === 'owner' || meta.role === 'admin')

  async function load(): Promise<void> {
    try {
      await meta.ensureRole()
    } catch (e) {
      // 角色获取失败按只读处理，避免误放权
      meta.setRole('member')
    }
  }

  return { role, canEdit, load }
}
