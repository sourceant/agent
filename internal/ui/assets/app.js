/* The local code graph.
 *
 * Colours are concrete hex rather than the CSS custom properties above,
 * because the graph draws to a canvas and cannot resolve them. They are the
 * dashboard's palette, mapped onto what a code graph holds.
 */
const COLOURS = {
  repository: '#E20C18',
  directory: '#9560f0',
  file: '#3b82f6',
  import: '#f59e0b',
  function: '#4ade80',
  method: '#2dd4bf',
  class: '#c084fc',
  struct: '#22d3ee',
  interface: '#22d3ee',
  enum: '#22d3ee',
}
const OTHER = '#a1a1aa'

const LAYOUTS = [
  { id: 'force', label: 'Force', dag: null },
  { id: 'tree', label: 'Tree', dag: 'td' },
  { id: 'radial', label: 'Radial', dag: 'radialout' },
  { id: 'sideways', label: 'Sideways', dag: 'lr' },
]

/* A file's kind is its language and a symbol's kind is what the parser called
 * it, so kind alone cannot tell a Python file from a Python function. The
 * labels the index carries can, which is what this reads. */
function groupOf(node) {
  if (node.synthetic) return node.synthetic
  const labels = node.labels || []
  if (labels.includes('File')) return 'file'
  if (labels.includes('Import')) return 'import'
  return (node.kind || '').toLowerCase()
}

/* Files hold their symbols and their imports, and nothing holds the files, so
 * drawing the index as it is stored scatters a repository into one island per
 * file. The directories are already in every path; this reads them out and
 * hangs the files off them, which is the difference between a repository and
 * confetti. The nodes it adds are marked synthetic: they are how this view
 * arranges what the index found, not something the index found. */
function withFolders(data, repository) {
  const root = { id: 'tree:', name: repository, kind: 'repository', synthetic: 'repository', path: '' }
  const folders = new Map([['', root]])
  const links = [...data.links]

  const folderFor = (path) => {
    if (folders.has(path)) return folders.get(path)
    const cut = path.lastIndexOf('/', path.length - 2)
    const parentPath = cut === -1 ? '' : path.slice(0, cut + 1)
    const parent = folderFor(parentPath)
    const folder = {
      id: `tree:${path}`,
      name: path.slice(parentPath.length).replace(/\/$/, ''),
      kind: 'directory',
      synthetic: 'directory',
      path,
    }
    folders.set(path, folder)
    links.push({ source: parent.id, target: folder.id, type: 'contains' })
    return folder
  }

  for (const node of data.nodes) {
    if (groupOf(node) !== 'file' || !node.path) continue
    const cut = node.path.lastIndexOf('/')
    const folder = folderFor(cut === -1 ? '' : node.path.slice(0, cut + 1))
    links.push({ source: folder.id, target: node.id, type: 'contains' })
  }

  return { nodes: [...folders.values(), ...data.nodes], links }
}

function colourOf(node) {
  return COLOURS[groupOf(node)] || OTHER
}

function shortName(name) {
  return name && name.length > 30 ? `${name.slice(0, 29)}…` : name
}

const element = {
  canvas: document.getElementById('canvas'),
  overlay: document.getElementById('overlay'),
  repository: document.getElementById('repository'),
  layouts: document.getElementById('layouts'),
  search: document.getElementById('search'),
  tests: document.getElementById('tests'),
  imports: document.getElementById('imports'),
  folders: document.getElementById('folders'),
  symbols: document.getElementById('symbols'),
  legend: document.getElementById('legend'),
  tally: document.getElementById('tally'),
  truncated: document.getElementById('truncated'),
  theme: document.getElementById('theme'),
  details: document.getElementById('details'),
  detailsName: document.getElementById('details-name'),
  detailsKind: document.getElementById('details-kind'),
  detailsPath: document.getElementById('details-path'),
  detailsLinks: document.getElementById('details-links'),
  closeDetails: document.getElementById('close-details'),
}

const state = {
  graph: null,
  loaded: { nodes: [], links: [], truncated: false },
  layout: 'force',
  matching: null,
  selected: null,
  labelled: false,
}

function canvasColour(name) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function say(message) {
  element.overlay.innerHTML = message
  element.overlay.hidden = false
}

