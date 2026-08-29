/* Everything comes from the agent. It is the process that is always up and the
 * one that knows where the indexer is listening. */

async function call(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: options.body ? { 'Content-Type': 'application/json' } : undefined,
  })
  const text = await response.text()
  const body = text ? JSON.parse(text) : null
  if (!response.ok) throw new Error(body?.error || `the agent answered ${response.status}`)
  return body
}

const query = (values) =>
  new URLSearchParams(Object.entries(values).filter(([, value]) => value !== '' && value !== false))

export const api = {
  status: () => call('/health'),

  repositories: () => call('/api/repositories'),
  addRepository: (path, name) =>
    call('/api/repositories', { method: 'POST', body: JSON.stringify({ path, name }) }),
  dropRepository: (path) =>
    call(`/api/repositories?${query({ path })}`, { method: 'DELETE' }),
  /* Update reads only what changed, which is what a watcher wants. Somebody
   * pressing a button for this means read it again: how a file is read changes
   * with the indexer, and an update pass sees an unchanged file and skips it. */
  index: (repository = '', { everything = false, update = false } = {}) =>
    call('/api/index', {
      method: 'POST',
      body: JSON.stringify({ repository, everything, update }),
    }),

  graph: (repository, { includeTests = false, pathPrefix = '' } = {}) =>
    call(`/api/graph?${query({ repository, include_tests: includeTests, path_prefix: pathPrefix })}`),

  knowledge: (repository) => call(`/api/knowledge?${query({ repository, limit: 100 })}`),
  recordKnowledge: (item) =>
    call('/api/knowledge', { method: 'PUT', body: JSON.stringify(item) }),
  forgetKnowledge: (repository, id) =>
    call(`/api/knowledge?${query({ repository, id })}`, { method: 'DELETE' }),

  browse: (path = '') => call(`/api/browse?${query({ path })}`),

  /* Reading what a repository already states. Asking without recording is the
   * safe half, so a person can see what would be written before it is. */
  initialize: (repository, dryRun = false) =>
    call('/api/knowledge/initialize', {
      method: 'POST',
      body: JSON.stringify({ repository, dry_run: dryRun }),
    }),

  settings: () => call('/api/settings'),
  setSetting: (key, value) =>
    call('/api/settings', { method: 'PUT', body: JSON.stringify({ key, value }) }),
}
