/* The graph, drawn the way the dashboard draws one.
 *
 * Node radius, label placement, link colour and width, the arrows and the
 * midpoint edge labels are all the dashboard's, so a person who has seen the
 * hosted graph recognises this one. Colours are concrete hex rather than the
 * CSS custom properties, because the graph draws to a canvas and cannot
 * resolve them.
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
  { id: 'layered', label: 'Layered', dag: 'lr' },
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

function colourOf(node) {
  return COLOURS[groupOf(node)] || OTHER
}

function shortName(name) {
  return name && name.length > 28 ? `${name.slice(0, 27)}…` : name
}

function canvasColour(name) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
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

class CodeGraph {
  constructor(element, { onSelect } = {}) {
    this.element = element
    this.onSelect = onSelect || (() => {})
    this.graph = null
    this.layout = 'force'
    this.matching = null
    this.selected = null
    this.labelled = false
    this.observer = new ResizeObserver(() => this.resize())
    this.observer.observe(element)
  }

  destroy() {
    this.observer.disconnect()
    this.graph?._destructor?.()
    this.graph = null
    this.element.innerHTML = ''
  }

  resize() {
    if (!this.graph) return
    this.graph.width(this.element.clientWidth).height(this.element.clientHeight)
  }

  setLayout(id) {
    this.layout = id
    this.draw()
  }

  highlight(term) {
    const needle = term.trim().toLowerCase()
    this.matching = needle
      ? new Set(this.data.nodes
          .filter((node) => node.name.toLowerCase().includes(needle) ||
            (node.path || '').toLowerCase().includes(needle))
          .map((node) => node.id))
      : null
    this.repaint()
  }

  select(node) {
    this.selected = node ? node.id : null
    this.repaint()
    this.onSelect(node)
  }

  repaint() {
    if (this.graph) this.graph.nodeCanvasObject(this.paintNode)
  }

  show(data) {
    this.data = data
    this.selected = null
    this.draw()
  }

  draw() {
    const data = this.data
    const dag = LAYOUTS.find((layout) => layout.id === this.layout).dag
    // A drawing small enough to read gets its names at any zoom, as the hosted
    // graph does. A large one would be soup, so there they wait for a zoom.
    this.labelled = data.nodes.length <= 400

    this.paintNode = (node, ctx, scale) => {
      const dimmed = this.matching !== null && !this.matching.has(node.id)
      const colour = colourOf(node)
      ctx.globalAlpha = dimmed ? 0.15 : 1

      ctx.beginPath()
      ctx.arc(node.x, node.y, node.id === this.selected ? 6 : 4, 0, 2 * Math.PI)
      ctx.fillStyle = colour
      ctx.fill()
      if (node.id === this.selected) {
        ctx.lineWidth = 1.5 / scale
        ctx.strokeStyle = canvasColour('--canvas-label')
        ctx.stroke()
      }

      if (this.labelled || scale > 1.4 || this.matching !== null) {
        const size = Math.max(11 / scale, 2)
        const heavy = groupOf(node) === 'repository' || groupOf(node) === 'file'
        ctx.font = `${heavy ? 'bold ' : ''}${size}px Inter, sans-serif`
        ctx.fillStyle = colour
        ctx.textAlign = 'left'
        ctx.textBaseline = 'middle'
        ctx.fillText(shortName(node.name), node.x + 6, node.y)
      }
      ctx.globalAlpha = 1
    }

    if (!this.graph) {
      this.graph = new ForceGraph(this.element)
      this.graph
        .backgroundColor('rgba(0,0,0,0)')
        .nodeRelSize(4)
        .nodeLabel((node) => `${node.name} · ${node.kind}`)
        .nodePointerAreaPaint((node, colour, ctx) => {
          ctx.fillStyle = colour
          ctx.beginPath()
          ctx.arc(node.x, node.y, 7, 0, 2 * Math.PI)
          ctx.fill()
        })
        .linkWidth(0.7)
        .linkDirectionalArrowLength(3)
        .linkDirectionalArrowRelPos(1)
        .linkCanvasObjectMode(() => 'after')
        .linkCanvasObject((link, ctx, scale) => {
          const start = link.source
          const end = link.target
          if (!link.type || typeof start !== 'object' || typeof end !== 'object') return
          if (!this.labelled && scale <= 1.4) return
          const size = Math.max(9 / scale, 1.5)
          ctx.font = `${size}px monospace`
          ctx.fillStyle = canvasColour('--canvas-label')
          ctx.textAlign = 'center'
          ctx.textBaseline = 'middle'
          ctx.fillText(link.type, (start.x + end.x) / 2, (start.y + end.y) / 2)
        })
        .onNodeClick((node) => this.select(node))
        .onBackgroundClick(() => this.select(null))
      this.graph.onEngineStop(() => this.graph.zoomToFit(500, 40))
    }

    this.graph
      .nodeCanvasObject(this.paintNode)
      .linkColor(() => canvasColour('--canvas-link'))
      .dagMode(dag)
      .dagLevelDistance(dag ? 90 : 40)
      .onDagError(() => undefined)
      .width(this.element.clientWidth)
      .height(this.element.clientHeight)
      .graphData(data)
  }
}
