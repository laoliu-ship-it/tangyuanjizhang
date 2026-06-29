import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { authApi } from '../../services/api'
import { useAuthStore } from '../../store/auth'
import { useTenantStore } from '../../store/tenant'

type BookType = 'restaurant' | 'family' | 'startup'

const bookTypeOptions = [
  { value: 'restaurant' as BookType, label: '饭店', emoji: '🍽️', desc: '餐厅经营记账' },
  { value: 'family' as BookType,     label: '家庭', emoji: '🏠', desc: '家庭收支管理' },
  { value: 'startup' as BookType,    label: '创业', emoji: '🚀', desc: '初创企业财务' },
]

export default function Register() {
  const navigate = useNavigate()
  const { setAuth } = useAuthStore()
  const { setTenants } = useTenantStore()

  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [bookType, setBookType] = useState<BookType>('restaurant')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (password !== confirmPassword) {
      alert('两次输入的密码不一致')
      return
    }
    if (password.length < 6) {
      alert('密码至少需要 6 个字符')
      return
    }
    setLoading(true)
    try {
      const res = await authApi.register({ username, email, password, book_type: bookType })
      const { token, user_id, username: uname, tenants } = res.data.data
      setAuth(token, user_id, uname)
      setTenants((tenants ?? []).map((t: { id: number; name: string }) => ({ id: t.id, name: t.name })))
      navigate('/', { replace: true })
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : '注册失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-md p-8">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-blue-600">汤圆记账</h1>
          <p className="text-gray-500 mt-2 text-sm">创建您的账号</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">用户名</label>
            <input
              type="text"
              value={username}
              onChange={e => setUsername(e.target.value)}
              placeholder="请输入用户名"
              required
              className="w-full px-4 py-3 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">邮箱</label>
            <input
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              placeholder="请输入邮箱"
              required
              className="w-full px-4 py-3 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">密码</label>
            <input
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder="至少 6 个字符"
              required
              minLength={6}
              className="w-full px-4 py-3 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">确认密码</label>
            <input
              type="password"
              value={confirmPassword}
              onChange={e => setConfirmPassword(e.target.value)}
              placeholder="再次输入密码"
              required
              className="w-full px-4 py-3 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">账本类型</label>
            <div className="grid grid-cols-3 gap-2">
              {bookTypeOptions.map(opt => (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => setBookType(opt.value)}
                  className={`flex flex-col items-center p-3 rounded-xl border-2 transition-colors text-sm cursor-pointer ${
                    bookType === opt.value
                      ? 'border-blue-500 bg-blue-50 text-blue-700'
                      : 'border-gray-200 hover:border-gray-300 text-gray-600'
                  }`}
                >
                  <span className="text-2xl mb-1">{opt.emoji}</span>
                  <span className="font-medium">{opt.label}</span>
                  <span className="text-xs text-gray-400 mt-0.5">{opt.desc}</span>
                </button>
              ))}
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white font-semibold rounded-xl transition-colors text-sm"
          >
            {loading ? '注册中...' : '注册'}
          </button>
        </form>

        <p className="text-center text-sm text-gray-500 mt-6">
          已有账号？
          <Link to="/login" className="text-blue-600 hover:underline font-medium ml-1">
            立即登录
          </Link>
        </p>
      </div>
    </div>
  )
}
