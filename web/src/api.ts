// Typed client for the Raptor management API. Every UI action maps to a
// documented /api/v1 endpoint.

export interface Token {
  uuid: string
  alias?: string
  url: string
  default_status: number
  default_content: string
  default_content_type: string
  timeout: number
  cors: boolean
  expiry: number
  actions: boolean
  request_limit: number
  description: string
  listen: number
  redirect: string
  group_id?: string
  premium: boolean
  has_password: boolean
  created_at: string
  updated_at: string
  latest_request_at?: string | null
}

export interface RequestFile {
  id: string
  request_id: string
  filename: string
  content_type: string
  size: number
}

export interface CapturedRequest {
  uuid: string
  token_id: string
  type: string
  method: string
  ip: string
  hostname: string
  user_agent: string
  content: string
  query: Record<string, string[]> | null
  headers: Record<string, string[]> | null
  url: string
  size: number
  sorting: number
  files?: RequestFile[]
  created_at: string
}

export interface RequestPage {
  data: CapturedRequest[]
  total: number
  page: number
  per_page: number
}

export type TokenInput = Partial<{
  alias: string
  default_status: number
  default_content: string
  default_content_type: string
  timeout: number
  cors: boolean
  expiry: number
  request_limit: number
  description: string
  redirect: string
}>

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`
    try {
      const body = await res.json()
      if (body?.error) msg = body.error
    } catch {
      /* ignore */
    }
    throw new Error(msg)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export const api = {
  listTokens: () => req<{ data: Token[] }>('/tokens').then((r) => r.data ?? []),
  createToken: (body: TokenInput = {}) =>
    req<Token>('/tokens', { method: 'POST', body: JSON.stringify(body) }),
  updateToken: (id: string, body: TokenInput) =>
    req<Token>(`/tokens/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteToken: (id: string) => req<void>(`/tokens/${id}`, { method: 'DELETE' }),

  listRequests: (id: string, page = 1, perPage = 50) =>
    req<RequestPage>(`/tokens/${id}/requests?page=${page}&per_page=${perPage}`),
  deleteRequest: (id: string, rid: string) =>
    req<void>(`/tokens/${id}/requests/${rid}`, { method: 'DELETE' }),
  clearRequests: (id: string) =>
    req<{ deleted: number }>(`/tokens/${id}/requests`, { method: 'DELETE' }),

  rawURL: (id: string, rid: string) => `/api/v1/tokens/${id}/requests/${rid}/raw`,
  csvURL: (id: string) => `/api/v1/tokens/${id}/requests.csv`,
  streamURL: (id: string) => `/api/v1/tokens/${id}/stream`,
}
