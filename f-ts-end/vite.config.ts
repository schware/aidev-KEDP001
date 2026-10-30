import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { mockApi } from './mock/plugin.ts'

// 개발 서버는 /api를 Go 조회 API로 넘긴다(specification/40.api 1장).
// --mode mock이면 Go 대신 mock/의 가짜 데이터를 돌려준다.
export default defineConfig(({ mode }) => ({
  plugins: [react(), tailwindcss(), ...(mode === 'mock' ? [mockApi()] : [])],
  // 상세 화면 조각은 ECharts(선 차트만 넣음) 때문에 500 kB를 조금 넘는다(gzip 약 180 kB).
  build: { chunkSizeWarningLimit: 600 },
  server: {
    host: '127.0.0.1',
    proxy: mode === 'mock' ? undefined : { '/api': { target: 'http://127.0.0.1:8080', xfwd: true } },
  },
}))
