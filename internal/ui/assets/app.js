/* The local SourceAnt app: what this machine has indexed, and what is known
 * about it. Everything comes from the agent, which is the only thing that
 * knows where the indexer is. */

const view = document.getElementById('view')
const layer = document.getElementById('layer')
const tabs = document.getElementById('tabs')
const themeButton = document.getElementById('theme')

const PAGES = [
  { id: '', label: 'Overview', icon: 'layout' },
  { id: 'repositories', label: 'Repositories', icon: 'boxes' },
  { id: 'graph', label: 'Code graph', icon: 'network' },
  { id: 'knowledge', label: 'Knowledge', icon: 'lightbulb' },
]

const state = {
  page: '',
  repositories: [],
  repository: '',
  status: null,
  graph: null,
  error: '',
}

function escape(text) {
  return String(text ?? '').replace(/[&<>"']/g, (c) =>
    ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c])
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: options.body ? { 'Content-Type': 'application/json' } : undefined,
  })
  const text = await response.text()
  const body = text ? JSON.parse(text) : null
  if (!response.ok) throw new Error(body?.error || `the agent answered ${response.status}`)
  return body
}

/* Rendering */

document.querySelector('.logo-mark').innerHTML = icon('ant', 15)

function renderTabs() {
  tabs.innerHTML = PAGES.map((page) => `
    <a href="#/${page.id}" ${page.id === state.page ? 'aria-current="page"' : ''}>
      ${icon(page.icon, 14)}<span>${page.label}</span>
    </a>`).join('')
}

function head({ tile, iconName, title, sub, actions = '' }) {
  return `
    <div class="page-head">
      <div class="page-title">
        <span class="tile ${tile}">${icon(iconName, 24)}</span>
        <div><h1>${escape(title)}</h1><p class="sub">${escape(sub)}</p></div>
      </div>
      <div class="toolbar-group">${actions}</div>
    </div>`
}

function notice(message, bad = true) {
  return message ? `<p class="notice ${bad ? 'bad' : ''}">${escape(message)}</p>` : ''
}

function needRepository() {
  return `
    <div class="card"><div class="empty">
      <h2>Nothing indexed yet</h2>
      <p>Add a folder and SourceAnt reads it into a graph you can look at and record against.</p>
      <a class="btn" href="#/repositories">${icon('plus', 16)} Add a repository</a>
    </div></div>`
}

function repositoryPicker() {
  if (state.repositories.length < 2) return ''
  return `<select id="pick-repo" aria-label="Repository">${state.repositories.map((repository) =>
    `<option value="${escape(repository.name)}" ${repository.name === state.repository ? 'selected' : ''}>
      ${escape(repository.name)}</option>`).join('')}</select>`
}

/* Overview */

async function overview() {
  view.innerHTML = head({
    tile: 'memory', iconName: 'layout',
    title: 'Overview',
    sub: 'What SourceAnt has on this machine.',
  }) + notice(state.error) + '<div id="body"></div>'

  const body = document.getElementById('body')
  if (state.repositories.length === 0) {
    body.innerHTML = needRepository()
    return
  }

  const counts = await Promise.all(state.repositories.map(async (repository) => {
    const [graph, knowledge] = await Promise.all([
      api(`/api/graph?repository=${encodeURIComponent(repository.name)}`).catch(() => null),
      api(`/api/knowledge?repository=${encodeURIComponent(repository.name)}`).catch(() => null),
    ])
    return {
      repository,
      files: graph ? graph.nodes.filter((n) => (n.labels || []).includes('File')).length : 0,
      nodes: graph ? graph.nodes.length : 0,
      knowledge: knowledge ? knowledge.total : 0,
    }
  }))

  const total = (key) => counts.reduce((sum, item) => sum + item[key], 0)
  body.innerHTML = `
    <div class="stats">
      ${stat('Repositories', state.repositories.length)}
      ${stat('Files', total('files'))}
      ${stat('Nodes', total('nodes'))}
      ${stat('Knowledge', total('knowledge'))}
    </div>
    <div class="grid-cards">${counts.map(({ repository, files, knowledge }) => `
      <div class="card hoverable"><div class="item">
        <span class="item-icon">${icon('folder', 22)}</span>
        <div class="item-body">
          <div class="item-head"><h3>${escape(repository.name)}</h3>
            <span class="badge ${files ? 'success' : 'warning'}">${files ? 'Indexed' : 'Not indexed'}</span>
          </div>
          <p class="item-path">${escape(repository.path)}</p>
          <div class="item-meta">
            <span>${icon('file', 14)} ${files.toLocaleString()} files</span>
            <span>${icon('lightbulb', 14)} ${knowledge.toLocaleString()} recorded</span>
          </div>
        </div>
        <div class="item-actions">
          <a class="btn ghost sm" href="#/graph">Graph</a>
        </div>
      </div></div>`).join('')}
    </div>
    <p class="tight">Reviews are not here. A review reads a pull request, which is a thing the
    hosted service does; nothing on this machine produces one.</p>`
}

