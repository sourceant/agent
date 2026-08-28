<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { colourFor, groupOf, shortName } from '~/lib/graph'

/* The dashboard's KnowledgeGraph, drawing code instead of contexts.
 *
 * 2D uses force-graph; every other mode uses 3d-force-graph with SpriteText
 * labels, which is what the hosted graph does. Colours are concrete hex, not
 * the CSS custom properties: the graph renders to canvas and WebGL, which
 * cannot resolve them. */

const props = defineProps({
  data: { type: Object, default: null },
  mode: { type: String, default: '2d' },
  repository: { type: String, default: '' },
})
const emit = defineEmits(['select'])

const el = ref(null)
let graph = null
let engine = null
let ro = null

const dagModeFor = (mode) =>
  mode === 'tree' ? 'td' : mode === 'radial' ? 'radialout' : mode === 'layered' ? 'zout' : null

// What holds a repository together reads larger than what it holds.
function textHeightFor(node) {
  const group = groupOf(node)
  if (group === 'repository') return 9
  if (group === 'directory' || group === 'file') return 7
  return 5.5
}

function effective() {
  const source = props.data || { nodes: [], links: [] }
  return {
    nodes: source.nodes.map((node) => ({ ...node, color: colourFor(node) })),
    links: source.links.map((link) => ({ ...link })),
  }
}

function teardown() {
  if (graph) {
    graph._destructor?.()
    graph = null
  }
  if (el.value) el.value.innerHTML = ''
}

async function mount(mode) {
  if (!el.value) return
  const want = mode === '2d' ? '2d' : '3d'

  if (graph && engine === want) {
    if (want === '3d') graph.dagMode(dagModeFor(mode))
    graph.graphData(effective())
    return
  }

  teardown()
  engine = want

  if (want === '2d') {
    const ForceGraph = (await import('force-graph')).default
    graph = new ForceGraph(el.value)
    graph
      .backgroundColor('rgba(0,0,0,0)')
      .graphData(effective())
      .nodeRelSize(4)
      .nodeColor((node) => node.color)
      .nodeLabel('name')
      .nodeCanvasObject((node, ctx, scale) => {
        ctx.beginPath()
        ctx.arc(node.x, node.y, 4, 0, 2 * Math.PI)
        ctx.fillStyle = node.color
        ctx.fill()
        const fontSize = Math.max(11 / scale, 2)
        const heavy = groupOf(node) === 'repository' || groupOf(node) === 'file'
        ctx.font = `${heavy ? 'bold ' : ''}${fontSize}px Inter, sans-serif`
        ctx.fillStyle = node.color
        ctx.textAlign = 'left'
        ctx.textBaseline = 'middle'
        ctx.fillText(shortName(node.name), node.x + 6, node.y)
      })
      .nodePointerAreaPaint((node, colour, ctx) => {
        ctx.fillStyle = colour
        ctx.beginPath()
        ctx.arc(node.x, node.y, 6, 0, 2 * Math.PI)
        ctx.fill()
      })
      .linkColor(() => '#52525b')
      .linkWidth(0.7)
      .linkDirectionalArrowLength(3)
      .linkDirectionalArrowRelPos(1)
      .linkCanvasObjectMode(() => 'after')
      .linkCanvasObject((link, ctx, scale) => {
        const start = link.source
        const end = link.target
        if (!link.type || typeof start !== 'object' || typeof end !== 'object') return
        const fontSize = Math.max(9 / scale, 1.5)
        ctx.font = `${fontSize}px monospace`
        ctx.fillStyle = '#9ca3af'
        ctx.textAlign = 'center'
        ctx.textBaseline = 'middle'
        ctx.fillText(link.type, (start.x + end.x) / 2, (start.y + end.y) / 2)
      })
      .onNodeClick((node) => emit('select', node))
      .onBackgroundClick(() => emit('select', null))
      .width(el.value.clientWidth)
      .height(el.value.clientHeight)
    graph.onEngineStop(() => graph.zoomToFit(500, 40))
    return
  }

  const ForceGraph3D = (await import('3d-force-graph')).default
  const SpriteText = (await import('three-spritetext')).default
  graph = new ForceGraph3D(el.value)
  graph
    .backgroundColor('rgba(0,0,0,0)')
    .showNavInfo(false)
    .enableNodeDrag(false)
    .onDagError(() => undefined)
    .dagLevelDistance(46)
    .dagMode(dagModeFor(mode))
    .graphData(effective())
    .nodeLabel('name')
    .nodeThreeObject((node) => {
      const sprite = new SpriteText(shortName(node.name))
      sprite.color = node.color
      sprite.textHeight = textHeightFor(node)
      sprite.fontWeight = groupOf(node) === 'repository' ? '700' : '500'
      return sprite
    })
    .linkColor(() => '#52525b')
    .linkOpacity(0.35)
    .linkWidth(0.6)
    .linkDirectionalArrowLength(2.5)
    .linkDirectionalArrowRelPos(1)
    .linkThreeObjectExtend(true)
    .linkThreeObject((link) => {
      if (!link.type) return null
      const sprite = new SpriteText(link.type)
      sprite.color = '#9ca3af'
      sprite.textHeight = 3
      return sprite
    })
    .linkPositionUpdate((sprite, { start, end }) => {
      if (!sprite) return
      sprite.position.set((start.x + end.x) / 2, (start.y + end.y) / 2, (start.z + end.z) / 2)
    })
    .onNodeClick((node) => emit('select', node))
    .onBackgroundClick(() => emit('select', null))
    .width(el.value.clientWidth)
    .height(el.value.clientHeight)
  graph.onEngineStop(() => graph.zoomToFit(500, 40))
}

onMounted(async () => {
  await mount(props.mode)
  ro = new ResizeObserver(() => {
    if (graph && el.value) graph.width(el.value.clientWidth).height(el.value.clientHeight)
  })
  if (el.value) ro.observe(el.value)
})

watch(() => props.mode, (mode) => mount(mode))
watch(() => props.repository, () => mount(props.mode))
watch(() => props.data, () => mount(props.mode), { deep: true })

onBeforeUnmount(() => {
  if (ro) {
    ro.disconnect()
    ro = null
  }
  teardown()
})
</script>

<template>
  <div ref="el" class="h-full w-full" />
</template>
