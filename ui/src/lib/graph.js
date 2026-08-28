/* What a node is, what colour it draws in, and how a repository is arranged.
 *
 * Shared by the renderer and the page around it, so a legend and a drawing
 * cannot disagree about what a colour means. */

export const COLOURS = {
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

export const OTHER = '#a1a1aa'

/* A file's kind is its language and a symbol's kind is what the parser called
 * it, so kind alone cannot tell a Python file from a Python function. The
 * labels the index carries can. */
export function groupOf(node) {
  if (node.synthetic) return node.synthetic
  const labels = node.labels || []
  if (labels.includes('File')) return 'file'
  if (labels.includes('Import')) return 'import'
  return (node.kind || '').toLowerCase()
}

export function colourFor(node) {
  return COLOURS[groupOf(node)] || OTHER
}

export function shortName(name) {
  return name && name.length > 28 ? `${name.slice(0, 27)}…` : name
}

/* Files hold their symbols and their imports, and nothing holds the files, so
 * drawing the index as it is stored scatters a repository into one island per
 * file. The directories are already in every path; this reads them out and
 * hangs the files off them, which is the difference between a repository and
 * confetti. The nodes it adds are marked synthetic: they are how this view
 * arranges what the index found, not something the index found. */
export function withFolders(data, repository) {
  const root = {
    id: 'tree:',
    name: repository,
    kind: 'repository',
    synthetic: 'repository',
    path: '',
  }
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

/* The hosted graph's modes, and what each asks the layout for. 'zout' is a 3D
 * layout, which is why every mode but the first renders in three dimensions. */
export const MODES = [
  { id: '2d', label: '2D' },
  { id: 'tree', label: 'Tree' },
  { id: 'radial', label: 'Radial' },
  { id: 'layered', label: 'Layered' },
  { id: 'web', label: 'Force' },
]
