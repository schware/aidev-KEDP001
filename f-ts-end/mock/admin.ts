// npm run dev:mock에서만 쓰는 가짜 관리자 API. 주소, 상태, 응답은 specification/40.api/README.md 5장과 같다.
// 계정과 비밀번호는 확인하지 않는다: 둘 다 넣으면 로그인된다. 계정이 `fail`이면 로그인 실패(401)다.
// MOCK_ADMIN=forbidden으로 띄우면 모든 관리자 API가 403(허용 IP 아님)이다.
import { randomBytes } from 'node:crypto'
import type { IncomingMessage, ServerResponse } from 'node:http'
import type { AdminIndicator, SourceInfo } from '../src/admin/api.ts'
import { fakes, sorted, type Fake } from './data.ts'

type Send = (res: ServerResponse, status: number, body?: unknown) => true

const sessions = new Map<string, { account: string; expires: number }>()
const cookieName = 'kedp_admin_session'
let collectTime = '06:00'

// 40.api 5.9 응답 예시
const sources: SourceInfo[] = [
  { code: 'data-go-kr', name: '공공데이터포털', key_set: true, last_collected_at: null, status: 'unchecked' },
  { code: 'ecos', name: 'ECOS', key_set: true, last_collected_at: '2026-10-25T06:00:03+09:00', status: 'connected' },
  { code: 'kosis', name: 'KOSIS', key_set: false, last_collected_at: '2026-10-25T06:00:05+09:00', status: 'no_key' },
]

function toAdmin(f: Fake): AdminIndicator {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { series, seriesCycle, last_collected_at, ...a } = f
  return a
}

const list = () => ({ indicators: sorted().map(toAdmin) })

/** 서울 시각 RFC 3339. */
function seoulTime(ms: number): string {
  return new Date(ms + 9 * 3600_000).toISOString().slice(0, 19) + '+09:00'
}

function sessionOf(req: IncomingMessage) {
  const m = new RegExp(`(?:^|;\\s*)${cookieName}=([^;]+)`).exec(req.headers.cookie ?? '')
  const s = m ? sessions.get(m[1]) : undefined
  return s && s.expires > Date.now() ? { token: m![1], ...s } : undefined
}

async function readBody(req: IncomingMessage): Promise<unknown> {
  const chunks: Buffer[] = []
  for await (const c of req) chunks.push(c as Buffer)
  const text = Buffer.concat(chunks).toString('utf8')
  if (text === '') return undefined
  try {
    return JSON.parse(text)
  } catch {
    return Symbol.for('invalid')
  }
}

type Body = Record<string, unknown>
const editKeys = ['name', 'cycle', 'unit', 'chart_period', 'decimals', 'notice', 'related']
const addKeys = [...editKeys, 'id', 'source', 'stat_code', 'item_code', 'classes', 'org_id', 'value_field', 'visible']
const isStr = (v: unknown): v is string => typeof v === 'string'
const nonEmpty = (v: unknown) => isStr(v) && v.trim() !== ''
const optStr = (v: unknown) => v === undefined || v === null || isStr(v)

/** 5.5, 5.6의 칸 검사. 틀린 칸을 모두 돌려준다. */
function validate(b: Body, keys: string[], selfId: string, required: string[]): Record<string, string> {
  const e: Record<string, string> = {}
  for (const k of Object.keys(b)) if (!keys.includes(k)) e[k] = 'unknown field'
  for (const k of required) if (b[k] === undefined) e[k] = 'required'
  if ('name' in b && !nonEmpty(b.name)) e.name = 'must not be empty'
  if ('cycle' in b && !['D', 'M', 'Q', 'A'].includes(b.cycle as string)) e.cycle = 'must be one of D, M, Q, A'
  if ('chart_period' in b && !(isStr(b.chart_period) && /^[1-9]\d{0,3} (years|months|days)$/.test(b.chart_period)))
    e.chart_period = 'must be N years, N months or N days (N 1-9999)'
  if (b.decimals !== undefined && b.decimals !== null && !(Number.isInteger(b.decimals) && (b.decimals as number) >= 0 && (b.decimals as number) <= 4))
    e.decimals = 'must be 0-4 or null'
  for (const k of ['unit', 'notice']) if (!optStr(b[k])) e[k] = 'must be a string or null'
  if (b.related !== undefined) {
    const r = b.related
    if (!Array.isArray(r) || !r.every(isStr)) e.related = 'must be an array of ids'
    else if (r.includes(selfId)) e.related = 'must not include itself'
    else if (new Set(r).size !== r.length) e.related = 'must not have duplicates'
    else if (r.some((id) => !fakes.some((f) => f.id === id))) e.related = 'unknown indicator id'
  }
  return e
}

