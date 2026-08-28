/* What the agent answers with when asked for a graph.
 *
 * One definition, shared by the renderer and everything feeding it. Two of them
 * is how a drawing and its data drift apart.
 */

export interface GraphNode {
  id: string
  name: string
  /**
   * Knowledge: system | part | contract | decision | constraint | convention |
   * pattern | workaround | discussion | review | file | code.
   * Code: class | function | method | module | route | variable | symbol.
   */
  kind: string
  status?: string
  /** Where it lives, when the graph is of code. */
  path?: string | null
  /** What it is written in, when it is a file. */
  language?: string
  /** What the index filed it under, when the local index answered. */
  labels?: string[]
  /** How many lines meet here. The canvas sizes a node by this. */
  degree?: number
  /** Which part of the code it belongs to, or null if that part is not coloured. */
  community?: number | null
}

/** One part of a code graph: a set of symbols more connected to each other
 *  than to the rest, named after where it lives or what it is built around. */
export interface GraphCommunity {
  id: number
  name: string
  size: number
}

/** A typed edge between two graph nodes. */
export interface GraphLink {
  source: string
  target: string
  type?: string
}

/** Nodes and edges served for the graph view. */
export interface KnowledgeGraphData {
  nodes: GraphNode[]
  links: GraphLink[]
  /** The parts worth colouring, biggest first. Code graphs only. */
  communities?: GraphCommunity[]
  /** True when the answer was cut to a size a person can read. */
  truncated?: boolean
  /** The node it was walked out from, or null if it drew everything. */
  focus?: string | null
}

/** One repository indexed on this machine. */
export interface Repository {
  name: string
  path: string
}

/** One thing recorded about a repository. */
export interface Knowledge {
  id: string
  kind: string
  status: string
  summary: string
  properties: Record<string, unknown>
}
