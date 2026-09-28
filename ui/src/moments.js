/* A moment said the way somebody reads it: how long ago, or the date once that
 * stops being the useful answer. */

export function when(stamp) {
  if (!stamp) return ''
  const at = new Date(stamp)
  const ago = Math.round((Date.now() - at.getTime()) / 60000)
  if (ago < 1) return 'just now'
  if (ago < 60) return `${ago} minute${ago === 1 ? '' : 's'} ago`
  const hours = Math.round(ago / 60)
  if (hours < 24) return `${hours} hour${hours === 1 ? '' : 's'} ago`
  return at.toLocaleDateString()
}
