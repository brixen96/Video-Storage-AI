<template>
  <div class="lazy-image-wrapper" :style="{ paddingBottom: aspectRatio }">
    <img
      v-if="isVisible"
      :src="currentSrc"
      :alt="alt"
      @load="onLoad"
      @error="onError"
      :class="{ loaded: isLoaded, error: hasError }"
      class="lazy-image"
    />
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
.lazy-image-wrapper {
  position: relative;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.05);
}

.lazy-image {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.lazy-image.loaded {
  opacity: 1;
}

.lazy-placeholder,
.lazy-error {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: rgba(255, 255, 255, 0.3);
  font-size: 2rem;
}
</style>