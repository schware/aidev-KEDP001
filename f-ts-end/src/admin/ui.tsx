// 관리자 화면에서 같이 쓰는 작은 조각.
import type { ReactNode } from 'react'

export const inputClass =
  'w-full rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm focus:border-blue-500 focus:outline-none disabled:bg-slate-100'
export const buttonClass =
  'rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50'
export const primaryButtonClass =
  'rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50'

/** 허용 IP가 아닐 때(403) 이 글 하나만 보여 준다. */
export function Forbidden() {
  return (
    <div className="min-h-screen bg-slate-50 py-24 text-center text-slate-700">
      <title>KEDP 관리자</title>
      허용된 주소가 아닙니다
    </div>
  )
}

/** 입력칸 하나와 이름, 그 옆(아래)의 오류. 서버 400 fields도 같은 자리에 보인다. */
export function Field({
  label,
  error,
  hint,
  children,
}: {
  label: string
  error?: string
  hint?: string
  children: ReactNode
}) {
  return (
    <label className="block text-sm">
      <span className="mb-1 block font-medium text-slate-700">{label}</span>
      {children}
      {hint && !error && <span className="mt-1 block text-xs text-slate-500">{hint}</span>}
      {error && (
        <span role="alert" className="mt-1 block text-xs text-red-600">
          {error}
        </span>
      )}
    </label>
  )
}

/** 입력칸 자리가 없는 오류(모르는 칸 등)와 일반 오류. */
export function FormErrors({ errors, known, message }: { errors: Record<string, string>; known: string[]; message?: string }) {
  const rest = Object.entries(errors).filter(([k]) => !known.includes(k))
  if (!message && rest.length === 0) {
    return null
  }
  return (
    <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
      {message && <p>{message}</p>}
      {rest.map(([k, v]) => (
        <p key={k}>
          {k}: {v}
        </p>
      ))}
    </div>
  )
}
