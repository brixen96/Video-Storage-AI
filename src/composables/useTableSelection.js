/**
 * useTableSelection Composable
 *
 * Provides reusable table/list selection logic for checkbox-based bulk selection.
 * Eliminates duplicate selection logic across VideosPage, TagsPage, EditListPage, etc.
 *
 * @example
 * // Options API usage
 * import { useTableSelection } from '@/composables/useTableSelection'
 *
 * data() {
 *   return {
 *     ...useTableSelection().data(),
 *   }
 * },
 * computed: {
 *   ...useTableSelection().computed(function() { return this.filteredVideos }),
 * },
 * methods: {
 *   ...useTableSelection().methods(function() { return this.selectedItems }, function(val) { this.selectedItems = val }),
 * }
 *
 * @example
 * // Composition API usage
 * import { useTableSelection } from '@/composables/useTableSelection'
 *
 * const items = ref([])
 * const { selectedItems, isSelected, toggleSelection, toggleSelectAll, clearSelection, selectedCount, allSelected } = useTableSelection(items)
 */

import { ref, computed } from 'vue'

/**
 * Composition API version
 * @param {Ref<Array>} items - Reactive reference to the items array (filtered items)
 * @returns {Object} Selection state and methods
 */
export function useTableSelection(items) {
	const selectedItems = ref([])

	const selectedCount = computed(() => selectedItems.value.length)

	const allSelected = computed(() => {
		if (!items || !items.value) return false
		return items.value.length > 0 && selectedItems.value.length === items.value.length
	})

	const isSelected = (itemId) => {
		return selectedItems.value.includes(itemId)
	}

	const toggleSelection = (itemId) => {
		const index = selectedItems.value.indexOf(itemId)
		if (index > -1) {
			selectedItems.value.splice(index, 1)
		} else {
			selectedItems.value.push(itemId)
		}
	}

	const toggleSelectAll = () => {
		if (allSelected.value) {
			selectedItems.value = []
		} else {
			selectedItems.value = items.value.map((item) => item.id)
		}
	}

	const clearSelection = () => {
		selectedItems.value = []
	}

	const selectItems = (itemIds) => {
		selectedItems.value = [...itemIds]
	}

	return {
		selectedItems,
		selectedCount,
		allSelected,
		isSelected,
		toggleSelection,
		toggleSelectAll,
		clearSelection,
		selectItems,
	}
}

/**
 * Options API version
 * Returns data, computed, and methods objects for spreading into component
 *
 * @returns {Object} Object with data(), computed(), and methods() functions
 */
export function useTableSelectionOptionsAPI() {
	return {
		/**
		 * Returns data properties
		 */
		data() {
			return {
				selectedItems: [],
			}
		},

		/**
		 * Returns computed properties
		 * @param {Function} getFilteredItems - Function that returns the filtered items array (e.g., () => this.filteredVideos)
		 */
		computed(getFilteredItems) {
			return {
				selectedCount() {
					return this.selectedItems.length
				},
				allSelected() {
					const items = getFilteredItems.call(this)
					return items.length > 0 && this.selectedItems.length === items.length
				},
			}
		},

		/**
		 * Returns methods
		 * @param {Function} getFilteredItems - Function that returns the filtered items array
		 */
		methods(getFilteredItems) {
			return {
				isSelected(itemId) {
					return this.selectedItems.includes(itemId)
				},
				toggleSelection(itemId) {
					const index = this.selectedItems.indexOf(itemId)
					if (index > -1) {
						this.selectedItems.splice(index, 1)
					} else {
						this.selectedItems.push(itemId)
					}
				},
				toggleSelectAll() {
					if (this.allSelected) {
						this.selectedItems = []
					} else {
						const items = getFilteredItems.call(this)
						this.selectedItems = items.map((item) => item.id)
					}
				},
				clearSelection() {
					this.selectedItems = []
				},
				selectItems(itemIds) {
					this.selectedItems = [...itemIds]
				},
			}
		},
	}
}

export default useTableSelection
