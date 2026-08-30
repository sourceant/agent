import { useRouter } from 'vue-router'

/** Going up to the listing a detail page belongs to.
 *
 * Up, not back. This control carries a label saying where it goes, so it has
 * to go there: sending it wherever somebody arrived from means "Back to
 * repositories" lands on skills, which is the label lying about itself.
 *
 * Retracing is the browser's own back button, which is always there and needs
 * no help from us.
 */
export function useUp() {
  const router = useRouter()
  return (to) => router.push(to)
}