async function read(path) {
  const response = await fetch(path)
  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new Error(body.error || `the agent answered ${response.status}`)
  }
  return response.json()
}

/* What is drawn, after the toggles and before the layout. Dropping a node has
 * to drop the links that reach it, or the renderer is handed an edge with no
 * end and stops drawing entirely. */
/* A repository's every function is a texture rather than a picture, so what
 * opens is its shape: folders and files. Symbols and imports are there to be
 * asked for. */
function wanted(node) {
  const group = groupOf(node)
  if (group === 'import') return element.imports.checked
  if (group === 'file') return true
  return element.symbols.checked
}

function visible() {
  const nodes = state.loaded.nodes.filter(wanted)
  const kept = new Set(nodes.map((node) => node.id))
  const links = state.loaded.links
    .filter((link) => kept.has(link.source.id || link.source) && kept.has(link.target.id || link.target))
    .map((link) => ({ source: link.source.id || link.source, target: link.target.id || link.target, type: link.type }))

  const data = { nodes: nodes.map((node) => ({ ...node })), links }
  return element.folders.checked ? withFolders(data, element.repository.value) : data
}

function draw() {
  const data = visible()
  const dag = LAYOUTS.find((layout) => layout.id === state.layout).dag

  if (!state.graph) {
    state.graph = new ForceGraph(element.canvas)
    state.graph
      .backgroundColor('rgba(0,0,0,0)')
      .nodeRelSize(4)
      .nodeLabel((node) => `${node.name} · ${node.kind}`)
      .nodeCanvasObject(paintNode)
      .nodePointerAreaPaint((node, colour, ctx) => {
        ctx.fillStyle = colour
        ctx.beginPath()
        ctx.arc(node.x, node.y, 7, 0, 2 * Math.PI)
        ctx.fill()
      })
      .linkColor(() => canvasColour('--canvas-link'))
      .linkWidth(0.7)
      .linkDirectionalArrowLength(3)
      .linkDirectionalArrowRelPos(1)
      .onNodeClick(select)
      .onBackgroundClick(() => select(null))
    state.graph.onEngineStop(() => state.graph.zoomToFit(400, 40))
  }

  // A drawing small enough to read gets its names at any zoom. A large one
  // would be soup, so there the names wait until something is zoomed into.
  state.labelled = data.nodes.length <= 400

  state.graph
    .dagMode(dag)
    .dagLevelDistance(dag ? 90 : 40)
    .onDagError(() => undefined)
    .width(element.canvas.clientWidth)
    .height(element.canvas.clientHeight)
    .graphData(data)

  element.tally.textContent = `${data.nodes.length.toLocaleString()} nodes · ${data.links.length.toLocaleString()} links`
  element.overlay.hidden = data.nodes.length > 0
  if (data.nodes.length === 0) {
    say('Nothing here yet. Index it with <code>sourceant index</code>.')
  }
  renderLegend(data.nodes)
}

function paintNode(node, ctx, scale) {
  const dimmed = state.matching !== null && !state.matching.has(node.id)
  const colour = colourOf(node)
  ctx.globalAlpha = dimmed ? 0.15 : 1

  ctx.beginPath()
  ctx.arc(node.x, node.y, node.id === state.selected ? 6 : 4, 0, 2 * Math.PI)
  ctx.fillStyle = colour
  ctx.fill()
  if (node.id === state.selected) {
    ctx.lineWidth = 1.5 / scale
    ctx.strokeStyle = canvasColour('--canvas-label')
    ctx.stroke()
  }

  if (state.labelled || scale > 1.4 || state.matching !== null) {
    const size = Math.max(11 / scale, 2)
    ctx.font = `${groupOf(node) === 'file' ? '600 ' : ''}${size}px Inter, sans-serif`
    ctx.fillStyle = colour
    ctx.textAlign = 'left'
    ctx.textBaseline = 'middle'
    ctx.fillText(shortName(node.name), node.x + 6, node.y)
  }
  ctx.globalAlpha = 1
}

function renderLegend(nodes) {
  const present = new Set(nodes.map(groupOf))
  const seen = []
  for (const group of present) {
    seen.push({ group, colour: COLOURS[group] || OTHER })
  }
  seen.sort((a, b) => a.group.localeCompare(b.group))
  element.legend.innerHTML = seen
    .map(({ group, colour }) =>
      `<span><i class="swatch" style="background:${colour}"></i>${group || 'other'}</span>`)
    .join('')
}

