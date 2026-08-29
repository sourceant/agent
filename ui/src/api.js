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

  /* Where recent change has landed on what the rest of the code leans on.
   * Either fact alone says little; it is the overlap that is worth a person's
   * time, and is also the shortest list of files worth reading first. */
  attention: (repository) => call(`/api/attention?${query({ repository })}`),

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
  /* Written where this product owns the folder: a repository, so the team gets
   * it by pulling, or the machine, for what somebody wants everywhere. What
   * sits in a coding agent's own folders is read and never written. */
  recordSkill: (skill) =>
    call('/api/skills', {
      method: 'PUT',
      body: JSON.stringify({ scope: 'repository', paths: [], reviews: null, ...skill }),
    }),
  forgetSkill: (repository, scope, id) =>
    call(`/api/skills?${query({ repository, scope, id })}`, { method: 'DELETE' }),

  /* Reading a checkout's own work before anybody else has been asked to. Asking
   * without a model is the free half: what changed and what bears on it, with
   * nothing judged.
   *
   * The agent runs it and holds the answer, because a review takes tens of
   * seconds and a connection held open that long loses work when anything
   * interrupts it. */
  startReview: (repository, { against = '', title = '', description = '', skills = [], useModel = true } = {}) =>
    call('/api/reviews', {
      method: 'POST',
      body: JSON.stringify({ repository, against, title, description, skills, use_model: useModel }),
    }),
  reviewed: (id) => call(`/api/reviews/${id}`),
  reviews: (repository = '') => call(`/api/reviews?${query({ repository })}`),

  settings: () => call('/api/settings'),
  setSetting: (key, value) =>
    call('/api/settings', { method: 'PUT', body: JSON.stringify({ key, value }) }),
  resetSetting: (key) => call(`/api/settings?${query({ key })}`, { method: 'DELETE' }),
}
