<template>
	<div class="scheduler-page">
		<div class="page-header">
			<h1>
				<font-awesome-icon :icon="['fas', 'clock']" />
				Automated Scheduler
			</h1>
			<div class="header-actions">
				<div class="scheduler-status" :class="{ 'status-active': schedulerRunning }">
					<font-awesome-icon :icon="['fas', schedulerRunning ? 'check-circle' : 'times-circle']" />
					<span>{{ schedulerRunning ? 'Running' : 'Stopped' }}</span>
				</div>
				<button class="btn btn-primary" @click="showCreateJobModal = true">
					<font-awesome-icon :icon="['fas', 'plus']" />
					Create Job
				</button>
			</div>
		</div>

		<div v-if="loading" class="loading-container">
			<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" />
			<p>Loading jobs...</p>
		</div>

		<div v-else class="jobs-container">
			<div v-if="jobs.length === 0" class="empty-state">
				<font-awesome-icon :icon="['fas', 'calendar-times']" size="4x" />
				<h3>No Scheduled Jobs</h3>
				<p>Create automated jobs to scrape threads, verify links, or clean up old data.</p>
				<button class="btn btn-primary" @click="showCreateJobModal = true">
					<font-awesome-icon :icon="['fas', 'plus']" />
					Create Your First Job
				</button>
			</div>

			<div v-else class="jobs-list">
				<div v-for="job in jobs" :key="job.id" class="job-card" :class="{ 'job-disabled': !job.enabled }">
					<div class="job-header">
						<div class="job-icon" :class="`icon-${job.job_type}`">
							<font-awesome-icon :icon="getJobIcon(job.job_type)" />
						</div>
						<div class="job-info">
							<h3 class="job-title">{{ getJobTitle(job.job_type) }}</h3>
							<div class="job-meta">
								<span class="job-schedule">
									<font-awesome-icon :icon="['fas', 'sync-alt']" />
									{{ getScheduleDescription(job) }}
								</span>
								<span v-if="job.target_type" class="job-target">
									<font-awesome-icon :icon="['fas', 'bullseye']" />
									{{ job.target_type }}: {{ job.target_id }}
								</span>
							</div>
						</div>
						<div class="job-actions">
							<button
								class="action-btn toggle-btn"
								:class="{ 'active': job.enabled }"
								@click="toggleJob(job.id)"
								:title="job.enabled ? 'Disable' : 'Enable'"
							>
								<font-awesome-icon :icon="['fas', job.enabled ? 'toggle-on' : 'toggle-off']" />
							</button>
							<button class="action-btn edit-btn" @click="editJob(job)" title="Edit">
								<font-awesome-icon :icon="['fas', 'edit']" />
							</button>
							<button class="action-btn delete-btn" @click="deleteJob(job.id)" title="Delete">
								<font-awesome-icon :icon="['fas', 'trash']" />
							</button>
						</div>
					</div>

					<div class="job-stats">
						<div class="stat-item">
							<span class="stat-label">Total Runs</span>
							<span class="stat-value">{{ job.run_count || 0 }}</span>
						</div>
						<div class="stat-item success">
							<span class="stat-label">Successful</span>
							<span class="stat-value">{{ job.success_count || 0 }}</span>
						</div>
						<div class="stat-item failure">
							<span class="stat-label">Failed</span>
							<span class="stat-value">{{ job.failure_count || 0 }}</span>
						</div>
						<div class="stat-item">
							<span class="stat-label">Last Run</span>
							<span class="stat-value">{{ formatDateTime(job.last_run_at) }}</span>
						</div>
						<div class="stat-item">
							<span class="stat-label">Next Run</span>
							<span class="stat-value">{{ formatDateTime(job.next_run_at) }}</span>
						</div>
					</div>

					<div class="job-footer">
						<button class="btn btn-sm btn-secondary" @click="loadJobHistory(job.id)">
							<font-awesome-icon :icon="['fas', 'history']" />
							View History
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Create/Edit Job Modal -->
		<div v-if="showCreateJobModal" class="modal-overlay" @click.self="closeModal">
			<div class="modal-content">
				<div class="modal-header">
					<h2>{{ editingJob ? 'Edit Job' : 'Create New Job' }}</h2>
					<button class="close-btn" @click="closeModal">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>

				<div class="modal-body">
					<div class="form-group">
						<label>Job Type</label>
						<select v-model="jobForm.job_type" class="form-control">
							<option value="scrape_thread">Scrape Thread</option>
							<option value="verify_links">Verify Links</option>
							<option value="cleanup_old_activities">Cleanup Old Activities</option>
							<option value="cleanup_old_audit_logs">Cleanup Old Audit Logs</option>
						</select>
					</div>

					<div class="form-group">
						<label>Schedule Type</label>
						<select v-model="jobForm.schedule_type" class="form-control">
							<option value="interval">Interval (Every X minutes)</option>
							<option value="once">Run Once</option>
						</select>
					</div>

					<div v-if="jobForm.schedule_type === 'interval'" class="form-group">
						<label>Interval (minutes)</label>
						<input
							v-model.number="jobForm.schedule_config.interval_minutes"
							type="number"
							class="form-control"
							min="5"
							placeholder="e.g., 60 for hourly"
						/>
					</div>

					<div v-if="jobForm.job_type === 'scrape_thread' || jobForm.job_type === 'verify_links'" class="form-group">
						<label>Target Thread ID</label>
						<input
							v-model.number="jobForm.target_id"
							type="number"
							class="form-control"
							placeholder="Thread ID to target"
						/>
					</div>

					<div v-if="jobForm.job_type.includes('cleanup')" class="form-group">
						<label>Days to Keep</label>
						<input
							v-model.number="jobForm.schedule_config.timeout_minutes"
							type="number"
							class="form-control"
							min="1"
							placeholder="e.g., 30 days"
						/>
					</div>

					<div class="form-group">
						<label class="checkbox-label">
							<input v-model="jobForm.enabled" type="checkbox" />
							<span>Enabled (job will run automatically)</span>
						</label>
					</div>
				</div>

				<div class="modal-footer">
					<button class="btn btn-secondary" @click="closeModal">Cancel</button>
					<button class="btn btn-primary" @click="saveJob" :disabled="!isFormValid">
						<font-awesome-icon :icon="['fas', 'save']" />
						{{ editingJob ? 'Update' : 'Create' }} Job
					</button>
				</div>
			</div>
		</div>

		<!-- Job History Modal -->
		<div v-if="showHistoryModal" class="modal-overlay" @click.self="showHistoryModal = false">
			<div class="modal-content modal-large">
				<div class="modal-header">
					<h2>Execution History</h2>
					<button class="close-btn" @click="showHistoryModal = false">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>

				<div class="modal-body">
					<div v-if="loadingHistory" class="loading-container">
						<font-awesome-icon :icon="['fas', 'spinner']" spin />
						<p>Loading history...</p>
					</div>

					<div v-else-if="jobHistory.length === 0" class="empty-state">
						<font-awesome-icon :icon="['fas', 'clock']" size="3x" />
						<p>No execution history yet</p>
					</div>

					<div v-else class="history-list">
						<div v-for="execution in jobHistory" :key="execution.id" class="history-item" :class="`status-${execution.status}`">
							<div class="history-status">
								<font-awesome-icon :icon="getExecutionStatusIcon(execution.status)" />
							</div>
							<div class="history-info">
								<div class="history-meta">
									<span class="history-date">{{ formatDateTime(execution.started_at) }}</span>
									<span v-if="execution.completed_at" class="history-duration">
										{{ execution.duration_ms }}ms
									</span>
								</div>
								<div v-if="execution.error_message" class="history-error">
									{{ execution.error_message }}
								</div>
								<div v-if="execution.result_data" class="history-result">
									{{ parseResultData(execution.result_data) }}
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, getCurrentInstance } from 'vue'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast

const loading = ref(true)
const schedulerRunning = ref(false)
const jobs = ref([])
const showCreateJobModal = ref(false)
const showHistoryModal = ref(false)
const editingJob = ref(null)
const loadingHistory = ref(false)
const jobHistory = ref([])

const jobForm = ref({
	job_type: 'scrape_thread',
	schedule_type: 'interval',
	schedule_config: {
		interval_minutes: 60,
		timeout_minutes: 30
	},
	target_type: '',
	target_id: null,
	enabled: true
})

const isFormValid = computed(() => {
	if (!jobForm.value.job_type || !jobForm.value.schedule_type) {
		return false
	}

	if (jobForm.value.schedule_type === 'interval' && !jobForm.value.schedule_config.interval_minutes) {
		return false
	}

	if ((jobForm.value.job_type === 'scrape_thread' || jobForm.value.job_type === 'verify_links') && !jobForm.value.target_id) {
		return false
	}

	return true
})

const loadSchedulerStatus = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scheduler/status')
		const data = await response.json()
		if (data.success) {
			schedulerRunning.value = data.data.running || false
		}
	} catch (error) {
		console.error('Failed to load scheduler status:', error)
	}
}

const loadJobs = async () => {
	loading.value = true
	try {
		const response = await fetch('http://localhost:8080/api/v1/scheduler/jobs')
		const data = await response.json()
		if (data.success) {
			jobs.value = data.data.jobs || []
		}
	} catch (error) {
		console.error('Failed to load jobs:', error)
		toast.error('Failed to load jobs')
	} finally {
		loading.value = false
	}
}