function validateAdd(b: Body): Record<string, string> {
  const e = validate(b, addKeys, String(b.id), ['id', 'name', 'source', 'stat_code', 'cycle', 'chart_period'])
  if ('id' in b && !(isStr(b.id) && /^[a-z0-9]+(-[a-z0-9]+)*$/.test(b.id))) e.id = 'must be lowercase letters and digits joined by -'
  if ('source' in b && !['ecos', 'kosis', 'data-go-kr'].includes(b.source as string)) e.source = 'must be one of ecos, kosis, data-go-kr'
  if ('stat_code' in b && !nonEmpty(b.stat_code)) e.stat_code = 'must not be empty'
  if ((b.source === 'ecos' || b.source === 'kosis') && !nonEmpty(b.item_code)) e.item_code = 'required for ecos, kosis'
  if (b.source === 'data-go-kr' && b.item_code != null && b.item_code !== '') e.item_code = 'must be empty for data-go-kr'
  if (b.source === 'kosis' && !nonEmpty(b.org_id)) e.org_id = 'required for kosis'
  if (b.source === 'data-go-kr' && !nonEmpty(b.value_field)) e.value_field = 'required for data-go-kr'
  if (b.classes !== undefined && !(Array.isArray(b.classes) && b.classes.every(nonEmpty))) e.classes = 'must be an array of non-empty codes'
  if (b.visible !== undefined && typeof b.visible !== 'boolean') e.visible = 'must be true or false'
  return e
}

const strOrNull = (v: unknown) => (isStr(v) && v.trim() !== '' ? v.trim() : null)
const nextOrder = () => Math.max(0, ...fakes.map((f) => f.order ?? 0)) + 1

function applyEdit(f: Fake, b: Body) {
  f.name = (b.name as string).trim()
  f.cycle = b.cycle as Fake['cycle']
  f.unit = strOrNull(b.unit)
  f.chart_period = b.chart_period as string
  f.decimals = (b.decimals as number | null | undefined) ?? null
  f.notice = strOrNull(b.notice)
  f.related = b.related as string[]
  if (f.order === null) {
    f.order = nextOrder()
    f.visible = false
  }
}

