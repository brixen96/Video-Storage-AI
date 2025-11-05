<template>
	<div class="performer-details-page">
		<div class="container-fluid py-4">
			<h1 class="mb-4">Performer Details</h1>
			<p class="lead">Performer ID: {{ $route.params.id }}</p>
		</div>
	</div>
</template>

<script>
export default {
	name: 'PerformerDetailsPage',
}
</script>

<style scoped>
.performer-details-page {
	min-height: calc(100vh - 60px);
	background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
	color: #fff;
}
</style>