function stat(label, value) {
  return `<div class="card stat"><div class="stat-label">${escape(label)}</div>
    <div class="stat-value">${value.toLocaleString()}</div></div>`
}

/* Repositories */

async function repositories() {
  view.innerHTML = head({
    tile: 'graph', iconName: 'boxes',
    title: 'Repositories',
    sub: 'The folders SourceAnt reads on this machine.',
    actions: `<button class="btn" id="add">${icon('plus', 16)} Add a folder</button>`,
  }) + notice(state.error) + '<div id="body"></div>'

  document.getElementById('add').onclick = openPicker
  const body = document.getElementById('body')

  if (state.repositories.length === 0) {
    body.innerHTML = `<div class="card"><div class="empty">
      <h2>No folders yet</h2>
      <p>Point SourceAnt at a repository on this machine and it reads the files into a graph.</p>
      <button class="btn" id="add-empty">${icon('plus', 16)} Add a folder</button>
    </div></div>`
    document.getElementById('add-empty').onclick = openPicker
    return
  }

  body.innerHTML = `<div class="grid-cards">${state.repositories.map((repository) => `
    <div class="card"><div class="item">
      <span class="item-icon">${icon('folder', 22)}</span>
      <div class="item-body">
        <div class="item-head"><h3>${escape(repository.name)}</h3></div>
        <p class="item-path">${escape(repository.path)}</p>
        <div class="item-meta" data-counts="${escape(repository.name)}">
          <span class="muted">Reading…</span>
        </div>
      </div>
      <div class="item-actions">
        <button class="btn outline sm" data-index="${escape(repository.name)}">
          ${icon('refresh', 14)} Re-index</button>
        <button class="btn ghost icon" data-drop="${escape(repository.path)}"
          aria-label="Remove ${escape(repository.name)}">${icon('trash', 16)}</button>
      </div>
    </div></div>`).join('')}</div>`

  for (const button of body.querySelectorAll('[data-index]')) {
    button.onclick = () => reindex(button.dataset.index, button)
  }
  for (const button of body.querySelectorAll('[data-drop]')) {
    button.onclick = () => drop(button.dataset.drop)
  }

  for (const repository of state.repositories) {
    const graph = await api(`/api/graph?repository=${encodeURIComponent(repository.name)}`).catch(() => null)
    const slot = body.querySelector(`[data-counts="${CSS.escape(repository.name)}"]`)
    if (!slot) continue
    const files = graph ? graph.nodes.filter((n) => (n.labels || []).includes('File')).length : 0
    slot.innerHTML = files
      ? `<span>${icon('file', 14)} ${files.toLocaleString()} files</span>
         <span>${icon('link', 14)} ${graph.links.length.toLocaleString()} links</span>`
      : '<span class="muted">Not indexed yet. Re-index to read it.</span>'
  }
}

async function reindex(name, button) {
  const original = button.innerHTML
  button.disabled = true
  button.innerHTML = `${icon('loader', 14, 'spin')} Reading…`
  try {
    await api('/api/index', { method: 'POST', body: JSON.stringify({ repository: name }) })
    state.error = ''
  } catch (error) {
    state.error = error.message
  }
  button.disabled = false
  button.innerHTML = original
  await route()
}

