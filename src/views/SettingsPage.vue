<template>
	<div class="settings-page">
		<div class="container-fluid py-4">
			<h1 class="mb-4">
				<font-awesome-icon :icon="['fas', 'cog']" class="me-2" />
				Settings
			</h1>
			<p class="lead">Application settings and configuration</p>
			<p>Customize your Video Storage AI experience</p>
		</div>
	</div>
</template>

<script>
export default {
	name: 'SettingsPage',
}
</script>

<style scoped>
.settings-page {
	min-height: calc(100vh - 60px);
	background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
	color: #fff;
}
</style>
