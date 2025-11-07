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
@import '@/styles/pages/performers_details_page.css';
</style>
