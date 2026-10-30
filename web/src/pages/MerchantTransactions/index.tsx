import { useParams, useNavigate, useLocation } from 'react-router-dom'
import MerchantTransactionsList from '../../components/MerchantTransactionsList'

interface LocationState {
  merchantName?: string
  startDate?: string
  endDate?: string
}

export default function MerchantTransactions() {
  const { merchantId } = useParams<{ merchantId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const state = (location.state ?? {}) as LocationState

  const merchantName = state.merchantName ?? '商户交易记录'
  const id = Number(merchantId)

  return (
    <div className="p-4 md:p-6 max-w-5xl mx-auto">
      <div className="flex items-center gap-3 mb-5">
        <button
          onClick={() => navigate('/statistics')}
          className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors flex-shrink-0 -ml-1"
        >
          <svg className="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <h1 className="text-xl md:text-2xl font-bold text-gray-800 truncate">{merchantName}</h1>
      </div>

      {Number.isFinite(id) && id > 0 ? (
        <MerchantTransactionsList merchantId={id} startDate={state.startDate} endDate={state.endDate} />
      ) : (
        <div className="text-center py-10 text-gray-400 text-sm">商户信息无效</div>
      )}
    </div>
  )
}
