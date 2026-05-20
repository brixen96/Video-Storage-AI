<template>
	<div class="row g-3 mb-4">
		<div class="col-6 col-lg-2">
			<div class="stat-card">
				<div class="stat-icon download">
					<font-awesome-icon :icon="['fas', 'arrow-down']" />
				</div>
				<div class="stat-content">
					<div class="stat-value">{{ formatSpeed(currentSpeed) }}</div>
					<div class="stat-label">Download Speed</div>
				</div>
			</div>
		</div>
		<div class="col-6 col-lg-2">
			<div class="stat-card">
				<div class="stat-icon queue">
					<font-awesome-icon :icon="['fas', 'list']" />
				</div>
				<div class="stat-content">
					<div class="stat-value">{{ linkGrabberCount }}</div>
					<div class="stat-label">LinkGrabber</div>
				</div>
			</div>
		</div>
		<div class="col-6 col-lg-2">
			<div class="stat-card">
				<div class="stat-icon active">
					<font-awesome-icon :icon="['fas', 'download']" />
				</div>
				<div class="stat-content">
					<div class="stat-value">{{ activeDownloads }}</div>
					<div class="stat-label">Active Downloads</div>
				</div>
			</div>
		</div>
		<div class="col-6 col-lg-2">
			<div class="stat-card">
				<div class="stat-icon limit">
					<font-awesome-icon :icon="['fas', 'tachometer-alt']" />
				</div>
				<div class="stat-content">
					<div class="stat-value">{{ speedLimit > 0 ? formatSpeed(speedLimit) : '∞' }}</div>
					<div class="stat-label">Speed Limit</div>
				</div>
			</div>
		</div>
		<div class="col-6 col-lg-2">
			<div class="stat-card">
				<div class="stat-icon" style="background: linear-gradient(135deg, #17a2b8, #117a8b)">
					<font-awesome-icon :icon="['fas', 'database']" />
				</div>
				<div class="stat-content">
					<div class="stat-value">{{ formatSize(totalDownloadSize) }}</div>
					<div class="stat-label">Total Size</div>
				</div>
			</div>
		</div>
		<div class="col-6 col-lg-2">
			<div class="stat-card">
				<div class="stat-icon" style="background: linear-gradient(135deg, #28a745, #1e7e34)">
					<font-awesome-icon :icon="['fas', 'check-circle']" />
				</div>
				<div class="stat-content">
					<div class="stat-value">{{ completedDownloads }}</div>
					<div class="stat-label">Completed</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
defineProps({
	currentSpeed: { type: Number, default: 0 },
	linkGrabberCount: { type: Number, default: 0 },
	activeDownloads: { type: Number, default: 0 },
	speedLimit: { type: Number, default: 0 },
	totalDownloadSize: { type: Number, default: 0 },
	completedDownloads: { type: Number, default: 0 },
})

const formatSpeed = (bytesPerSecond) => {
	if (!bytesPerSecond || bytesPerSecond === 0) return '0 B/s'
	const units = ['B/s', 'KB/s', 'MB/s', 'GB/s']
	let i = 0
	let speed = bytesPerSecond
	while (speed >= 1024 && i < units.length - 1) {
		speed /= 1024
		i++
	}
	return `${speed.toFixed(1)} ${units[i]}`
}

const formatSize = (bytes) => {
	if (!bytes || bytes === 0) return '0 B'
	const units = ['B', 'KB', 'MB', 'GB', 'TB']
	let i = 0
	let size = bytes
	while (size >= 1024 && i < units.length - 1) {
		size /= 1024
		i++
	}
	return `${size.toFixed(1)} ${units[i]}`
}
</script>

<style scoped>
.stat-card {
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: 1rem;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 12px;
	border: 1px solid rgba(255, 255, 255, 0.08);
	transition: all 0.3s ease;
}

.stat-card:hover {
	background: rgba(255, 255, 255, 0.08);
	transform: translateY(-2px);
}

.stat-icon {
	width: 48px;
	height: 48px;
	display: flex;
	align-items: center;
	justify-content: center;
	border-radius: 12px;
	font-size: 1.25rem;
}

.stat-icon.download {
	background: linear-gradient(135deg, #4caf50, #2e7d32);
	color: white;
}

.stat-icon.queue {
	background: linear-gradient(135deg, #2196f3, #1565c0);
	color: white;
}

.stat-icon.active {
	background: linear-gradient(135deg, #ff9800, #f57c00);
	color: white;
}

.stat-icon.limit {
	background: linear-gradient(135deg, #9c27b0, #6a1b9a);
	color: white;
}

.stat-content {
	flex: 1;
}

.stat-value {
	font-size: 1.25rem;
	font-weight: 600;
	color: #fff;
}

.stat-label {
	font-size: 0.8rem;
	color: rgba(255, 255, 255, 0.6);
}

@media (max-width: 768px) {
	.stat-card {
		padding: 0.75rem;
	}

	.stat-icon {
		width: 40px;
		height: 40px;
		font-size: 1rem;
	}

	.stat-value {
		font-size: 1rem;
	}
}
</style>
