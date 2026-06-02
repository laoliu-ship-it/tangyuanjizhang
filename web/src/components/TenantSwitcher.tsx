import { useTenantStore } from '../store/tenant'

export default function TenantSwitcher() {
  const { tenants, currentTenantId, switchTenant } = useTenantStore()

  if (tenants.length === 0) return null

  const currentTenant = tenants.find(t => t.id === currentTenantId)

  return (
    <div className="relative">
      <select
        value={currentTenantId ?? ''}
        onChange={e => switchTenant(Number(e.target.value))}
        className="appearance-none bg-white border border-gray-300 rounded-lg px-3 py-1.5 pr-8 text-sm font-medium text-gray-700 cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
      >
        {tenants.map(t => (
          <option key={t.id} value={t.id}>
            {t.name}
          </option>
        ))}
      </select>
      <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-gray-500">
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </div>
      {currentTenant && (
        <span className="sr-only">当前租户：{currentTenant.name}</span>
      )}
    </div>
  )
}
