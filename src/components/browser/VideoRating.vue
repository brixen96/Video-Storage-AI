<template>
	<div class="video-rating" @click.stop @mouseenter.stop @mouseleave.stop="hoverRating = 0">
		<span
			v-for="star in 5"
			:key="star"
			class="rating-star"
			:class="{ filled: star <= (hoverRating || rating), hovered: hoverRating > 0 && star <= hoverRating }"
			@mouseenter="hoverRating = star"
			@click.stop="$emit('rate', star === rating ? 0 : star)"
		>★</span>
	</div>
</template>

<script setup>
import { ref } from 'vue'

defineProps({
	rating: {
		type: Number,
		default: 0,
	},
})

defineEmits(['rate'])

const hoverRating = ref(0)
</script>

<style scoped>
.video-rating {
	display: inline-flex;
	gap: 1px;
	line-height: 1;
}

.rating-star {
	font-size: 0.85rem;
	color: rgba(255, 255, 255, 0.18);
	cursor: pointer;
	transition: color 0.1s, transform 0.1s;
	user-select: none;
}

.rating-star.filled {
	color: #f5c518;
}

.rating-star.hovered {
	color: #f5c518cc;
	transform: scale(1.15);
}
</style>
