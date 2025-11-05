<template>
	<div class="studios-page">
		<div class="container-fluid py-4">
			<h1 class="mb-4">
				<font-awesome-icon :icon="['fas', 'building']" class="me-2" />
				Studios
			</h1>
			<p class="lead">Production studios and groups</p>
			<p>Organize videos by studio and sub-labels</p>
		</div>
	</div>
</template>

<script>
export default {
	name: 'StudiosPage',
}
</script>

<style scoped>
.studios-page {
	min-height: calc(100vh - 60px);
	background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
	color: #fff;
}
</style>
