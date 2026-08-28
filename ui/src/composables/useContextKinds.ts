/**
 * What each kind of knowledge looks like, in one place.
 *
 * Six screens carried their own copy of this map, identical to the character,
 * so a colour could only be changed by finding all six. Lookups are
 * case-insensitive because the API answers lowercase and some screens title-case
 * it for display.
 *
 * Tailwind only ever sees literal class names, so these are spelled out in full
 * rather than assembled.
 */
const TONE: Record<string, string> = {
  decision: 'bg-blue-600/10 text-blue-600 border-blue-600/20',
  constraint: 'bg-amber-600/10 text-amber-600 border-amber-600/20',
  workaround: 'bg-red-600/10 text-red-600 border-red-600/20',
  convention: 'bg-violet-600/10 text-violet-600 border-violet-600/20',
  pattern: 'bg-emerald-600/10 text-emerald-600 border-emerald-600/20',
  discussion: 'bg-cyan-600/10 text-cyan-600 border-cyan-600/20',
  review: 'bg-pink-600/10 text-pink-600 border-pink-600/20',
  plan: 'bg-lime-600/10 text-lime-600 border-lime-600/20',
}

const FILL: Record<string, string> = {
  decision: 'bg-blue-600',
  constraint: 'bg-amber-600',
  workaround: 'bg-red-600',
  convention: 'bg-violet-600',
  pattern: 'bg-emerald-600',
  discussion: 'bg-cyan-600',
  review: 'bg-pink-600',
  plan: 'bg-lime-600',
}

/**
 * Concrete hex, for the graph: it draws to canvas, where a CSS variable is
 * nothing.
 *
 * Measured rather than chosen. The set these replace had two pairs a reader
 * could not separate: part against system was indistinguishable to a red-green
 * colourblind reader, and review against discussion was hard to tell apart with
 * full colour vision.
 *
 * Eight hues cannot all be separated from one another, which is a fact about
 * eight rather than about these eight: red-green colourblindness collapses pink
 * onto green whatever shades are picked. These pass every check except one pair
 * of neighbours, and a drawing that needs more than colour to be read needs a
 * second channel rather than better colours.
 */
const INK: Record<string, string> = {
  system: '#7c3aed', part: '#d97706', contract: '#0891b2',
  decision: '#2563eb', constraint: '#d97706', workaround: '#dc2626',
  convention: '#7c3aed', pattern: '#059669', discussion: '#0891b2',
  review: '#db2777', plan: '#65a30d',
}

export function useContextKinds() {
  const NEUTRAL = 'bg-muted text-muted-foreground border-border'

  function tone(kind?: string | null) {
    return TONE[(kind ?? '').toLowerCase()] ?? NEUTRAL
  }

  function fill(kind?: string | null) {
    return FILL[(kind ?? '').toLowerCase()] ?? 'bg-muted-foreground'
  }

  function ink(kind?: string | null) {
    return INK[(kind ?? '').toLowerCase()] ?? '#a1a1aa'
  }

  const legend = Object.keys(INK).map(kind => ({
    kind,
    label: kind.charAt(0).toUpperCase() + kind.slice(1),
    color: INK[kind]!,
  }))

  return { tone, fill, ink, legend, kinds: Object.keys(TONE) }
}
