import { ref } from 'vue'
import { defineStore } from 'pinia'

/**
 * Example: button with integrated loading spinner
 *
 * <button
 *   :disabled="loading.is('<type>', <element_id>, '<action>')"
 * >
 *   <span
 *     v-if="loading.is('<type>', <element_id>, '<action>')"
 *     class="loading loading-spinner loading-xs animate-spin"
 *   ></span>
 *     <!-- Button Content Here -->
 * </button>
 */

/**
 * Global registry for UI loading states (spinners / disabled buttons).
 * Maintains a Set of active async action keys. A key stays active from `start(...segments)` to
 * `stop(...segments)` or for the duration of `withLoading(fn, ...segments)`.
 *
 * Segments -> key rules:
 * - Drop empty / null / undefined / false segments
 * - Join remaining segments with '|'
 * Suggested pattern: domain, id?, action (e.g. 'vm', 42, 'restart')
 *
 * Boolean only (no ref counting). Multiple start() calls on same key need matching manual handling
 * if you later extend behavior.
 *
 * - start(...segments)  Add key
 * - stop(...segments)   Remove key
 * - is(...segments)     Check if active
 * - clear(prefix?)      Clear all or those starting with prefix
 * - makeKey(...segments) Build key string (utility)
 */
export const useLoadingStore = defineStore('loading', () => {
  const active = ref<Set<string>>(new Set())
  type KeySeg = string | number | boolean | null | undefined

  const makeKey = (...segments: KeySeg[]) =>
    segments.filter((s) => s !== '' && s !== null && s !== undefined && s !== false).join('|')

  function start(...segments: KeySeg[]) {
    const k = makeKey(...segments)
    if (k) active.value.add(k)
  }
  function stop(...segments: KeySeg[]) {
    const k = makeKey(...segments)
    if (k) active.value.delete(k)
  }
  function is(...segments: KeySeg[]) {
    const k = makeKey(...segments)
    return k ? active.value.has(k) : false
  }
  function clear(prefix?: string) {
    if (!prefix) return active.value.clear()
    for (const k of Array.from(active.value)) if (k.startsWith(prefix)) active.value.delete(k)
  }

  return { active, start, stop, is, clear, makeKey }
})
