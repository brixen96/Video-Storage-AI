<template>
	<div class="tags-page">
		<div class="container-fluid py-4">
			<h1 class="mb-4">
				<font-awesome-icon :icon="['fas', 'tags']" class="me-2" />
				Tags
			</h1>
			<p class="lead">Tag management dashboard</p>
			<p>Create, edit, and organize video tags with custom colors and icons</p>
		</div>
	</div>
</template>

<script>
export default {
	name: 'TagsPage',
}
</script>

<style scoped>
.tags-page {
	min-height: calc(100vh - 60px);
	background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
	color: #fff;
}
</style>
