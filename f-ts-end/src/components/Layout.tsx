import type { ReactNode } from 'react'
import { Link } from 'react-router'

export function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto max-w-5xl px-4 py-4">
          <Link to="/" className="inline-block">
            <span className="block text-xl font-bold tracking-tight">KEDP</span>
            <span className="block text-xs text-slate-500">Korea Economic Data Platform</span>
          </Link>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-6 sm:py-8">{children}</main>
    </div>
  )
}

/** 불러오는 중, 오류, 없음 같은 짧은 안내. */
export function Message({ children }: { children: ReactNode }) {
  return <p className="py-12 text-center text-slate-500">{children}</p>
}

export function NotFound({ what = '페이지' }: { what?: string }) {
  return (
    <div className="py-12 text-center">
      <p className="text-slate-600">{what}를 찾을 수 없습니다.</p>
      <Link to="/" className="mt-4 inline-block text-blue-600 hover:underline">
        메인 화면으로
      </Link>
    </div>
  )
}
