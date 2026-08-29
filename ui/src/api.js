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
  initialize: (repository, { dryRun = false, useModel = false } = {}) =>
    call('/api/knowledge/initialize', {
      method: 'POST',
      body: JSON.stringify({ repository, dry_run: dryRun, use_model: useModel }),
    }),

  /* The rules a team already wrote down for whatever reads their code, from
   * this machine's agent folders and from the repository's own. */
  skills: (repository = '') => call(`/api/skills?${query({ repository })}`),
  skill: (id, repository = '') => call(`/api/skills/${id}?${query({ repository })}`),
  /* Only a repository's own are written. What somebody keeps in their agent
   * folders is theirs, and the core refuses to write there. */
  recordSkill: (skill) => call('/api/skills', { method: 'PUT', body: JSON.stringify(skill) }),
  forgetSkill: (repository, id) =>
    call(`/api/skills?${query({ repository, id })}`, { method: 'DELETE' }),

  /* Reading a checkout's own work before anybody else has been asked to. Asking
   * without a model is the free half: what changed and which rules bear on it,
   * with nothing judged. */
  review: (repository, { against = '', title = '', description = '', skills = [], useModel = true } = {}) =>
    call('/api/reviews', {
      method: 'POST',
      body: JSON.stringify({ repository, against, title, description, skills, use_model: useModel }),
    }),

  settings: () => call('/api/settings'),
  setSetting: (key, value) =>
    call('/api/settings', { method: 'PUT', body: JSON.stringify({ key, value }) }),
  resetSetting: (key) => call(`/api/settings?${query({ key })}`, { method: 'DELETE' }),
}
