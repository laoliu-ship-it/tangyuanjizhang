import { useState } from 'react'

type FAQ = {
  id: string
  question: string
  answer: React.ReactNode
}

const faqs: FAQ[] = [
  {
    id: 'add-to-home',
    question: '如何将记账本收藏到手机桌面？',
    answer: <AddToHomeGuide />,
  },
]

function AddToHomeGuide() {
  const [platform, setPlatform] = useState<'ios' | 'android'>('ios')

  return (
    <div>
      {/* 平台切换 */}
      <div className="flex bg-gray-100 rounded-lg p-1 w-fit mb-4">
        <button
          onClick={() => setPlatform('ios')}
          className={`px-4 py-1.5 rounded-md text-sm font-medium transition-colors ${
            platform === 'ios'
              ? 'bg-white text-gray-800 shadow-sm'
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          iOS
        </button>
        <button
          onClick={() => setPlatform('android')}
          className={`px-4 py-1.5 rounded-md text-sm font-medium transition-colors ${
            platform === 'android'
              ? 'bg-white text-gray-800 shadow-sm'
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          Android
        </button>
      </div>

      {platform === 'ios' ? (
        <div className="space-y-4">
          <div className="flex items-start gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-blue-100 text-blue-600 text-xs font-bold flex items-center justify-center">1</span>
            <p className="text-sm text-gray-700">在 iPhone 上打开 <strong>Safari 浏览器</strong>，访问我们的网站</p>
          </div>
          <div className="flex items-start gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-blue-100 text-blue-600 text-xs font-bold flex items-center justify-center">2</span>
            <div className="text-sm text-gray-700">
              点击底部导航栏的 <strong>分享按钮</strong>（方框带向上箭头的图标
              <svg className="inline w-4 h-4 ml-1 text-blue-500" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 3L4 9v12h5v-7h6v7h5V9l-8-6z" />
              </svg>
              ）
            </div>
          </div>
          <div className="flex items-start gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-blue-100 text-blue-600 text-xs font-bold flex items-center justify-center">3</span>
            <p className="text-sm text-gray-700">在弹出的菜单中向下滑动，找到并点击 <strong>"添加到主屏幕"</strong></p>
          </div>
          <div className="flex items-start gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-blue-100 text-blue-600 text-xs font-bold flex items-center justify-center">4</span>
            <p className="text-sm text-gray-700">确认后，桌面就会出现记账本的快捷图标，以后点击即可快速打开</p>
          </div>

          {/* iOS 示意图区域 */}
          <div className="mt-4 p-4 bg-gray-50 rounded-xl border border-dashed border-gray-300">
            <p className="text-xs text-gray-400 mb-2">操作示意：</p>
            <div className="flex gap-4 justify-center">
              <div className="text-center">
                <div className="w-24 h-40 bg-white rounded-2xl shadow-sm border border-gray-200 flex flex-col items-center justify-center mx-auto">
                  <svg className="w-8 h-8 text-blue-500 mb-1" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 3L4 9v12h5v-7h6v7h5V9l-8-6z" />
                  </svg>
                  <span className="text-[10px] text-gray-400">分享</span>
                </div>
                <span className="text-[10px] text-gray-400 mt-1">点击分享</span>
              </div>
              <div className="text-center">
                <div className="w-24 h-40 bg-white rounded-2xl shadow-sm border border-gray-200 flex flex-col items-center justify-center mx-auto">
                  <svg className="w-6 h-6 text-green-500 mb-1" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <rect x="5" y="2" width="14" height="20" rx="2" />
                    <path d="M12 18v-6M9 15l3 3 3-3" />
                  </svg>
                  <span className="text-[10px] text-gray-400">添加到主屏幕</span>
                </div>
                <span className="text-[10px] text-gray-400 mt-1">添加到主屏幕</span>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="flex items-start gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-green-100 text-green-600 text-xs font-bold flex items-center justify-center">1</span>
            <p className="text-sm text-gray-700">在 Android 手机上打开 <strong>Chrome 浏览器</strong>，访问我们的网站</p>
          </div>
          <div className="flex items-start gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-green-100 text-green-600 text-xs font-bold flex items-center justify-center">2</span>
            <div className="text-sm text-gray-700">
              点击右上角的 <strong>菜单按钮</strong>（三个竖点的图标 ⋮）
            </div>
          </div>
          <div className="flex items-start gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-green-100 text-green-600 text-xs font-bold flex items-center justify-center">3</span>
            <p className="text-sm text-gray-700">在弹出的菜单中点击 <strong>"添加到主屏幕"</strong>（部分版本可能显示"安装应用"）</p>
          </div>
          <div className="flex items-start gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-green-100 text-green-600 text-xs font-bold flex items-center justify-center">4</span>
            <p className="text-sm text-gray-700">确认后，桌面就会出现记账本的快捷图标，以后点击即可快速打开</p>
          </div>

          {/* Android 示意图区域 */}
          <div className="mt-4 p-4 bg-gray-50 rounded-xl border border-dashed border-gray-300">
            <p className="text-xs text-gray-400 mb-2">操作示意：</p>
            <div className="flex gap-4 justify-center">
              <div className="text-center">
                <div className="w-24 h-40 bg-white rounded-2xl shadow-sm border border-gray-200 flex flex-col items-center justify-center mx-auto">
                  <span className="text-2xl text-gray-600 mb-1">⋮</span>
                  <span className="text-[10px] text-gray-400">菜单</span>
                </div>
                <span className="text-[10px] text-gray-400 mt-1">点击菜单</span>
              </div>
              <div className="text-center">
                <div className="w-24 h-40 bg-white rounded-2xl shadow-sm border border-gray-200 flex flex-col items-center justify-center mx-auto">
                  <svg className="w-6 h-6 text-green-500 mb-1" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <rect x="5" y="2" width="14" height="20" rx="2" />
                    <path d="M12 18v-6M9 15l3 3 3-3" />
                  </svg>
                  <span className="text-[10px] text-gray-400">添加到主屏幕</span>
                </div>
                <span className="text-[10px] text-gray-400 mt-1">添加到主屏幕</span>
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="mt-4 p-3 bg-blue-50 rounded-lg">
        <p className="text-xs text-blue-600">
          <strong>提示：</strong>添加到桌面后，打开应用就像使用原生 App 一样方便，没有浏览器地址栏干扰。
        </p>
      </div>
    </div>
  )
}

function FAQItem({ faq }: { faq: FAQ }) {
  const [open, setOpen] = useState(false)

  return (
    <div className="border-b border-gray-100 last:border-0">
      <button
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between py-4 text-left"
      >
        <span className="text-sm font-medium text-gray-800">{faq.question}</span>
        <svg
          className={`w-5 h-5 text-gray-400 transition-transform ${open ? 'rotate-180' : ''}`}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <path d="M6 9l6 6 6-6" />
        </svg>
      </button>
      {open && (
        <div className="pb-4 text-gray-600">
          {faq.answer}
        </div>
      )}
    </div>
  )
}

export default function Help() {
  return (
    <div className="p-4 md:p-6 max-w-3xl mx-auto">
      <h1 className="text-xl md:text-2xl font-bold text-gray-800 mb-5">帮助中心</h1>

      <div className="bg-white rounded-2xl p-4 md:p-5 shadow-sm border border-gray-100">
        <h2 className="text-sm font-semibold text-gray-700 mb-3">常见问题</h2>
        {faqs.map((faq) => (
          <FAQItem key={faq.id} faq={faq} />
        ))}
      </div>
    </div>
  )
}
