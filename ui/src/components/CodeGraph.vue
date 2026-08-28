<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, computed } from 'vue'
import type { GraphNode, KnowledgeGraphData } from '~/types'
import { useContextKinds } from '~/composables/useContextKinds'

interface Props {
  height?: string
  repo?: string
  mode?: '2d' | '3d' | 'web' | 'tree' | 'radial' | 'layered'
  /** The graph to draw. Nothing is drawn until there is one. */
  data?: KnowledgeGraphData | null
  /** Parts the reader has switched off in the legend. */
  hidden?: number[]
}
const emit = defineEmits<{ select: [id: string] }>()
const props = withDefaults(defineProps<Props>(), {
  height: '460px',
  repo: '',
  mode: '2d',
  data: null,
  hidden: () => [],
})

/** A node once the canvas has what it needs: how big, what colour, and whether
 *  it is busy enough to be worth naming. The engine adds a position in place. */
interface Drawn extends GraphNode {
  color: string
  radius: number
  named: boolean
  x: number
  y: number
  z?: number
}
/** A link once the engine has resolved its ends from ids to the nodes. */
interface DrawnLink {
  source: Drawn | string
  target: Drawn | string
  type?: string
}
/** Either end, once resolved. Before the first tick they are still ids. */
const ends = (l: DrawnLink): [Drawn | null, Drawn | null] => [
  typeof l.source === 'object' ? l.source : null,
  typeof l.target === 'object' ? l.target : null,
]
const together = (l: DrawnLink): Drawn | null => {
  const [a, b] = ends(l)
  return a && b && a.community === b.community && a.community !== null ? a : null
}

const el = ref<HTMLElement | null>(null)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let graph: any = null
let engine: '2d' | '3d' | null = null
let ro: ResizeObserver | null = null
// Where every node was last time, so a refetch settles rather than restarting.
const seats = new Map<string, { x: number, y: number, z: number }>()

const { ink } = useContextKinds()

/**
 * Ten colours for the parts of a code graph, and no more: past this the palette
 * repeats and colour stops meaning anything. Tableau 10, which is chosen to stay
 * distinguishable next to itself.
 */
const PARTS = [
  '#4E79A7', '#F28E2B', '#E15759', '#76B7B2', '#59A14F',
  '#EDC948', '#B07AA1', '#FF9DA7', '#9C755F', '#BAB0AC',
]
/** Everything outside the coloured parts, and every line: present, not shouting. */
const QUIET = '#6b7280'

/**
 * The graph draws to canvas and WebGL, where a CSS custom property is nothing,
 * so these have to be concrete. Kinds come from the one table the rest of the
 * app uses; only the two the knowledge graph adds are named here.
 */
function colorFor(node: { kind: string, community?: number | null }): string {
  if (typeof node.community === 'number') {
    return PARTS[node.community % PARTS.length]!
  }
  const kind = (node.kind || '').toLowerCase()
  if (kind === 'code') return '#E20C18'
  if (kind === 'file') return QUIET
  // A code graph node outside a coloured part, rather than a knowledge kind.
  if (['class', 'function', 'method', 'module', 'route', 'variable', 'symbol'].includes(kind)) {
    return QUIET
  }
  return ink(kind)
}

function shortName(name: string): string {
  return name && name.length > 28 ? `${name.slice(0, 27)}…` : name
}

const hasGraph = computed(() => !!props.data && props.data.nodes.length > 0)
defineExpose({ hasGraph })

const hiddenParts = computed(() => new Set(props.hidden))

/**
 * The drawing, with everything the canvas needs worked out once rather than per
 * frame: size, colour, and whether a node is big enough to earn a label.
 */
function effectiveData() {
  const src = props.data ?? { nodes: [], links: [] }
  const shown = src.nodes.filter(
    n => typeof n.community !== 'number' || !hiddenParts.value.has(n.community),
  )
  const here = new Set(shown.map(n => n.id))
  const busiest = Math.max(1, ...shown.map(n => n.degree ?? 0))

  const nodes = shown.map((n) => {
    const seat = seats.get(n.id)
    return {
      ...n,
      ...(seat ?? {}),
      color: colorFor(n),
      // Sized against how much meets here, not against whichever node happens
      // to be busiest. Measured relatively, a graph where the busiest node has
      // three neighbours drew half of itself at full size, which says "these
      // are the big ones" about a graph that has none.
      //
      // Square root, not linear: linear makes one hub enormous and flattens
      // everything else onto the floor, which is the same as no sizing at all.
      radius: Math.min(3 + 2.4 * Math.sqrt(n.degree ?? 0), 11),
      // Named where something actually meets, and where a name is worth the
      // room. Every label is a moving object once the layout settles, so a
      // drawing that names most of itself is text sliding over a picture.
      named: (n.degree ?? 0) >= Math.max(4, busiest * 0.5),
    }
  })

  const links = src.links
    .filter(l => here.has(String(l.source)) && here.has(String(l.target)))
    .map(l => ({ ...l }))

  return { nodes, links }
}

