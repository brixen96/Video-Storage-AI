<template>
	<nav class="navbar navbar-expand-lg navbar-dark bg-dark sticky-top shadow-lg">
		<div class="container-fluid">
			<!-- Brand -->
			<router-link to="/" class="navbar-brand d-flex align-items-center">
				<font-awesome-icon :icon="['fas', 'video']" class="me-2 brand-icon" />
				<span class="brand-text">Video Storage AI</span>
			</router-link>

			<!-- Mobile toggle button -->
			<button
				class="navbar-toggler"
				type="button"
				data-bs-toggle="collapse"
				data-bs-target="#navbarNav"
				aria-controls="navbarNav"
				aria-expanded="false"
				aria-label="Toggle navigation"
			>
				<span class="navbar-toggler-icon"></span>
			</button>

			<!-- Navigation links -->
			<div class="collapse navbar-collapse" id="navbarNav">
				<ul class="navbar-nav me-auto mb-2 mb-lg-0">
					<li class="nav-item">
						<router-link to="/" class="nav-link" exact-active-class="active">
							<font-awesome-icon :icon="['fas', 'home']" class="me-1" />
							Home
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/browser" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'folder-open']" class="me-1" />
							Browser
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/performers" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'users']" class="me-1" />
							Performers
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/videos" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
							Videos
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/libraries" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
							Libraries
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/studios" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'building']" class="me-1" />
							Studios
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/tags" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'tags']" class="me-1" />
							Tags
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/activity" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'chart-line']" class="me-1" />
							Activity
						</router-link>
					</li>
				</ul>

				<!-- Right side items -->
				<div class="d-flex align-items-center gap-3">
					<!-- Activity Status Widget -->
					<div class="widget-container d-none d-lg-block">
						<ActivityStatusWidget />
					</div>

					<!-- Settings -->
					<router-link to="/settings" class="btn btn-outline-light btn-sm">
						<font-awesome-icon :icon="['fas', 'cog']" />
					</router-link>
				</div>
			</div>
		</div>
	</nav>
</template>

<script>
import ActivityStatusWidget from './ActivityStatusWidget.vue'

export default {
	name: 'NavBar',
	components: {
		ActivityStatusWidget,
	},
	data() {
		return {
			searchQuery: '',
		}
	},
	methods: {
		handleSearch() {
			if (this.searchQuery.trim()) {
				this.$router.push({ path: '/videos', query: { search: this.searchQuery } })
			}
		},
	},
}
</script>

<style scoped>
.navbar {
	background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%) !important;
	border-bottom: 2px solid #0f3460;
}

.navbar-brand {
	font-weight: 600;
	font-size: 1.4rem;
	transition: all 0.3s ease;
}

.navbar-brand:hover {
	transform: translateY(-2px);
	color: #00d9ff !important;
}

.brand-icon {
	font-size: 1.6rem;
	color: #00d9ff;
	animation: pulse 2s ease-in-out infinite;
}

.brand-text {
	background: linear-gradient(90deg, #00d9ff, #00a8cc);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	background-clip: text;
}

@keyframes pulse {
	0%,
	100% {
		opacity: 1;
	}
	50% {
		opacity: 0.7;
	}
}

.nav-link {
	color: rgba(255, 255, 255, 0.8) !important;
	font-weight: 500;
	transition: all 0.3s ease;
	padding: 0.5rem 1rem !important;
	border-radius: 0.5rem;
	margin: 0 0.2rem;
}

.nav-link:hover {
	color: #00d9ff !important;
	background-color: rgba(0, 217, 255, 0.1);
	transform: translateY(-2px);
}

.nav-link.active {
	color: #fff !important;
	background: linear-gradient(135deg, #0f3460, #00d9ff);
	box-shadow: 0 4px 15px rgba(0, 217, 255, 0.3);
}

.search-input {
	background-color: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.2);
	color: #fff;
	min-width: 200px;
	transition: all 0.3s ease;
}

.search-input:focus {
	background-color: rgba(255, 255, 255, 0.15);
	border-color: #00d9ff;
	box-shadow: 0 0 0 0.2rem rgba(0, 217, 255, 0.25);
	color: #fff;
}

.search-input::placeholder {
	color: rgba(255, 255, 255, 0.5);
}

.btn-outline-secondary {
	border-color: rgba(255, 255, 255, 0.2);
	color: #fff;
}

.btn-outline-secondary:hover {
	background-color: #00d9ff;
	border-color: #00d9ff;
	color: #1a1a2e;
}

.btn-outline-light {
	transition: all 0.3s ease;
}

.btn-outline-light:hover {
	background-color: #00d9ff;
	border-color: #00d9ff;
	color: #1a1a2e;
	transform: rotate(90deg);
}

.widget-container {
	min-width: fit-content;
}

/* Mobile responsiveness */
@media (max-width: 991px) {
	.nav-link {
		margin: 0.2rem 0;
	}

	.search-input {
		min-width: 150px;
	}
}
</style>
