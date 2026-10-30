// npm run dev:mock에서만 쓰는 가짜 조회 API와 관리자 API. Go 서버 없이 화면을 확인한다.
// 주소와 오류 응답은 specification/40.api/README.md 1.4, 2장, 3장, 5장과 같다.
import type { ServerResponse } from 'node:http'
import type { Plugin } from 'vite'
import { handleAdmin } from './admin.ts'
import { cards, detail } from './data.ts'

function send(res: ServerResponse, status: number, body?: unknown): true {
  res.statusCode = status
  res.setHeader('Content-Type', 'application/json; charset=utf-8')
  res.end(JSON.stringify(body))
  return true
}

export function mockApi(): Plugin {
  return {
    name: 'kedp-mock-api',
    configureServer(server) {
      server.middlewares.use('/api', (req, res, next) => {
        // 미들웨어 안의 req.url은 /api 뒤의 경로다.
        const path = new URL(req.url ?? '/', 'http://mock').pathname
        if (path.startsWith('/admin/')) {
          handleAdmin(req, res, path, send).catch(next)
          return
        }
        if (req.method !== 'GET' && req.method !== 'HEAD') {
          res.setHeader('Allow', 'GET, HEAD')
          return send(res, 405, { error: 'method not allowed' })
        }
        // 화면이 오류를 어떻게 보여주는지 보려고 둔다.
        if (path === '/indicators/fake-500') {
          return send(res, 500, { error: 'internal error' })
        }
        if (path === '/indicators') {
          return send(res, 200, cards())
        }
        const m = /^\/indicators\/([^/]+)$/.exec(path)
        if (m) {
          const d = detail(decodeURIComponent(m[1]))
          return d ? send(res, 200, d) : send(res, 404, { error: 'indicator not found' })
        }
        send(res, 404, { error: 'not found' })
      })
    },
  }
}
