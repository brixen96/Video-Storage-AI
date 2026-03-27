import { ref, watch } from 'vue'
import { debounce } from '@/utils/debounce'

/**
 * Returns a debounced copy of the given source ref.
 * Updates only after `delay` ms of inactivity.
 *
 * @param {import('vue').Ref} source - the reactive ref to debounce
 * @param {number} delay - milliseconds (default 300)
 * @returns {import('vue').Ref} debounced ref
 */
export function useDebouncedRef(source, delay = 300) {
	const debounced = ref(source.value)
	const update = debounce((val) => {
		debounced.value = val
	}, delay)
	watch(source, update)
	return debounced
}
