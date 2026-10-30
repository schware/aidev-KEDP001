import { createContext, useContext } from 'react'
import type { Session } from './api.ts'

export interface AdminAuth {
  session: Session
  /** 오류가 401이면 로그인 화면으로, 403이면 "허용된 주소가 아닙니다"로 바꾸고 true를 돌려준다. */
  handleAuthError: (err: unknown) => boolean
}

export const AdminAuthContext = createContext<AdminAuth | null>(null)

/** 관리자 화면(AdminLayout 안)에서 로그인 상태와 401/403 처리를 쓴다. */
export function useAdminAuth(): AdminAuth {
  const auth = useContext(AdminAuthContext)
  if (!auth) {
    throw new Error('useAdminAuth must be used inside AdminLayout')
  }
  return auth
}
