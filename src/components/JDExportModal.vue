<template>
	<div class="modal-backdrop" @click.self="$emit('close')">
		<div class="modal-content">
			<div class="modal-header">
				<h5 class="modal-title">
					<font-awesome-icon :icon="['fas', 'file-export']" class="me-2" />
					Export Links
				</h5>
				<button class="btn-close" @click="$emit('close')"></button>
			</div>
			<div class="modal-body">
				<div class="mb-3">
					<label class="form-label">Exported URLs</label>
					<textarea :value="text" class="form-control" rows="10" readonly></textarea>
				</div>
				<div class="d-flex gap-2">
					<button class="btn btn-primary" @click="copyLinks">
						<font-awesome-icon :icon="['fas', 'copy']" class="me-1" />
						Copy to Clipboard
					</button>
					<button class="btn btn-outline-primary" @click="downloadLinks">
						<font-awesome-icon :icon="['fas', 'download']" class="me-1" />
						Download as File
					</button>
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" @click="$emit('close')">Close</button>
			</div>
		</div>
	</div>
</template>

<script setup>
const props = defineProps({
	text: { type: String, default: '' },
})
const emit = defineEmits(['close', 'toast'])

const copyLinks = async () => {
	try {
		await navigator.clipboard.writeText(props.text)
		emit('toast', 'Links copied to clipboard', 'success')
	} catch {
		emit('toast', 'Failed to copy links', 'error')
	}
}

const downloadLinks = () => {
	const blob = new Blob([props.text], { type: 'text/plain' })
	const url = URL.createObjectURL(blob)
	const a = document.createElement('a')
	a.href = url
	a.download = `jdownloader-links-${new Date().toISOString().slice(0, 10)}.txt`
	a.click()
	URL.revokeObjectURL(url)
	emit('toast', 'Links downloaded', 'success')
}
</script>

<style scoped>
.modal-backdrop {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1050;
}

.modal-content {
	background: #1a1a2e;
	border-radius: 12px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	width: 90%;
	max-width: 600px;
	max-height: 90vh;
	overflow: auto;
}

.modal-header {
	padding: 1rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.modal-title {
	margin: 0;
	color: #fff;
}

.btn-close {
	background: transparent;
	border: none;
	color: rgba(255, 255, 255, 0.6);
	font-size: 1.5rem;
	cursor: pointer;
	padding: 0;
	line-height: 1;
}

.btn-close:hover {
	color: #fff;
}

.btn-close::before {
	content: '×';
}

.modal-body {
	padding: 1.25rem;
}

.modal-footer {
	padding: 1rem 1.25rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	justify-content: flex-end;
	gap: 0.5rem;
}

.form-label {
	color: rgba(255, 255, 255, 0.8);
}

.form-control {
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.2);
	color: #fff;
}
</style>
