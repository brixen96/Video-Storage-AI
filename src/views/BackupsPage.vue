<template>
	<div class="backups-page">
		<div class="page-header">
			<div class="header-content">
				<h1><font-awesome-icon :icon="['fas', 'database']" /> Database Backups</h1>
				<p>Manage database backups and restore points</p>
			</div>
			<div class="header-actions">
				<button class="btn btn-primary" @click="createBackup" :disabled="creatingBackup">
					<font-awesome-icon :icon="['fas', creatingBackup ? 'spinner' : 'plus']" :spin="creatingBackup" />
					Create Backup
				</button>
			</div>
		</div>

		<!-- Stats Cards -->
		<div class="stats-grid" v-if="stats">
			<div class="stat-card">
				<div class="stat-icon">
					<font-awesome-icon :icon="['fas', 'database']" />
				</div>
				<div class="stat-content">
					<div class="stat-label">Total Backups</div>
					<div class="stat-value">{{ stats.total_backups }}</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon">
					<font-awesome-icon :icon="['fas', 'hdd']" />
				</div>
				<div class="stat-content">
					<div class="stat-label">Total Size</div>
					<div class="stat-value">{{ (stats.total_size_mb || 0).toFixed(2) }} MB</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon">
					<font-awesome-icon :icon="['fas', 'robot']" />
				</div>
				<div class="stat-content">
					<div class="stat-label">Automatic</div>
					<div class="stat-value">{{ stats.automatic_count }}</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon">
					<font-awesome-icon :icon="['fas', 'user']" />
				</div>
				<div class="stat-content">
					<div class="stat-label">Manual</div>
					<div class="stat-value">{{ stats.manual_count }}</div>
				</div>
			</div>
		</div>

		<!-- Backups List -->
		<div class="backups-section">
			<div class="section-header">
				<h2>Available Backups</h2>
				<button class="btn btn-secondary btn-sm" @click="loadBackups">
					<font-awesome-icon :icon="['fas', 'sync']" :spin="loading" />
					Refresh
				</button>
			</div>

			<div v-if="loading" class="loading-state">
				<font-awesome-icon :icon="['fas', 'spinner']" spin size="2x" />
				<p>Loading backups...</p>
			</div>

			<div v-else-if="backups.length === 0" class="empty-state">
				<font-awesome-icon :icon="['fas', 'database']" size="3x" />
				<p>No backups found</p>
				<button class="btn btn-primary" @click="createBackup">
					Create First Backup
				</button>
			</div>

			<div v-else class="backups-list">
				<div v-for="backup in backups" :key="backup.filename" class="backup-card">
					<div class="backup-icon" :class="`icon-${backup.type}`">
						<font-awesome-icon :icon="['fas', backup.type === 'automatic' ? 'robot' : 'user']" />
					</div>
					<div class="backup-info">
						<div class="backup-filename">{{ backup.filename }}</div>
						<div class="backup-meta">
							<span class="backup-type" :class="`type-${backup.type}`">
								{{ backup.type }}
							</span>
							<span class="backup-size">
								<font-awesome-icon :icon="['fas', 'hdd']" />
								{{ (backup.size / 1024 / 1024).toFixed(2) }} MB
							</span>
							<span class="backup-time">
								<font-awesome-icon :icon="['fas', 'clock']" />
								{{ formatTime(backup.created_at) }}
							</span>
						</div>
					</div>
					<div class="backup-actions">
						<button
							class="btn btn-sm btn-success"
							@click="downloadBackup(backup)"
							:title="'Download ' + backup.filename"
						>
							<font-awesome-icon :icon="['fas', 'download']" />
						</button>
						<button
							class="btn btn-sm btn-warning"
							@click="confirmRestore(backup)"
							:title="'Restore from ' + backup.filename"
						>
							<font-awesome-icon :icon="['fas', 'undo']" />
							Restore
						</button>
						<button
							class="btn btn-sm btn-danger"
							@click="confirmDelete(backup)"
							:title="'Delete ' + backup.filename"
						>
							<font-awesome-icon :icon="['fas', 'trash']" />
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Restore Confirmation Modal -->
		<div v-if="showRestoreModal" class="modal-overlay" @click="showRestoreModal = false">
			<div class="modal-content" @click.stop>
				<div class="modal-header">
					<h3><font-awesome-icon :icon="['fas', 'exclamation-triangle']" /> Confirm Restore</h3>
					<button class="close-btn" @click="showRestoreModal = false">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<p class="warning-text">
						<strong>WARNING:</strong> Restoring from a backup will replace your current database with the backup data.
						All changes made after <strong>{{ formatTime(selectedBackup?.created_at) }}</strong> will be lost.
					</p>
					<p>A safety backup of your current database will be created automatically before restoring.</p>
					<p>Are you sure you want to restore from <strong>{{ selectedBackup?.filename }}</strong>?</p>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="showRestoreModal = false">Cancel</button>
					<button class="btn btn-warning" @click="restoreBackup" :disabled="restoring">
						<font-awesome-icon :icon="['fas', restoring ? 'spinner' : 'undo']" :spin="restoring" />
						{{ restoring ? 'Restoring...' : 'Restore Database' }}
					</button>
				</div>
			</div>
		</div>

		<!-- Delete Confirmation Modal -->
		<div v-if="showDeleteModal" class="modal-overlay" @click="showDeleteModal = false">
			<div class="modal-content" @click.stop>
				<div class="modal-header">
					<h3><font-awesome-icon :icon="['fas', 'trash']" /> Confirm Delete</h3>
					<button class="close-btn" @click="showDeleteModal = false">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<p>Are you sure you want to delete <strong>{{ selectedBackup?.filename }}</strong>?</p>
					<p>This action cannot be undone.</p>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="showDeleteModal = false">Cancel</button>
					<button class="btn btn-danger" @click="deleteBackup" :disabled="deleting">
						<font-awesome-icon :icon="['fas', deleting ? 'spinner' : 'trash']" :spin="deleting" />
						{{ deleting ? 'Deleting...' : 'Delete Backup' }}
					</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, onMounted, getCurrentInstance } from 'vue'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast

const loading = ref(true)
const creatingBackup = ref(false)
const restoring = ref(false)
const deleting = ref(false)
const backups = ref([])
const stats = ref(null)
const selectedBackup = ref(null)
const showRestoreModal = ref(false)
const showDeleteModal = ref(false)

const loadBackups = async () => {
	loading.value = true
	try {
		const response = await fetch('http://localhost:8080/api/v1/backups')
		const data = await response.json()
		if (data.success) {
			backups.value = data.data.backups || []
		}
	} catch (error) {
		console.error('Failed to load backups:', error)
		toast.error('Failed to load backups')
	} finally {
		loading.value = false
	}
}

const loadStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/backups/stats')
		const data = await response.json()
		if (data.success) {
			stats.value = data.data
		}
	} catch (error) {
		console.error('Failed to load stats:', error)
	}
}

const createBackup = async () => {
	creatingBackup.value = true
	try {
		const response = await fetch('http://localhost:8080/api/v1/backups', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ type: 'manual' }),
		})
		const data = await response.json()
		if (data.success) {
			toast.success('Backup created successfully!')
			await loadBackups()
			await loadStats()
		} else {
			toast.error(data.error?.message || 'Failed to create backup')
		}
	} catch (error) {
		console.error('Failed to create backup:', error)
		toast.error('Failed to create backup')
	} finally {
		creatingBackup.value = false
	}
}

const confirmRestore = (backup) => {
	selectedBackup.value = backup
	showRestoreModal.value = true
}

const restoreBackup = async () => {
	restoring.value = true
	try {
		const response = await fetch(
			`http://localhost:8080/api/v1/backups/${selectedBackup.value.filename}/restore`,
			{
				method: 'POST',
			}
		)
		const data = await response.json()
		if (data.success) {
			toast.success('Database restored successfully! Please reload the page.')
			showRestoreModal.value = false
			setTimeout(() => window.location.reload(), 2000)
		} else {
			toast.error(data.error?.message || 'Failed to restore backup')
		}
	} catch (error) {
		console.error('Failed to restore backup:', error)
		toast.error('Failed to restore backup')
	} finally {
		restoring.value = false
	}
}

const confirmDelete = (backup) => {
	selectedBackup.value = backup
	showDeleteModal.value = true
}

const deleteBackup = async () => {
	deleting.value = true
	try {
		const response = await fetch(`http://localhost:8080/api/v1/backups/${selectedBackup.value.filename}`, {
			method: 'DELETE',
		})
		const data = await response.json()
		if (data.success) {
			toast.success('Backup deleted successfully')
			showDeleteModal.value = false
			await loadBackups()
			await loadStats()
		} else {
			toast.error(data.error?.message || 'Failed to delete backup')
		}
	} catch (error) {
		console.error('Failed to delete backup:', error)
		toast.error('Failed to delete backup')
	} finally {
		deleting.value = false
	}
}

