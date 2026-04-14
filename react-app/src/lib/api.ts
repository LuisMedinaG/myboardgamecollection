import type { Game } from '../types/game'

const BASE = '/api/v1'

// ── Token storage ──────────────────────────────────────────────────────────────
const ACCESS_KEY  = 'mbgc_access'
const REFRESH_KEY = 'mbgc_refresh'

export const tokens = {
  getAccess:  () => localStorage.getItem(ACCESS_KEY),
  getRefresh: () => localStorage.getItem(REFRESH_KEY),
  setAccess:  (t: string) => localStorage.setItem(ACCESS_KEY, t),
  set(access: string, refresh: string) {
    localStorage.setItem(ACCESS_KEY, access)
    localStorage.setItem(REFRESH_KEY, refresh)
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}

// ── Error ──────────────────────────────────────────────────────────────────────
export class ApiError extends Error {
  readonly status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

// ── Core fetch with auto-refresh ───────────────────────────────────────────────
let refreshPromise: Promise<void> | null = null

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  retry = true,
): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = tokens.getAccess()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(BASE + path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  // Auto-refresh on 401 — coalesce concurrent requests into one refresh attempt
  if (res.status === 401 && retry) {
    const refresh = tokens.getRefresh()
    if (!refresh) throw new ApiError(401, 'unauthorized')

    if (!refreshPromise) {
      refreshPromise = request<{ data: { access_token: string } }>(
        'POST', '/auth/refresh', { refresh_token: refresh }, false,
      ).then(r => {
        tokens.setAccess(r.data.access_token)
      }).finally(() => {
        refreshPromise = null
      })
    }
    await refreshPromise
    return request<T>(method, path, body, false)
  }

  if (res.status === 204) return undefined as T

  const json = await res.json()
  if (!res.ok) throw new ApiError(res.status, json.error ?? 'request failed')
  return json as T
}

// ── API response types (Go → snake_case) ──────────────────────────────────────
interface VibeAPI     { id: number; name: string }
interface PlayerAidAPI { id: number; game_id: number; filename: string; label: string }

interface GameAPI {
  id:                   number
  bgg_id:               number
  name:                 string
  description:          string
  year_published:       number
  image:                string
  thumbnail:            string
  min_players:          number
  max_players:          number
  play_time:            number
  categories:           string[]
  mechanics:            string[]
  types:                string[]
  weight:               number
  rating:               number
  language_dependence:  number
  recommended_players:  number[]
  rules_url:            string
  vibes:                VibeAPI[]
  player_aids?:         PlayerAidAPI[]
}

// ── Mapper: snake_case API → camelCase Game ────────────────────────────────────
function mapGame(g: GameAPI): Game {
  return {
    id:                 g.id,
    bggId:              g.bgg_id,
    name:               g.name,
    description:        g.description,
    yearPublished:      g.year_published,
    image:              g.image,
    thumbnail:          g.thumbnail,
    minPlayers:         g.min_players,
    maxPlayers:         g.max_players,
    playTime:           g.play_time,
    categories:         g.categories         ?? [],
    mechanics:          g.mechanics          ?? [],
    types:              g.types              ?? [],
    weight:             g.weight,
    rating:             g.rating,
    languageDependence: g.language_dependence,
    recommendedPlayers: g.recommended_players ?? [],
    rulesUrl:           g.rules_url,
    vibes:              (g.vibes ?? []).map(v => v.name),
  }
}

// ── Public types ───────────────────────────────────────────────────────────────
export interface Vibe {
  id: number
  name: string
}

export interface PlayerAid {
  id: number
  gameId: number
  filename: string
  label: string
}

export interface GameDetail extends Game {
  playerAids: PlayerAid[]
}

export interface GamesListParams {
  q?:           string
  category?:    string
  players?:     string
  playtime?:    string
  weight?:      string
  rating?:      string
  lang?:        string
  rec_players?: string
  page?:        number
  limit?:       number
}

export interface GamesListResponse {
  data:       Game[]
  total:      number
  page:       number
  per_page:   number
  categories: string[]
}

export interface DiscoverResponse {
  data:  Game[]
  total: number
  vibe:  Vibe
}

export interface DiscoverParams {
  vibe_id:      number
  type?:        string
  category?:    string
  mechanic?:    string
  players?:     string
  playtime?:    string
  weight?:      string
  rating?:      string
  lang?:        string
  rec_players?: string
}

// ── API methods ────────────────────────────────────────────────────────────────
export const api = {
  // Auth
  async login(username: string, password: string) {
    const r = await request<{ data: { access_token: string; refresh_token: string; expires_in: number } }>(
      'POST', '/auth/login', { username, password },
    )
    tokens.set(r.data.access_token, r.data.refresh_token)
    return r.data
  },

  async logout() {
    const refresh = tokens.getRefresh()
    if (refresh) {
      await request('POST', '/auth/logout', { refresh_token: refresh }).catch(() => {})
    }
    tokens.clear()
  },

  async ping() {
    const r = await request<{ data: { pong: boolean; username: string } }>('GET', '/ping')
    return r.data
  },

  // Games
  async listGames(params: GamesListParams = {}): Promise<GamesListResponse> {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== '') qs.set(k, String(v))
    }
    const suffix = qs.size ? `?${qs}` : ''
    const r = await request<{ data: GameAPI[]; total: number; page: number; per_page: number; categories: string[] }>(
      'GET', `/games${suffix}`,
    )
    return { ...r, data: r.data.map(mapGame) }
  },

  async getGame(id: number): Promise<GameDetail> {
    const r = await request<{ data: GameAPI & { player_aids: PlayerAidAPI[] } }>('GET', `/games/${id}`)
    return {
      ...mapGame(r.data),
      playerAids: (r.data.player_aids ?? []).map(a => ({
        id:       a.id,
        gameId:   a.game_id,
        filename: a.filename,
        label:    a.label,
      })),
    }
  },

  async deleteGame(id: number) {
    return request('DELETE', `/games/${id}`)
  },

  async setGameVibes(gameId: number, vibeIds: number[]) {
    return request<{ data: { game_id: number; vibe_ids: number[] } }>(
      'POST', `/games/${gameId}/vibes`, { vibe_ids: vibeIds },
    )
  },

  async bulkVibes(gameIds: number[], vibeIds: number[]) {
    return request<{ data: { updated: number } }>(
      'POST', '/games/bulk-vibes', { game_ids: gameIds, vibe_ids: vibeIds },
    )
  },

  // Vibes
  async listVibes(): Promise<Vibe[]> {
    const r = await request<{ data: VibeAPI[] }>('GET', '/vibes')
    return r.data
  },

  async createVibe(name: string): Promise<Vibe> {
    const r = await request<{ data: VibeAPI }>('POST', '/vibes', { name })
    return r.data
  },

  async updateVibe(id: number, name: string): Promise<Vibe> {
    const r = await request<{ data: VibeAPI }>('PUT', `/vibes/${id}`, { name })
    return r.data
  },

  async deleteVibe(id: number) {
    return request('DELETE', `/vibes/${id}`)
  },

  // Discover
  async discover(params: DiscoverParams): Promise<DiscoverResponse> {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== '') qs.set(k, String(v))
    }
    const r = await request<{ data: GameAPI[]; total: number; vibe: VibeAPI }>(
      'GET', `/discover?${qs}`,
    )
    return { data: r.data.map(mapGame), total: r.total, vibe: r.vibe }
  },

  // Profile
  async getProfile() {
    const r = await request<{ data: { username: string; bgg_username: string } }>('GET', '/profile')
    return r.data
  },

  async setBGGUsername(bggUsername: string) {
    const r = await request<{ data: { bgg_username: string } }>(
      'PUT', '/profile/bgg-username', { bgg_username: bggUsername },
    )
    return r.data
  },

  async changePassword(currentPassword: string, newPassword: string) {
    return request<void>('PUT', '/profile/password', {
      current_password: currentPassword,
      new_password:     newPassword,
    })
  },
}
