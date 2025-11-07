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
@import '@/styles/pages/settings_page.css';
</style>