const downloadBackup = (backup) => {
	window.open(`http://localhost:8080/api/v1/backups/${backup.filename}/download`, '_blank')
}

const formatTime = (dateStr) => {
	const date = new Date(dateStr)
	return date.toLocaleString()
}

onMounted(() => {
	loadBackups()
	loadStats()
})
</script>

<style scoped>
.backups-page {
	padding: 2rem;
	max-width: 1400px;
	margin: 0 auto;
}

.page-header {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
	margin-bottom: 2rem;
	padding-bottom: 1.5rem;
	border-bottom: 2px solid rgba(255, 255, 255, 0.1);
}

.header-content h1 {
	margin: 0;
	font-size: 2rem;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	background-clip: text;
}

.header-content p {
	margin: 0.5rem 0 0 0;
	color: #aaa;
}

.stats-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
	gap: 1.5rem;
	margin-bottom: 2rem;
}

.stat-card {
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 1.5rem;
	display: flex;
	gap: 1rem;
	align-items: center;
	transition: all 0.3s ease;
}

.stat-card:hover {
	background: rgba(255, 255, 255, 0.05);
	transform: translateY(-2px);
}

.stat-icon {
	width: 60px;
	height: 60px;
	border-radius: 12px;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.5rem;
	color: white;
}

.stat-content {
	flex: 1;
}

.stat-label {
	font-size: 0.9rem;
	color: #aaa;
	margin-bottom: 0.25rem;
}

.stat-value {
	font-size: 1.8rem;
	font-weight: 700;
	color: #e0e0e0;
}

.backups-section {
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 2rem;
}

.section-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 1.5rem;
}

.section-header h2 {
	margin: 0;
	color: #e0e0e0;
	font-size: 1.3rem;
}

.loading-state,
.empty-state {
	text-align: center;
	padding: 3rem;
	color: #aaa;
}

.empty-state svg {
	color: #555;
	margin-bottom: 1rem;
}

.backups-list {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.backup-card {
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: 1rem;
	background: rgba(255, 255, 255, 0.02);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 8px;
	transition: all 0.2s ease;
}

.backup-card:hover {
	background: rgba(255, 255, 255, 0.05);
	border-color: rgba(255, 255, 255, 0.2);
}

.backup-icon {
	width: 50px;
	height: 50px;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.2rem;
	flex-shrink: 0;
}

.backup-icon.icon-automatic {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	color: white;
}

.backup-icon.icon-manual {
	background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
	color: white;
}

.backup-info {
	flex: 1;
	min-width: 0;
}

.backup-filename {
	font-weight: 600;
	color: #e0e0e0;
	margin-bottom: 0.5rem;
	font-size: 0.95rem;
}

.backup-meta {
	display: flex;
	gap: 1rem;
	flex-wrap: wrap;
	font-size: 0.85rem;
	color: #aaa;
}

.backup-type {
	padding: 0.2rem 0.6rem;
	border-radius: 4px;
	font-weight: 600;
	text-transform: uppercase;
	font-size: 0.75rem;
}

.backup-type.type-automatic {
	background: rgba(103, 126, 234, 0.2);
	color: #667eea;
}

.backup-type.type-manual {
	background: rgba(240, 147, 251, 0.2);
	color: #f093fb;
}

.backup-actions {
	display: flex;
	gap: 0.5rem;
	flex-shrink: 0;
}

.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 2000;
	padding: 1rem;
}

.modal-content {
	background: rgba(30, 30, 40, 0.98);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	max-width: 600px;
	width: 100%;
	max-height: 90vh;
	overflow-y: auto;
	box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
}

.modal-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.modal-header h3 {
	margin: 0;
	color: #e0e0e0;
	font-size: 1.2rem;
}

.close-btn {
	background: none;
	border: none;
	color: #aaa;
	font-size: 1.2rem;
	cursor: pointer;
	padding: 0.5rem;
	transition: color 0.2s;
}

.close-btn:hover {
	color: #fff;
}

.modal-body {
	padding: 1.5rem;
	color: #ccc;
}

.warning-text {
	background: rgba(255, 152, 0, 0.1);
	border-left: 4px solid #ff9800;
	padding: 1rem;
	border-radius: 4px;
	margin-bottom: 1rem;
}

.modal-footer {
	display: flex;
	justify-content: flex-end;
	gap: 1rem;
	padding: 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}
</style>
