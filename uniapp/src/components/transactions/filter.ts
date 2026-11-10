export type SortBy = 'transaction_date' | 'created_at'
export type FilterType = '' | 'income' | 'expense'

export interface TransactionFilter {
  sortBy: SortBy
  type: FilterType
  categoryId: number
  merchantId: number
  merchantName: string
  startDate: string
  endDate: string
}

export function createEmptyFilter(): TransactionFilter {
  return {
    sortBy: 'transaction_date',
    type: '',
    categoryId: 0,
    merchantId: 0,
    merchantName: '',
    startDate: '',
    endDate: '',
  }
}
