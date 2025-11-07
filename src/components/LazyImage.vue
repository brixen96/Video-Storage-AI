<template>
	<div class="lazy-image-wrapper" :style="{ paddingBottom: aspectRatio }">
		<img v-if="isVisible" :src="currentSrc" :alt="alt" @load="onLoad" @error="onError" :class="{ loaded: isLoaded, error: hasError }" class="lazy-image" />
		<div v-if="!isLoaded && !hasError" class="lazy-placeholder">
			<font-awesome-icon :icon="['fas', 'spinner']" spin />
		</div>
		<div v-if="hasError" class="lazy-error">
			<font-awesome-icon :icon="['fas', 'image']" />
		</div>
	</div>
</template>

<script>
export default {
	name: 'LazyImage',
	props: {
		src: String,
		alt: String,
		placeholder: String,
		aspectRatio: {
			type: String,
			default: '56.25%', // 16:9
		},
	},
	data() {
		return {
			isVisible: false,
			isLoaded: false,
			hasError: false,
			currentSrc: this.placeholder || '',
			observer: null,
		}
	},
	mounted() {
		this.setupIntersectionObserver()
	},
	beforeUnmount() {
		if (this.observer) {
			this.observer.disconnect()
		}
	},
	methods: {
		setupIntersectionObserver() {
			this.observer = new IntersectionObserver(
				(entries) => {
					entries.forEach((entry) => {
						if (entry.isIntersecting) {
							this.loadImage()
							this.observer.disconnect()
						}
					})
				},
				{ rootMargin: '50px' }
			)
			this.observer.observe(this.$el)
		},
		loadImage() {
			this.isVisible = true
			this.currentSrc = this.src
		},
		onLoad() {
			this.isLoaded = true
		},
		onError() {
			this.hasError = true
		},
	},
}
</script>

<style scoped>
@import '@/styles/components/lazy_image.css';
</style>