const saveJob = async () => {
	try {
		// Set target_type based on job_type
		if (jobForm.value.job_type === 'scrape_thread' || jobForm.value.job_type === 'verify_links') {
			jobForm.value.target_type = 'thread'
		}

		const url = editingJob.value
			? `http://localhost:8080/api/v1/scheduler/jobs/${editingJob.value.id}`
			: 'http://localhost:8080/api/v1/scheduler/jobs'

		const method = editingJob.value ? 'PUT' : 'POST'

		const response = await fetch(url, {
			method,
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(jobForm.value)
		})

		const data = await response.json()
		if (data.success) {
			toast.success(editingJob.value ? 'Job updated successfully!' : 'Job created successfully!')
			closeModal()
			await loadJobs()
		} else {
			toast.error(data.error || 'Failed to save job')
		}
	} catch (error) {
		console.error('Failed to save job:', error)
		toast.error('Failed to save job')
	}
}

const editJob = (job) => {
	editingJob.value = job

	// Parse schedule config
	let config = {}
	try {
		config = JSON.parse(job.schedule_config)
	} catch (e) {
		config = {}
	}

	jobForm.value = {
		job_type: job.job_type,
		schedule_type: job.schedule_type,
		schedule_config: {
			interval_minutes: config.interval_minutes || 60,
			timeout_minutes: config.timeout_minutes || 30
		},
		target_type: job.target_type || '',
		target_id: job.target_id,
		enabled: job.enabled
	}

	showCreateJobModal.value = true
}

const toggleJob = async (jobId) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scheduler/jobs/${jobId}/toggle`, {
			method: 'POST'
		})
		const data = await response.json()
		if (data.success) {
			toast.success(data.data.enabled ? 'Job enabled' : 'Job disabled')
			await loadJobs()
		} else {
			toast.error('Failed to toggle job')
		}
	} catch (error) {
		console.error('Failed to toggle job:', error)
		toast.error('Failed to toggle job')
	}
}

const deleteJob = async (jobId) => {
	if (!confirm('Are you sure you want to delete this job?')) {
		return
	}

	try {
		const response = await fetch(`http://localhost:8080/api/v1/scheduler/jobs/${jobId}`, {
			method: 'DELETE'
		})
		const data = await response.json()
		if (data.success) {
			toast.success('Job deleted successfully!')
			await loadJobs()
		} else {
			toast.error('Failed to delete job')
		}
	} catch (error) {
		console.error('Failed to delete job:', error)
		toast.error('Failed to delete job')
	}
}

