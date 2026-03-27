<template>
	<router-link v-if="to" :to="to" class="stat-card" :class="color">
		<div class="stat-icon">
			<font-awesome-icon :icon="['fas', icon]" />
		</div>
		<div class="stat-content">
			<div class="stat-value">{{ formattedValue }}</div>
			<div class="stat-label">{{ label }}</div>
		</div>
	</router-link>
	<div v-else class="stat-card" :class="color">
		<div class="stat-icon">
			<font-awesome-icon :icon="['fas', icon]" />
		</div>
		<div class="stat-content">
			<div class="stat-value">{{ formattedValue }}</div>
			<div class="stat-label">{{ label }}</div>
		</div>
	</div>
</template>

<script>
export default {
	name: 'DashboardStatCard',
	props: {
		value: {
			type: [Number, String],
			default: 0,
		},
		label: {
			type: String,
			required: true,
		},
		icon: {
			type: String,
			default: 'chart-bar',
		},
		color: {
			type: String,
			default: 'blue',
		},
		to: {
			type: String,
			default: null,
		},
	},
	computed: {
		formattedValue() {
			if (typeof this.value === 'number') {
				return this.value.toLocaleString()
			}
			return this.value
		},
	},
}
</script>

<style scoped>
.stat-card {
	background: rgba(30, 30, 40, 0.6);
	border-radius: 10px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	padding: 0.75rem 1rem;
	display: flex;
	align-items: center;
	gap: 0.75rem;
	transition: all 0.3s;
	text-decoration: none;
	color: inherit;
}

.stat-card:hover {
	transform: translateY(-1px);
	box-shadow: 0 4px 15px rgba(0, 0, 0, 0.3);
}

.stat-icon {
	width: 36px;
	height: 36px;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1rem;
}

.stat-card.blue .stat-icon {
	background: linear-gradient(135deg, rgba(59, 130, 246, 0.3), rgba(37, 99, 235, 0.3));
	color: #60a5fa;
}

.stat-card.purple .stat-icon {
	background: linear-gradient(135deg, rgba(168, 85, 247, 0.3), rgba(139, 92, 246, 0.3));
	color: #c084fc;
}

.stat-card.pink .stat-icon {
	background: linear-gradient(135deg, rgba(236, 72, 153, 0.3), rgba(219, 39, 119, 0.3));
	color: #f472b6;
}

.stat-card.green .stat-icon {
	background: linear-gradient(135deg, rgba(34, 197, 94, 0.3), rgba(22, 163, 74, 0.3));
	color: #4ade80;
}

.stat-card.yellow .stat-icon {
	background: linear-gradient(135deg, rgba(234, 179, 8, 0.3), rgba(202, 138, 4, 0.3));
	color: #facc15;
}

.stat-card.orange .stat-icon {
	background: linear-gradient(135deg, rgba(249, 115, 22, 0.3), rgba(234, 88, 12, 0.3));
	color: #fb923c;
}

.stat-card.cyan .stat-icon {
	background: linear-gradient(135deg, rgba(6, 182, 212, 0.3), rgba(8, 145, 178, 0.3));
	color: #22d3ee;
}

.stat-content {
	flex: 1;
}

.stat-value {
	font-size: 1.25rem;
	font-weight: 700;
	color: rgba(255, 255, 255, 0.95);
	line-height: 1.2;
}

.stat-label {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.5);
	margin-top: 0.125rem;
}
</style>
