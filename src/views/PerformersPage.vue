<template>
	<div class="performers-page">
		<div class="container-fluid py-4">
			<h1 class="mb-4">
				<font-awesome-icon :icon="['fas', 'users']" class="me-2" />
				Performers
			</h1>
			<p class="lead">Your performer collection will appear here.</p>
		</div>
	</div>
</template>

<script>
export default {
	name: 'PerformersPage',
}
</script>

<style scoped>
.performers-page {
	min-height: calc(100vh - 60px);
	background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
	color: #fff;
}
</style>