async function drop(path) {
  if (!confirm(`Stop covering ${path}?\n\nWhat was already indexed is left alone.`)) return
  try {
    await api(`/api/repositories?path=${encodeURIComponent(path)}`, { method: 'DELETE' })
    state.error = ''
  } catch (error) {
    state.error = error.message
  }
  await load()
  await route()
}

/* The folder picker.
 *
 * A browser will not tell a page the absolute path of a folder somebody chose,
 * so the agent lists this machine and the page navigates what it lists. */
function openPicker() {
  let here = ''
  const close = () => { layer.innerHTML = '' }

  const show = async (path) => {
    let listing
    try {
      listing = await api(`/api/browse?path=${encodeURIComponent(path || '')}`)
    } catch (error) {
      layer.querySelector('#picker').innerHTML =
        `<p class="empty">${escape(error.message)}</p>`
      return
    }
    here = listing.path
    layer.querySelector('#crumbs').textContent = here
    layer.querySelector('#chosen').textContent = here
    layer.querySelector('#picker').innerHTML = `
      ${listing.parent ? `<button data-go="${escape(listing.parent)}">
        ${icon('chevronUp', 14)} <span class="muted">Up one</span></button>` : ''}
      ${listing.entries.map((entry) => `
        <button data-go="${escape(entry.path)}">
          ${icon('folder', 14)} <span>${escape(entry.name)}</span>
          ${entry.repository ? '<span class="badge glow marker">git</span>' : ''}
        </button>`).join('')}
      ${listing.entries.length === 0 ? '<p class="empty">Nothing inside.</p>' : ''}`
    for (const button of layer.querySelectorAll('[data-go]')) {
      button.onclick = () => show(button.dataset.go)
    }
  }

  layer.innerHTML = `
    <div class="scrim" id="scrim"><div class="modal">
      <h2>Add a folder</h2>
      <p class="crumbs" id="crumbs"></p>
      <div class="picker" id="picker"></div>
      <div class="field-row" style="margin-top:1rem">
        <label class="field" for="repo-name">Name it (optional)</label>
        <input type="text" id="repo-name" placeholder="Taken from the git remote, or the folder name">
      </div>
      <p class="tight">Adding <code id="chosen"></code></p>
      <p class="notice" id="picker-error" hidden></p>
      <div class="modal-actions">
        <button class="btn" id="confirm">${icon('plus', 16)} Add and index</button>
        <button class="btn outline" id="cancel">Cancel</button>
      </div>
    </div></div>`

  layer.querySelector('#cancel').onclick = close
  layer.querySelector('#scrim').onclick = (event) => {
    if (event.target.id === 'scrim') close()
  }
  layer.querySelector('#confirm').onclick = async () => {
    const button = layer.querySelector('#confirm')
    const problem = layer.querySelector('#picker-error')
    button.disabled = true
    button.innerHTML = `${icon('loader', 16, 'spin')} Reading…`
    try {
      await api('/api/repositories', {
        method: 'POST',
        body: JSON.stringify({ path: here, name: layer.querySelector('#repo-name').value.trim() }),
      })
      await api('/api/index', { method: 'POST', body: JSON.stringify({ repository: '', everything: true }) })
      close()
      await load()
      await route()
    } catch (error) {
      problem.hidden = false
      problem.textContent = error.message
      button.disabled = false
      button.innerHTML = `${icon('plus', 16)} Add and index`
    }
  }

  show('')
}

/* Code graph */

let drawing = null