const loadJobHistory = async (jobId) => {
	showHistoryModal.value = true
	loadingHistory.value = true
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scheduler/jobs/${jobId}/history?limit=50`)
		const data = await response.json()
		if (data.success) {
			jobHistory.value = data.data.history || []
		}
	} catch (error) {
		console.error('Failed to load job history:', error)
		toast.error('Failed to load job history')
	} finally {
		loadingHistory.value = false
	}
}

const closeModal = () => {
	showCreateJobModal.value = false
	editingJob.value = null
	jobForm.value = {
		job_type: 'scrape_thread',
		schedule_type: 'interval',
		schedule_config: {
			interval_minutes: 60,
			timeout_minutes: 30
		},
		target_type: '',
		target_id: null,
		enabled: true
	}
}

const getJobIcon = (jobType) => {
	const icons = {
		'scrape_thread': ['fas', 'spider'],
		'verify_links': ['fas', 'check-double'],
		'cleanup_old_activities': ['fas', 'broom'],
		'cleanup_old_audit_logs': ['fas', 'trash-alt']
	}
	return icons[jobType] || ['fas', 'cog']
}

const getJobTitle = (jobType) => {
	const titles = {
		'scrape_thread': 'Scrape Thread',
		'verify_links': 'Verify Links',
		'cleanup_old_activities': 'Cleanup Old Activities',
		'cleanup_old_audit_logs': 'Cleanup Old Audit Logs'
	}
	return titles[jobType] || jobType
}

const getScheduleDescription = (job) => {
	try {
		const config = JSON.parse(job.schedule_config)
		if (job.schedule_type === 'interval') {
			const minutes = config.interval_minutes
			if (minutes >= 1440) {
				return `Every ${Math.floor(minutes / 1440)} day(s)`
			} else if (minutes >= 60) {
				return `Every ${Math.floor(minutes / 60)} hour(s)`
			} else {
				return `Every ${minutes} minute(s)`
			}
		} else if (job.schedule_type === 'once') {
			return 'Run once'
		}
		return job.schedule_type
	} catch (e) {
		return job.schedule_type
	}
}

const getExecutionStatusIcon = (status) => {
	if (status === 'completed') return ['fas', 'check-circle']
	if (status === 'failed') return ['fas', 'times-circle']
	return ['fas', 'spinner']
}

const formatDateTime = (dateStr) => {
	if (!dateStr) return 'Never'
	const date = new Date(dateStr)
	return date.toLocaleString()
}

const parseResultData = (resultData) => {
	if (!resultData) return ''
	try {
		const result = JSON.parse(resultData)
		return result.message || JSON.stringify(result)
	} catch (e) {
		return resultData
	}
}

onMounted(() => {
	loadSchedulerStatus()
	loadJobs()
})
</script>

<style scoped>
.scheduler-page {
	padding: 2rem;
	min-height: 100vh;
}

.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 2rem;
}

.page-header h1 {
	font-size: 2rem;
	font-weight: 700;
	display: flex;
	align-items: center;
	gap: 1rem;
	color: #e0e0e0;
}

.header-actions {
	display: flex;
	align-items: center;
	gap: 1rem;
}

.scheduler-status {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.5rem 1rem;
	border-radius: 8px;
	background: rgba(255, 255, 255, 0.05);
	backdrop-filter: blur(10px);
	border: 1px solid rgba(255, 255, 255, 0.1);
	color: #aaa;
}

.scheduler-status.status-active {
	background: rgba(76, 175, 80, 0.1);
	border-color: rgba(76, 175, 80, 0.3);
	color: #4caf50;
}

.loading-container {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	padding: 4rem;
	gap: 1rem;
	color: #aaa;
}

.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	padding: 4rem;
	text-align: center;
	color: #aaa;
}

.empty-state svg {
	color: #555;
	margin-bottom: 1rem;
}

.empty-state h3 {
	font-size: 1.5rem;
	margin-bottom: 0.5rem;
	color: #e0e0e0;
}

.jobs-list {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(500px, 1fr));
	gap: 1.5rem;
}

.job-card {
	background: rgba(255, 255, 255, 0.03);
	backdrop-filter: blur(20px);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 1.5rem;
	transition: all 0.3s ease;
}

.job-card:hover {
	background: rgba(255, 255, 255, 0.05);
	border-color: rgba(255, 255, 255, 0.2);
	transform: translateY(-2px);
}

.job-card.job-disabled {
	opacity: 0.5;
}

.job-header {
	display: flex;
	gap: 1rem;
	margin-bottom: 1rem;
}

.job-icon {
	width: 50px;
	height: 50px;
	border-radius: 10px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.5rem;
	flex-shrink: 0;
}

.job-icon.icon-scrape_thread {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.job-icon.icon-verify_links {
	background: linear-gradient(135deg, #4caf50 0%, #2196f3 100%);
}

.job-icon.icon-cleanup_old_activities {
	background: linear-gradient(135deg, #ff9800 0%, #ff5722 100%);
}

.job-icon.icon-cleanup_old_audit_logs {
	background: linear-gradient(135deg, #f44336 0%, #e91e63 100%);
}

.job-info {
	flex: 1;
}

.job-title {
	font-size: 1.1rem;
	font-weight: 600;
	margin: 0 0 0.5rem 0;
	color: #e0e0e0;
}

.job-meta {
	display: flex;
	gap: 1rem;
	font-size: 0.9rem;
	color: #aaa;
}

.job-meta span {
	display: flex;
	align-items: center;
	gap: 0.3rem;
}

.job-actions {
	display: flex;
	gap: 0.5rem;
}

.action-btn {
	width: 35px;
	height: 35px;
	border-radius: 8px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	background: rgba(255, 255, 255, 0.05);
	color: #aaa;
	cursor: pointer;
	display: flex;
	align-items: center;
	justify-content: center;
	transition: all 0.2s ease;
}

.action-btn:hover {
	background: rgba(255, 255, 255, 0.1);
	border-color: rgba(255, 255, 255, 0.2);
}

.toggle-btn.active {
	background: rgba(76, 175, 80, 0.2);
	border-color: rgba(76, 175, 80, 0.4);
	color: #4caf50;
}

.delete-btn:hover {
	background: rgba(244, 67, 54, 0.2);
	border-color: rgba(244, 67, 54, 0.4);
	color: #f44336;
}

.job-stats {
	display: grid;
	grid-template-columns: repeat(5, 1fr);
	gap: 1rem;
	padding: 1rem 0;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
	margin-top: 1rem;
}

.stat-item {
	text-align: center;
}

.stat-label {
	display: block;
	font-size: 0.75rem;
	color: #aaa;
	margin-bottom: 0.3rem;
}

.stat-value {
	display: block;
	font-size: 1.1rem;
	font-weight: 600;
	color: #e0e0e0;
}

.stat-item.success .stat-value {
	color: #4caf50;
}

.stat-item.failure .stat-value {
	color: #f44336;
}

.job-footer {
	margin-top: 1rem;
	padding-top: 1rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.8);
	backdrop-filter: blur(5px);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1000;
	padding: 2rem;
}

.modal-content {
	background: rgba(30, 30, 30, 0.98);
	backdrop-filter: blur(20px);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 16px;
	width: 100%;
	max-width: 600px;
	max-height: 90vh;
	overflow-y: auto;
}

.modal-content.modal-large {
	max-width: 800px;
}

.modal-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.modal-header h2 {
	margin: 0;
	color: #e0e0e0;
}

.close-btn {
	background: none;
	border: none;
	color: #aaa;
	font-size: 1.5rem;
	cursor: pointer;
	padding: 0;
	width: 30px;
	height: 30px;
	display: flex;
	align-items: center;
	justify-content: center;
}

.close-btn:hover {
	color: #fff;
}

.modal-body {
	padding: 1.5rem;
}

.form-group {
	margin-bottom: 1.5rem;
}

.form-group label {
	display: block;
	margin-bottom: 0.5rem;
	color: #e0e0e0;
	font-weight: 500;
}

.form-control {
	width: 100%;
	padding: 0.75rem;
	border-radius: 8px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	background: rgba(255, 255, 255, 0.05);
	color: #e0e0e0;
	font-size: 1rem;
}

.form-control:focus {
	outline: none;
	border-color: rgba(103, 126, 234, 0.5);
	background: rgba(255, 255, 255, 0.08);
}

.checkbox-label {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	cursor: pointer;
}

.checkbox-label input {
	width: 20px;
	height: 20px;
}

.modal-footer {
	display: flex;
	justify-content: flex-end;
	gap: 1rem;
	padding: 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.history-list {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.history-item {
	display: flex;
	gap: 1rem;
	padding: 1rem;
	border-radius: 8px;
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
}

.history-item.status-completed {
	border-left: 3px solid #4caf50;
}

.history-item.status-failed {
	border-left: 3px solid #f44336;
}

.history-status {
	font-size: 1.5rem;
}

.history-status svg {
	color: #4caf50;
}

.history-item.status-failed .history-status svg {
	color: #f44336;
}

.history-info {
	flex: 1;
}

.history-meta {
	display: flex;
	gap: 1rem;
	font-size: 0.9rem;
	color: #aaa;
	margin-bottom: 0.5rem;
}

.history-error {
	color: #f44336;
	font-size: 0.9rem;
}

.history-result {
	color: #aaa;
	font-size: 0.9rem;
}

.btn {
	padding: 0.75rem 1.5rem;
	border-radius: 8px;
	border: none;
	font-size: 1rem;
	font-weight: 500;
	cursor: pointer;
	display: inline-flex;
	align-items: center;
	gap: 0.5rem;
	transition: all 0.2s ease;
}

.btn-primary {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	color: white;
}

.btn-primary:hover {
	transform: translateY(-2px);
	box-shadow: 0 5px 20px rgba(103, 126, 234, 0.4);
}

.btn-primary:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

.btn-secondary {
	background: rgba(255, 255, 255, 0.1);
	color: #e0e0e0;
	border: 1px solid rgba(255, 255, 255, 0.2);
}

.btn-secondary:hover {
	background: rgba(255, 255, 255, 0.15);
}

.btn-sm {
	padding: 0.5rem 1rem;
	font-size: 0.9rem;
}
</style>
