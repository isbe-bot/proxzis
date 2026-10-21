async function request(path, options) {
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

function query(path, params) {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params || {})) {
    if (value !== undefined && value !== null && value !== '' && value !== 'all') search.set(key, String(value))
  }
  const suffix = search.toString()
  return request(suffix ? `${path}?${suffix}` : path)
}

export const listBots = params => query('/v1/bots', params)
export const getBot = id => request(`/v1/bots/${encodeURIComponent(id)}`)
export const listMatches = params => query('/v1/matches', params)
export const getMatch = id => request(`/v1/matches/${encodeURIComponent(id)}`)
export const getExhibition = () => request('/v1/exhibition')

export function createBot(bot) {
  return request('/v1/bots', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(bot),
  })
}
