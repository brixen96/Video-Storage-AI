<template>
	<div class="videos-page">
		<div class="container-fluid py-4">
			<h1 class="mb-4">
				<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
				Videos
			</h1>
			<p class="lead">Your video library will appear here.</p>
			<p>Browse, search, and manage your video collection</p>
		</div>
	</div>
</template>

<script>
export default {
	name: 'VideosPage',
}
</script>

<style scoped>
.videos-page {
	min-height: calc(100vh - 60px);
	background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
	color: #fff;
}
</style>