async function graphPage() {
  view.innerHTML = head({
    tile: 'graph', iconName: 'network',
    title: 'Code graph',
    sub: 'Your code, and how it holds together.',
    actions: repositoryPicker(),
  }) + notice(state.error) + '<div id="body" style="display:flex;flex-direction:column;flex:1;min-height:0"></div>'

  const body = document.getElementById('body')
  if (state.repositories.length === 0) {
    body.innerHTML = needRepository()
    return
  }

  body.innerHTML = `
    <div class="toolbar">
      <div class="segmented" id="layouts">${LAYOUTS.map((layout) => `
        <button type="button" data-layout="${layout.id}"
          aria-pressed="${layout.id === 'force'}">${layout.label}</button>`).join('')}</div>
      <div class="toolbar-group">
        <input type="search" id="find" placeholder="Find a file or symbol" aria-label="Find">
        <label class="checkline"><input type="checkbox" id="folders" checked> Folders</label>
        <label class="checkline"><input type="checkbox" id="symbols"> Symbols</label>
        <label class="checkline"><input type="checkbox" id="imports"> Imports</label>
        <label class="checkline"><input type="checkbox" id="tests"> Tests</label>
      </div>
    </div>
    <div class="stage">
      <div id="canvas"></div>
      <div class="overlay" id="overlay">Reading the index…</div>
      <aside class="details" id="details" hidden></aside>
    </div>
    <p class="notice" id="truncated" hidden></p>
    <div class="foot"><div class="legend" id="legend"></div><div class="tally" id="tally"></div></div>`

  const picker = document.getElementById('pick-repo')
  if (picker) picker.onchange = () => { state.repository = picker.value; loadGraph() }

  drawing?.destroy()
  drawing = new CodeGraph(document.getElementById('canvas'), { onSelect: showDetails })

  document.getElementById('layouts').onclick = (event) => {
    const button = event.target.closest('button[data-layout]')
    if (!button) return
    for (const other of document.querySelectorAll('#layouts button')) {
      other.setAttribute('aria-pressed', String(other === button))
    }
    drawing.setLayout(button.dataset.layout)
  }
  document.getElementById('find').oninput = (event) => drawing.highlight(event.target.value)
  for (const id of ['folders', 'symbols', 'imports']) {
    document.getElementById(id).onchange = redraw
  }
  document.getElementById('tests').onchange = loadGraph

  await loadGraph()
}

async function loadGraph() {
  const overlay = document.getElementById('overlay')
  if (!overlay) return
  overlay.hidden = false
  overlay.textContent = 'Reading the index…'
  try {
    const tests = document.getElementById('tests').checked
    state.graph = await api(`/api/graph?repository=${encodeURIComponent(state.repository)}${tests ? '&include_tests=true' : ''}`)
    redraw()
  } catch (error) {
    overlay.textContent = error.message
  }
}

function redraw() {
  if (!state.graph) return
  const keepImports = document.getElementById('imports').checked
  const keepSymbols = document.getElementById('symbols').checked
  const nodes = state.graph.nodes.filter((node) => {
    const group = groupOf(node)
    if (group === 'import') return keepImports
    if (group === 'file') return true
    return keepSymbols
  })
  const kept = new Set(nodes.map((node) => node.id))
  const links = state.graph.links
    .filter((link) => kept.has(link.source) && kept.has(link.target))
    .map((link) => ({ ...link }))

  let data = { nodes: nodes.map((node) => ({ ...node })), links }
  if (document.getElementById('folders').checked) data = withFolders(data, state.repository)

  drawing.show(data)

  const overlay = document.getElementById('overlay')
  overlay.hidden = data.nodes.length > 0
  if (data.nodes.length === 0) {
    overlay.innerHTML = 'Nothing here yet. Re-index it from <a href="#/repositories">Repositories</a>.'
  }
  document.getElementById('tally').textContent =
    `${data.nodes.length.toLocaleString()} nodes · ${data.links.length.toLocaleString()} links`

  const groups = [...new Set(data.nodes.map(groupOf))].sort()
  document.getElementById('legend').innerHTML = groups.map((group) =>
    `<span><i class="swatch" style="background:${COLOURS[group] || OTHER}"></i>${escape(group || 'other')}</span>`).join('')

  const truncated = document.getElementById('truncated')
  truncated.hidden = !state.graph.truncated
  truncated.textContent = 'This repository is larger than the limit, so this is part of it, not all of it.'
}