function select(node) {
  state.selected = node ? node.id : null
  element.details.hidden = !node
  if (node) {
    const degree = state.loaded.links.filter((link) =>
      (link.source.id || link.source) === node.id || (link.target.id || link.target) === node.id).length
    element.detailsName.textContent = node.name
    element.detailsKind.textContent = node.synthetic
      ? `${node.kind} · this view's arrangement`
      : (groupOf(node) === node.kind ? node.kind : `${groupOf(node)} · ${node.kind}`)
    element.detailsPath.textContent = node.path || '—'
    element.detailsLinks.textContent = node.synthetic ? '—' : degree
  }
  if (state.graph) state.graph.nodeCanvasObject(paintNode)
}

function highlight(term) {
  const needle = term.trim().toLowerCase()
  state.matching = needle
    ? new Set(state.loaded.nodes
        .filter((node) => node.name.toLowerCase().includes(needle) || (node.path || '').toLowerCase().includes(needle))
        .map((node) => node.id))
    : null
  if (state.graph) state.graph.nodeCanvasObject(paintNode)
}

async function load() {
  const repository = element.repository.value
  if (!repository) return
  say('Reading the index…')
  try {
    const query = new URLSearchParams({ repository })
    if (element.tests.checked) query.set('include_tests', 'true')
    const graph = await read(`/api/graph?${query}`)
    state.loaded = graph
    state.selected = null
    element.details.hidden = true
    element.truncated.hidden = !graph.truncated
    if (graph.truncated) {
      element.truncated.textContent =
        'This repository is larger than the limit, so this is part of it, not all of it.'
    }
    draw()
  } catch (error) {
    say(`Could not read the graph: ${error.message}`)
  }
}

async function start() {
  buildLayouts()
  applyStoredTheme()

  try {
    const repositories = await read('/api/repositories')
    if (repositories.length === 0) {
      element.repository.innerHTML = '<option value="">Nothing registered</option>'
      say('No repository is registered on this machine. Add one with <code>sourceant repo add &lt;path&gt;</code>.')
      return
    }
    element.repository.innerHTML = repositories
      .map((repository) => `<option value="${repository.name}">${repository.name}</option>`)
      .join('')
    await load()
  } catch (error) {
    say(`Could not reach the agent: ${error.message}`)
  }
}

function buildLayouts() {
  element.layouts.innerHTML = LAYOUTS
    .map((layout) =>
      `<button type="button" data-layout="${layout.id}" aria-pressed="${layout.id === state.layout}">${layout.label}</button>`)
    .join('')
  element.layouts.addEventListener('click', (event) => {
    const button = event.target.closest('button[data-layout]')
    if (!button) return
    state.layout = button.dataset.layout
    for (const other of element.layouts.querySelectorAll('button')) {
      other.setAttribute('aria-pressed', String(other === button))
    }
    draw()
  })
}

function applyStoredTheme() {
  let stored = null
  try {
    stored = localStorage.getItem('sourceant-theme')
  } catch {
    stored = null
  }
  setTheme(stored === 'light' ? 'light' : 'dark')
}

function setTheme(theme) {
  document.documentElement.className = theme
  element.theme.textContent = theme === 'dark' ? 'Light' : 'Dark'
  try {
    localStorage.setItem('sourceant-theme', theme)
  } catch {
    // A browser that refuses storage still gets the theme, just not the memory.
  }
  if (state.graph) state.graph.linkColor(() => canvasColour('--canvas-link'))
}

element.repository.addEventListener('change', load)
element.tests.addEventListener('change', load)
element.imports.addEventListener('change', draw)
element.folders.addEventListener('change', draw)
element.symbols.addEventListener('change', draw)
element.search.addEventListener('input', (event) => highlight(event.target.value))
element.closeDetails.addEventListener('click', () => select(null))
element.theme.addEventListener('click', () =>
  setTheme(document.documentElement.className === 'dark' ? 'light' : 'dark'))
new ResizeObserver(() => {
  if (state.graph) {
    state.graph.width(element.canvas.clientWidth).height(element.canvas.clientHeight)
  }
}).observe(element.canvas)

start()