/** /api/admin 아래 요청. 처리했으면 true. path는 /api 뒤의 경로다. */
export async function handleAdmin(req: IncomingMessage, res: ServerResponse, path: string, send: Send): Promise<boolean> {
  if (!path.startsWith('/admin/')) return false
  res.setHeader('Cache-Control', 'no-store')
  const method = req.method ?? 'GET'
  const route = path.slice('/admin'.length)

  if (process.env.MOCK_ADMIN === 'forbidden') {
    return send(res, 403, { error: 'forbidden' })
  }

  let body: unknown
  if (method === 'POST' || method === 'PUT') {
    if (!(req.headers['content-type'] ?? '').startsWith('application/json')) {
      return send(res, 415, { error: 'content type must be application/json' })
    }
    body = await readBody(req)
  }
  const isObj = (v: unknown): v is Body => typeof v === 'object' && v !== null && !Array.isArray(v)

  if (route === '/login' && method === 'POST') {
    if (!isObj(body)) return send(res, 400, { error: 'invalid JSON' })
    if (!nonEmpty(body.account) || !nonEmpty(body.password)) {
      return send(res, 400, { error: 'invalid input', fields: { account: 'required', password: 'required' } })
    }
    if (body.account === 'fail') return send(res, 401, { error: 'login failed' })
    const old = sessionOf(req)
    if (old) sessions.delete(old.token)
    const token = randomBytes(32).toString('hex')
    const expires = Date.now() + 8 * 3600_000
    sessions.set(token, { account: body.account as string, expires })
    res.setHeader('Set-Cookie', `${cookieName}=${token}; Path=/api/admin; HttpOnly; SameSite=Strict; Max-Age=28800`)
    return send(res, 200, { account: body.account, expires_at: seoulTime(expires) })
  }

  const s = sessionOf(req)
  if (!s) return send(res, 401, { error: 'login required' })

  const notAllowed = (allow: string) => {
    res.setHeader('Allow', allow)
    return send(res, 405, { error: 'method not allowed' })
  }

  if (route === '/logout') {
    if (method !== 'POST') return notAllowed('POST')
    sessions.delete(s.token)
    res.setHeader('Set-Cookie', `${cookieName}=; Path=/api/admin; HttpOnly; SameSite=Strict; Max-Age=0`)
    res.statusCode = 204
    res.end()
    return true
  }
  if (route === '/session') {
    if (method !== 'GET') return notAllowed('GET')
    return send(res, 200, { account: s.account, expires_at: seoulTime(s.expires) })
  }
  if (route === '/indicators') {
    if (method === 'GET') return send(res, 200, list())
    if (method !== 'POST') return notAllowed('GET, POST')
    if (!isObj(body)) return send(res, 400, { error: 'invalid JSON' })
    const e = validateAdd(body)
    if (Object.keys(e).length > 0) return send(res, 400, { error: 'invalid input', fields: e })
    if (fakes.some((f) => f.id === body.id)) return send(res, 409, { error: 'indicator already exists' })
    const f: Fake = {
      id: body.id as string,
      name: '',
      source: body.source as Fake['source'],
      stat_code: (body.stat_code as string).trim(),
      item_code: strOrNull(body.item_code),
      classes: (body.classes as string[] | undefined) ?? [],
      org_id: strOrNull(body.org_id),
      value_field: strOrNull(body.value_field),
      cycle: 'M',
      unit: null,
      source_unit: null,
      visible: (body.visible as boolean | undefined) ?? false,
      order: nextOrder(),
      chart_period: null,
      decimals: null,
      notice: null,
      related: [],
      last_collected_at: null,
    }
    applyEdit(f, { related: [], ...body })
    fakes.push(f)
    return send(res, 201, toAdmin(f))
  }
  const m = /^\/indicators\/([^/]+)$/.exec(route)
  if (m) {
    if (method !== 'PUT') return notAllowed('PUT')
    const f = fakes.find((x) => x.id === decodeURIComponent(m[1]))
    if (!f) return send(res, 404, { error: 'indicator not found' })
    if (!isObj(body)) return send(res, 400, { error: 'invalid JSON' })
    const e = validate(body, editKeys, f.id, ['name', 'cycle', 'chart_period', 'related'])
    if (Object.keys(e).length > 0) return send(res, 400, { error: 'invalid input', fields: e })
    applyEdit(f, body)
    return send(res, 200, toAdmin(f))
  }
  if (route === '/display-order') {
    if (method !== 'PUT') return notAllowed('PUT')
    const items = isObj(body) ? body.indicators : undefined
    const configured = fakes.filter((f) => f.order !== null).map((f) => f.id)
    const ok =
      Array.isArray(items) &&
      items.every((x) => isObj(x) && isStr(x.id) && typeof x.visible === 'boolean' && Number.isInteger(x.order)) &&
      new Set(items.map((x) => x.id)).size === items.length &&
      items.length === configured.length &&
      items.every((x) => configured.includes(x.id))
    if (!ok) return send(res, 400, { error: 'invalid input', fields: { indicators: 'must list every configured indicator once' } })
    for (const x of items as { id: string; visible: boolean; order: number }[]) {
      const f = fakes.find((y) => y.id === x.id)!
      f.visible = x.visible
      f.order = x.order
    }
    return send(res, 200, list())
  }
  if (route === '/collect-time') {
    if (method === 'GET') return send(res, 200, { collect_time: collectTime })
    if (method !== 'PUT') return notAllowed('GET, PUT')
    const t = isObj(body) ? body.collect_time : undefined
    if (!(isStr(t) && /^([01]\d|2[0-3]):[0-5]\d$/.test(t))) {
      return send(res, 400, { error: 'invalid input', fields: { collect_time: 'must be HH:MM (00:00-23:59)' } })
    }
    collectTime = t
    return send(res, 200, { collect_time: collectTime })
  }
  if (route === '/sources') {
    if (method !== 'GET') return notAllowed('GET')
    return send(res, 200, { sources })
  }
  return send(res, 404, { error: 'not found' })
}