function showDetails(node) {
  const panel = document.getElementById('details')
  if (!panel) return
  panel.hidden = !node
  if (!node) return
  const degree = state.graph.links.filter((link) =>
    link.source === node.id || link.target === node.id).length
  panel.innerHTML = `
    <button class="btn ghost icon" id="close-details" aria-label="Close">${icon('x', 14)}</button>
    <h2>${escape(node.name)}</h2>
    <dl>
      <dt>Kind</dt><dd>${escape(node.synthetic ? `${node.kind} · this view's arrangement` : node.kind)}</dd>
      <dt>Path</dt><dd>${escape(node.path || '—')}</dd>
      <dt>Links</dt><dd>${node.synthetic ? '—' : degree}</dd>
    </dl>`
  document.getElementById('close-details').onclick = () => drawing.select(null)
}

/* Knowledge */

async function knowledge() {
  view.innerHTML = head({
    tile: 'memory', iconName: 'lightbulb',
    title: 'Knowledge',
    sub: 'The decisions, conventions and constraints behind this code.',
    actions: `${repositoryPicker()}
      <button class="btn" id="record">${icon('plus', 16)} Record something</button>`,
  }) + notice(state.error) + '<div id="body"></div>'

  const body = document.getElementById('body')
  if (state.repositories.length === 0) {
    body.innerHTML = needRepository()
    return
  }

  const picker = document.getElementById('pick-repo')
  if (picker) picker.onchange = () => { state.repository = picker.value; knowledge() }
  document.getElementById('record').onclick = () => openRecord()

  let page
  try {
    page = await api(`/api/knowledge?repository=${encodeURIComponent(state.repository)}&limit=100`)
  } catch (error) {
    body.innerHTML = notice(error.message)
    return
  }

  if (page.items.length === 0) {
    body.innerHTML = `<div class="card"><div class="empty">
      <h2>Nothing recorded yet</h2>
      <p>Why a thing is the way it is outlives the code that does it. Write one down and every
      agent reading this repository over MCP gets it too.</p>
      <button class="btn" id="record-empty">${icon('plus', 16)} Record something</button>
    </div></div>`
    document.getElementById('record-empty').onclick = () => openRecord()
    return
  }

  body.innerHTML = `<div class="grid-cards">${page.items.map((item) => `
    <div class="card"><div class="item">
      <span class="item-icon">${icon('lightbulb', 22)}</span>
      <div class="item-body">
        <div class="item-head">
          <h3>${escape(item.id)}</h3>
          <span class="badge secondary">${escape(item.kind)}</span>
          ${item.status ? `<span class="badge outline">${escape(item.status)}</span>` : ''}
        </div>
        <p class="summary">${escape(item.summary)}</p>
        ${Object.keys(item.properties || {}).length ? `<dl class="props">${
          Object.entries(item.properties).map(([key, value]) =>
            `<dt>${escape(key)}</dt><dd>${escape(typeof value === 'string' ? value : JSON.stringify(value))}</dd>`).join('')
        }</dl>` : ''}
      </div>
      <div class="item-actions">
        <button class="btn ghost icon" data-edit="${escape(item.id)}" aria-label="Edit">${icon('pencil', 16)}</button>
        <button class="btn ghost icon" data-forget="${escape(item.id)}" aria-label="Remove">${icon('trash', 16)}</button>
      </div>
    </div></div>`).join('')}</div>`

  for (const button of body.querySelectorAll('[data-edit]')) {
    button.onclick = () => openRecord(page.items.find((item) => item.id === button.dataset.edit))
  }
  for (const button of body.querySelectorAll('[data-forget]')) {
    button.onclick = () => forget(button.dataset.forget)
  }
}

const KINDS = ['decision', 'convention', 'constraint', 'pattern', 'workaround', 'requirement']

function openRecord(existing) {
  const close = () => { layer.innerHTML = '' }
  layer.innerHTML = `
    <div class="scrim" id="scrim"><div class="modal">
      <h2>${existing ? 'Edit' : 'Record something'}</h2>
      <div class="field-row">
        <label class="field" for="k-id">Name</label>
        <input type="text" id="k-id" placeholder="retry-limit"
          value="${escape(existing?.id || '')}" ${existing ? 'readonly' : ''}>
      </div>
      <div class="field-row">
        <label class="field" for="k-kind">Kind</label>
        <select id="k-kind">${KINDS.map((kind) =>
          `<option ${existing?.kind === kind ? 'selected' : ''}>${kind}</option>`).join('')}</select>
      </div>
      <div class="field-row">
        <label class="field" for="k-summary">What is true</label>
        <textarea id="k-summary" placeholder="Charges retry three times, then stop.">${escape(existing?.summary || '')}</textarea>
      </div>
      <div class="field-row">
        <label class="field" for="k-why">Why</label>
        <textarea id="k-why" placeholder="The provider rate limits after four.">${escape(existing?.properties?.why || '')}</textarea>
      </div>
      <p class="notice" id="record-error" hidden></p>
      <div class="modal-actions">
        <button class="btn" id="save">${icon('check', 16)} Save</button>
        <button class="btn outline" id="cancel">Cancel</button>
      </div>
    </div></div>`

  layer.querySelector('#cancel').onclick = close
  layer.querySelector('#scrim').onclick = (event) => {
    if (event.target.id === 'scrim') close()
  }
  layer.querySelector('#save').onclick = async () => {
    const problem = layer.querySelector('#record-error')
    const id = layer.querySelector('#k-id').value.trim()
    const summary = layer.querySelector('#k-summary').value.trim()
    if (!id || !summary) {
      problem.hidden = false
      problem.textContent = 'A name and what is true are both needed.'
      return
    }
    const why = layer.querySelector('#k-why').value.trim()
    try {
      await api('/api/knowledge', {
        method: 'PUT',
        body: JSON.stringify({
          repository: state.repository,
          id,
          kind: layer.querySelector('#k-kind').value,
          status: existing?.status || 'accepted',
          summary,
          properties: why ? { ...(existing?.properties || {}), why } : (existing?.properties || {}),
        }),
      })
      close()
      await knowledge()
    } catch (error) {
      problem.hidden = false
      problem.textContent = error.message
    }
  }
}

async function forget(id) {
  if (!confirm(`Forget ${id}?`)) return
  try {
    await api(`/api/knowledge?repository=${encodeURIComponent(state.repository)}&id=${encodeURIComponent(id)}`,
      { method: 'DELETE' })
  } catch (error) {
    state.error = error.message
  }
  await knowledge()
}

/* Shell */

async function load() {
  try {
    state.repositories = await api('/api/repositories')
    state.error = ''
  } catch (error) {
    state.repositories = []
    state.error = `${error.message}. Is sourceant-agent running?`
  }
  if (!state.repositories.some((repository) => repository.name === state.repository)) {
    state.repository = state.repositories[0]?.name || ''
  }
}

async function route() {
  state.page = (location.hash.replace(/^#\/?/, '') || '').split('?')[0]
  if (!PAGES.some((page) => page.id === state.page)) state.page = ''
  renderTabs()
  view.classList.toggle('fills', state.page === 'graph')
  if (state.page !== 'graph') {
    drawing?.destroy()
    drawing = null
  }
  if (state.page === 'repositories') return repositories()
  if (state.page === 'graph') return graphPage()
  if (state.page === 'knowledge') return knowledge()
  return overview()
}

function setTheme(theme) {
  document.documentElement.className = theme
  themeButton.innerHTML = icon(theme === 'dark' ? 'sun' : 'moon', 16)
  themeButton.setAttribute('aria-label', theme === 'dark' ? 'Light mode' : 'Dark mode')
  try {
    localStorage.setItem('sourceant-theme', theme)
  } catch {
    // A browser that refuses storage still gets the theme, just not the memory.
  }
  drawing?.repaint()
}

themeButton.onclick = () =>
  setTheme(document.documentElement.className === 'dark' ? 'light' : 'dark')
window.addEventListener('hashchange', route)

let stored = null
try {
  stored = localStorage.getItem('sourceant-theme')
} catch {
  stored = null
}
setTheme(stored === 'light' ? 'light' : 'dark')

load().then(route)