/**
 * Lines fade as there are more of them. Two thousand at full strength is a
 * white haze with nodes somewhere inside it; the nodes stay bright and the mesh
 * behind them recedes.
 */
function linkAlpha(count: number): number {
  return Math.max(0.06, Math.min(0.5, Math.sqrt(400 / Math.max(1, count))))
}

/** A hex colour at a given strength, which is the only form a canvas stroke takes. */
function faded(hex: string, alpha: number): string {
  const n = parseInt(hex.slice(1), 16)
  const a = Math.max(0, Math.min(1, alpha)).toFixed(3)
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${a})`
}

const dagModeFor = (m: string): string | null =>
  m === 'tree' ? 'td' : m === 'radial' ? 'radialout' : m === 'layered' ? 'zout' : null

function remember() {
  if (!graph) return
  for (const node of graph.graphData().nodes) {
    if (Number.isFinite(node.x)) {
      seats.set(node.id, { x: node.x, y: node.y, z: node.z ?? 0 })
    }
  }
}

function teardown() {
  remember()
  if (graph) {
    graph._destructor?.()
    graph = null
  }
  if (el.value) el.value.innerHTML = ''
}

async function mount(mode: string) {
  if (!el.value) return
  const want: '2d' | '3d' = mode === '2d' ? '2d' : '3d'

  if (graph && engine === want) {
    if (want === '3d') graph.dagMode(dagModeFor(mode))
    remember()
    graph.graphData(effectiveData())
    return
  }

  teardown()
  engine = want

  if (want === '2d') {
    const ForceGraph = (await import('force-graph')).default
    const { forceCollide, forceManyBody, forceX, forceY } = await import('d3-force-3d')
    const drawing = effectiveData()
    const alpha = linkAlpha(drawing.links.length)
    let hovered: string | null = null

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    graph = new (ForceGraph as any)(el.value)
    graph
      .backgroundColor('rgba(0,0,0,0)')
      .graphData(drawing)
      .nodeRelSize(4)
      .nodeColor((n: Drawn) => n.color)
      .nodeLabel((n: Drawn) => (n.path ? `${n.name} — ${n.path}` : n.name))
      .nodeCanvasObject((node: Drawn, ctx: CanvasRenderingContext2D, scale: number) => {
        const lit = !hovered || hovered === node.id
        ctx.globalAlpha = lit ? 1 : 0.25
        ctx.beginPath()
        ctx.arc(node.x, node.y, node.radius, 0, 2 * Math.PI)
        ctx.fillStyle = node.color
        ctx.fill()

        if (!node.named && hovered !== node.id) {
          ctx.globalAlpha = 1
          return
        }
        // Held at a constant size on screen, so zooming in reveals more of the
        // graph rather than magnifying the words.
        const fontSize = Math.max(11 / scale, 1.5)
        ctx.font = `${fontSize}px ui-sans-serif, system-ui, sans-serif`
        ctx.fillStyle = node.color
        ctx.textAlign = 'left'
        ctx.textBaseline = 'middle'
        ctx.fillText(shortName(node.name), node.x + node.radius + 2 / scale, node.y)
        ctx.globalAlpha = 1
      })
      .nodePointerAreaPaint((node: Drawn, color: string, ctx: CanvasRenderingContext2D) => {
        ctx.fillStyle = color
        ctx.beginPath()
        ctx.arc(node.x, node.y, node.radius + 2, 0, 2 * Math.PI)
        ctx.fill()
      })
      // A line between two parts is the interesting one, so it keeps its colour
      // while a line inside a part takes the colour of the part it is inside.
      .linkColor((l: DrawnLink) => {
        const inside = together(l)
        return faded(inside ? inside.color : QUIET, alpha * (inside ? 1.6 : 1))
      })
      .linkWidth(0.6)
      .linkDirectionalArrowLength(2.5)
      .linkDirectionalArrowRelPos(1)
      .linkLabel((l: DrawnLink) => l.type ?? '')
      .onNodeHover((node: Drawn | null) => {
        hovered = node ? node.id : null
        if (el.value) el.value.style.cursor = node ? 'pointer' : ''
        // The flat drawing only repaints on a tick, and the layout settles, so
        // after it has come to rest nothing would show what is under the
        // cursor.
        graph?.refresh?.()
      })
      .width(el.value.clientWidth)
      .height(el.value.clientHeight)
      .onNodeClick((node: Drawn) => emit('select', String(node.id)))

    graph.d3AlphaDecay(0.02).d3VelocityDecay(0.35).cooldownTicks(220)

    /**
     * The defaults produce exactly the picture this replaces. Repulsion reaches
     * across the entire canvas with no cutoff, so anything loosely attached
     * accelerates away and never comes back; nothing pulls a node towards the
     * middle, because the built-in centring only slides the whole cloud; and
     * there is no collision at all, so the connected part collapses into a knot
     * too dense to read.
     */
    graph.d3Force('charge', forceManyBody().strength(-140).distanceMax(420))
    graph.d3Force('center', null)
    graph.d3Force('x', forceX(0).strength(0.03))
    graph.d3Force('y', forceY(0).strength(0.03))
    graph.d3Force('collide', forceCollide((n: Drawn) => n.radius + 2).iterations(2))
    // Symbols in one part sit closer together than symbols in different ones,
    // which is what makes a part read as a part.
    graph.d3Force('link')
      ?.distance((l: DrawnLink) => (together(l) ? 26 : 70))
      .strength((l: DrawnLink) => (together(l) ? 0.7 : 0.08))

    let fitted = false
    graph.onEngineTick(() => {
      // Fit once it has taken shape, rather than after the fifteen second
      // timeout the engine would otherwise be left to hit.
      if (!fitted && graph.d3Force('link')) {
        fitted = true
        setTimeout(() => graph?.zoomToFit(600, 60), 700)
      }
    })
    graph.onEngineStop(() => {
      remember()
      graph?.zoomToFit(600, 60)
    })
    return
  }

  const ForceGraph3D = (await import('3d-force-graph')).default
  const THREE = await import('three')
  const drawing = effectiveData()
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  graph = new (ForceGraph3D as any)(el.value)
  graph
    .backgroundColor('rgba(0,0,0,0)')
    .showNavInfo(false)
    .onDagError(() => undefined)
    .dagLevelDistance(46)
    .dagMode(dagModeFor(mode))
    .graphData(drawing)
    .nodeLabel((n: Drawn) => (n.path ? `${n.name} — ${n.path}` : n.name))
    // A sphere, not a floating word. Drawn as text the graph had no shape at
    // all: no hubs, no clusters, just labels hanging in the dark.
    .nodeThreeObject((node: Drawn) => {
      const geometry = new THREE.SphereGeometry(node.radius, 12, 10)
      const material = new THREE.MeshLambertMaterial({
        color: node.color,
        transparent: true,
        opacity: 0.92,
      })
      return new THREE.Mesh(geometry, material)
    })
    .linkColor(() => QUIET)
    .linkOpacity(linkAlpha(drawing.links.length))
    .linkWidth(0.5)
    .linkDirectionalArrowLength(2)
    .linkDirectionalArrowRelPos(1)
    .linkLabel((l: DrawnLink) => l.type ?? '')
    .width(el.value.clientWidth)
    .height(el.value.clientHeight)
    .onNodeClick((node: Drawn) => emit('select', String(node.id)))

  graph.d3Force('charge')?.strength(-140).distanceMax(420)
  graph.onEngineStop(() => {
    remember()
    graph?.zoomToFit(600, 60)
  })
}

onMounted(async () => {
  await mount(props.mode)
  ro = new ResizeObserver(() => {
    if (graph && el.value) graph.width(el.value.clientWidth).height(el.value.clientHeight)
  })
  if (el.value) ro.observe(el.value)
})

watch(() => props.mode, m => mount(m))
watch(() => props.repo, () => {
  // A different repository is a different graph, so nothing is worth keeping.
  seats.clear()
  mount(props.mode)
})
watch(() => props.data, () => mount(props.mode))
watch(() => props.hidden, () => mount(props.mode), { deep: true })

onBeforeUnmount(() => {
  if (ro) { ro.disconnect(); ro = null }
  teardown()
})
</script>

<template>
  <div ref="el" :style="{ height: props.height }" class="w-full" />
</template>
