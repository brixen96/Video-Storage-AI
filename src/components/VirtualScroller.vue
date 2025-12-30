<template>
	<div class="virtual-scroller-wrapper" :style="{ height: height }">
		<div ref="scrollContainer" class="scroll-container" @scroll="handleScroll">
			<div :style="{ height: `${totalHeight}px`, position: 'relative' }">
				<div :style="{ transform: `translateY(${offsetY}px)` }">
					<slot name="item" v-for="item in visibleItems" :key="getItemKey(item)" :item="item" :index="item._index"></slot>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'

const props = defineProps({
	items: {
		type: Array,
		required: true,
	},
	itemHeight: {
		type: Number,
		default: 200,
	},
	height: {
		type: String,
		default: '600px',
	},
	buffer: {
		type: Number,
		default: 5,
	},
	keyField: {
		type: String,
		default: 'id',
	},
})

const scrollContainer = ref(null)
const scrollTop = ref(0)
const containerHeight = ref(600)

// Calculate visible range
const visibleRange = computed(() => {
	const start = Math.floor(scrollTop.value / props.itemHeight)
	const visibleCount = Math.ceil(containerHeight.value / props.itemHeight)
	const end = start + visibleCount

	return {
		start: Math.max(0, start - props.buffer),
		end: Math.min(props.items.length, end + props.buffer),
	}
})

// Get visible items with index
const visibleItems = computed(() => {
	const { start, end } = visibleRange.value
	return props.items.slice(start, end).map((item, index) => ({
		...item,
		_index: start + index,
	}))
})

// Calculate offset for transform
const offsetY = computed(() => {
	return visibleRange.value.start * props.itemHeight
})

// Total height of all items
const totalHeight = computed(() => {
	return props.items.length * props.itemHeight
})

// Get item key
const getItemKey = (item) => {
	return item[props.keyField] || item._index
}

// Handle scroll event
const handleScroll = () => {
	if (scrollContainer.value) {
		scrollTop.value = scrollContainer.value.scrollTop
	}
}

// Update container height
const updateContainerHeight = () => {
	if (scrollContainer.value) {
		containerHeight.value = scrollContainer.value.clientHeight
	}
}

// Scroll to specific index
const scrollToIndex = (index) => {
	if (scrollContainer.value) {
		scrollContainer.value.scrollTop = index * props.itemHeight
	}
}

// Expose methods
defineExpose({
	scrollToIndex,
})

// Setup resize observer
let resizeObserver = null

onMounted(async () => {
	await nextTick()
	updateContainerHeight()

	// Watch for container resize
	if (window.ResizeObserver && scrollContainer.value) {
		resizeObserver = new ResizeObserver(() => {
			updateContainerHeight()
		})
		resizeObserver.observe(scrollContainer.value)
	}
})

onUnmounted(() => {
	if (resizeObserver) {
		resizeObserver.disconnect()
	}
})

// Watch for items change to reset scroll if needed
watch(
	() => props.items.length,
	() => {
		// Reset scroll if items drastically change
		if (scrollTop.value > totalHeight.value) {
			scrollTop.value = 0
			if (scrollContainer.value) {
				scrollContainer.value.scrollTop = 0
			}
		}
	}
)
</script>

<style scoped>
.virtual-scroller-wrapper {
	width: 100%;
	overflow: hidden;
}

.scroll-container {
	width: 100%;
	height: 100%;
	overflow-y: auto;
	overflow-x: hidden;
}

/* Smooth scrolling */
.scroll-container {
	scroll-behavior: smooth;
}

/* Custom scrollbar */
.scroll-container::-webkit-scrollbar {
	width: 8px;
}

.scroll-container::-webkit-scrollbar-track {
	background: rgba(0, 0, 0, 0.1);
	border-radius: 4px;
}

.scroll-container::-webkit-scrollbar-thumb {
	background: rgba(0, 217, 255, 0.3);
	border-radius: 4px;
}

.scroll-container::-webkit-scrollbar-thumb:hover {
	background: rgba(0, 217, 255, 0.5);
}
</style>
