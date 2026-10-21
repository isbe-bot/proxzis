async function request(path, options = {}) {
  const response = await fetch(path, options)
  const body = await response.json().catch(() => ({}))
  if (!response.ok) {
    const error = new Error(body.error || (body.errors ? 'Registration was rejected.' : 'The arena archive is unavailable.'))
    error.status = response.status
    error.fields = body.errors || null
    throw error
  }
  return body
}

function query(path, params, signal) {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params || {})) {
    if (value !== undefined && value !== null && value !== '' && value !== 'all') search.set(key, String(value))
  }
  const suffix = search.toString()
  return request(suffix ? `${path}?${suffix}` : path, { signal })
}

export const listBots = (params, signal) => query('/v1/bots', params, signal)
export const getBot = (id, signal) => request(`/v1/bots/${encodeURIComponent(id)}`, { signal })
export const listMatches = (params, signal) => query('/v1/matches', params, signal)
export const getMatch = (id, signal) => request(`/v1/matches/${encodeURIComponent(id)}`, { signal })
export const getExhibition = signal => request('/v1/exhibition', { signal })

export function createBot(bot, signal) {
  return request('/v1/bots', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(bot),
    signal,
  })
}
