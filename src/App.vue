<template>
	<div id="app">
		<NavBar />
		<router-view v-slot="{ Component }">
			<transition name="fade" mode="out-in">
				<component :is="Component" />
			</transition>
		</router-view>
	</div>
</template>

<script>
import NavBar from '@/components/NavBar.vue'

export default {
	name: 'App',
	components: {
		NavBar,
	},
}
</script>

<style>
@import 'bootstrap/dist/css/bootstrap.min.css';

* {
	margin: 0;
	padding: 0;
	box-sizing: border-box;
}

body {
	font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
	background: #0f0c29;
	overflow-x: hidden;
}

#app {
	min-height: 100vh;
}

/* Transition animations */
.fade-enter-active,
.fade-leave-active {
	transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
	opacity: 0;
}

/* Scrollbar styling */
::-webkit-scrollbar {
	width: 10px;
	height: 10px;
}

::-webkit-scrollbar-track {
	background: #1a1a2e;
}

::-webkit-scrollbar-thumb {
	background: #00d9ff;
	border-radius: 5px;
}

::-webkit-scrollbar-thumb:hover {
	background: #00a8cc;
}

/* Custom card styles */
.card {
	border: none;
	box-shadow: 0 4px 15px rgba(0, 0, 0, 0.2);
}

/* Button custom styles */
.btn-primary {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	border: none;
}

.btn-primary:hover {
	background: linear-gradient(135deg, #764ba2 0%, #667eea 100%);
	transform: translateY(-2px);
	box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
}
</style>
