<template>
	<div class="bannerlord-panel">
		<!-- Header: campaign selector -->
		<div class="bl-header">
			<div class="bl-title">
				<font-awesome-icon :icon="['fas', 'horse']" />
				<span>Bannerlord Campaigns</span>
			</div>
			<div class="bl-controls">
				<select v-model="selectedCampaign" class="form-select form-select-sm bl-select" :disabled="loadingCampaigns">
					<option value="" disabled>Select campaign...</option>
					<option v-for="c in campaigns" :key="c.id" :value="c.id">
						{{ c.id }} — {{ c.npc_count }} NPCs · {{ formatTime(c.last_modified) }}
					</option>
				</select>
				<button class="btn btn-sm btn-outline-info" @click="refresh" :disabled="loadingOverview">
					<font-awesome-icon :icon="['fas', 'sync-alt']" :spin="loadingOverview" />
				</button>
			</div>
		</div>

		<div v-if="loadingCampaigns" class="bl-empty">Scanning campaign folders...</div>
		<div v-else-if="campaigns.length === 0" class="bl-empty">
			No campaigns found in <code>{{ modPathLabel }}</code>
		</div>

		<template v-if="overview">
			<!-- Stats row -->
			<div class="bl-stats">
				<div class="bl-stat">
					<span class="bl-stat-value">{{ overview.npc_count }}</span>
					<span class="bl-stat-label">NPCs</span>
				</div>
				<div class="bl-stat watched">
					<span class="bl-stat-value">{{ overview.watched_count }}</span>
					<span class="bl-stat-label">Watched</span>
				</div>
				<div class="bl-stat">
					<span class="bl-stat-value">{{ (overview.dynamic_events || []).length }}</span>
					<span class="bl-stat-label">Events</span>
				</div>
				<div class="bl-stat">
					<span class="bl-stat-value">{{ atWarCount }}</span>
					<span class="bl-stat-label">At War</span>
				</div>
				<div class="bl-stat">
					<span class="bl-stat-value">{{ overview.alliance_count || 0 }}</span>
					<span class="bl-stat-label">Alliances</span>
				</div>
			</div>

			<div class="bl-grid">
				<!-- NPC Roster -->
				<div class="bl-section bl-roster">
					<div class="bl-section-header">
						NPC Roster
						<span class="bl-section-sub">click 👁 to watch — the Companion follows changes live</span>
					</div>
					<div class="bl-npc-grid">
						<div v-for="npc in npcs" :key="npc.file_path" class="bl-npc-card" :class="{ watched: npc.watched }">
							<div class="bl-npc-top">
								<span class="bl-npc-name" :title="npc.string_id">{{ npc.name || npc.file_name }}</span>
								<button class="bl-watch-btn" :class="{ on: npc.watched }" :title="npc.watched ? 'Unwatch — Companion stops following this NPC' : 'Watch — Companion follows this NPC'" @click="toggleWatch(npc)">
									<font-awesome-icon :icon="['fas', npc.watched ? 'eye' : 'eye-slash']" />
								</button>
							</div>
							<div class="bl-npc-badges">
								<span v-if="npc.relation_desc" class="badge" :class="relationClass(npc.relation_value)">{{ npc.relation_desc }} {{ npc.relation_value }}</span>
								<span v-if="npc.mood" class="badge bg-dark">{{ npc.mood }}</span>
							</div>
							<div v-if="npc.current_task" class="bl-npc-task">
								<font-awesome-icon :icon="['fas', 'route']" /> {{ npc.current_task }}
							</div>
							<p v-if="npc.last_response" class="bl-npc-response">{{ npc.last_response }}</p>
							<div v-if="npc.parse_error" class="bl-npc-error">⚠ {{ npc.parse_error }}</div>
						</div>
					</div>
				</div>

				<!-- Diplomacy & Wars -->
				<div class="bl-side">
					<!-- Dynamic events -->
					<div class="bl-section">
						<div class="bl-section-header">Dynamic Events</div>
						<div v-if="(overview.dynamic_events || []).length === 0" class="bl-empty-sm">No active events</div>
						<div v-for="ev in overview.dynamic_events" :key="ev.id" class="bl-event">
							<div class="bl-event-top">
								<span class="bl-badge-type" :class="'type-' + ev.type">{{ ev.type }}</span>
								<span class="bl-badge-imp" :class="{ high: ev.importance >= 7 }">★{{ ev.importance }}</span>
								<span v-if="ev.player_involved" class="badge bg-info text-dark">you</span>
							</div>
							<div class="bl-event-title">{{ ev.title }}</div>
							<div class="bl-event-desc">{{ ev.description }}</div>
							<div v-if="ev.kingdoms_involved?.length" class="bl-event-kingdoms">
								<span v-for="k in ev.kingdoms_involved" :key="k" class="bl-kingdom-chip">{{ k }}</span>
							</div>
						</div>
					</div>

					<!-- Kingdom statements -->
					<div class="bl-section">
						<div class="bl-section-header">Ruler Statements</div>
						<div v-if="(overview.kingdom_statements || []).length === 0" class="bl-empty-sm">No recent statements</div>
						<div v-for="(st, i) in overview.kingdom_statements" :key="i" class="bl-statement">
							<div class="bl-statement-top">
								<span class="bl-kingdom-chip">{{ st.kingdom_id }}</span>
								<span v-if="st.actions?.length" class="badge bg-warning text-dark">{{ st.actions.join(', ') }}</span>
								<span v-else class="badge bg-secondary">{{ st.action || 'statement' }}</span>
								<span v-if="st.target_kingdom" class="bl-kingdom-chip">→ {{ st.target_kingdom }}</span>
							</div>
							<p class="bl-statement-text">{{ st.statement_text }}</p>
						</div>
					</div>

					<!-- Kingdom wars -->
					<div class="bl-section">
						<div class="bl-section-header">Kingdom Strength</div>
						<div v-if="(overview.kingdom_wars || []).length === 0" class="bl-empty-sm">No war statistics</div>
						<table v-else class="bl-wars-table">
							<thead>
								<tr><th>Kingdom</th><th>Troops</th><th>Casualties</th><th>Fatigue</th><th>At War</th></tr>
							</thead>
							<tbody>
								<tr v-for="w in sortedWars" :key="w.kingdom">
									<td class="bl-war-kingdom">{{ w.kingdom }}</td>
									<td>{{ w.current_troops?.toLocaleString() }}</td>
									<td>{{ w.total_casualties?.toLocaleString() }}</td>
									<td>
										<span class="bl-fatigue" :class="fatigueClass(w.war_fatigue)">{{ Math.round(w.war_fatigue || 0) }}</span>
									</td>
									<td>{{ w.days_at_war ? w.days_at_war + 'd' : '—' }}</td>
								</tr>
							</tbody>
						</table>
					</div>
				</div>
			</div>
		</template>
	</div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { bannerlordAPI, aiAPI } from '@/services/api'

const campaigns = ref([])
const selectedCampaign = ref(localStorage.getItem('bannerlord_selected_campaign') || '')
const overview = ref(null)
const loadingCampaigns = ref(false)
const loadingOverview = ref(false)

const modPathLabel = '.../Modules/AIInfluence/save_data'

const npcs = computed(() => overview.value?.npcs || [])
const atWarCount = computed(() => (overview.value?.kingdom_wars || []).filter((w) => (w.days_at_war || 0) > 0).length)
const sortedWars = computed(() => [...(overview.value?.kingdom_wars || [])].sort((a, b) => (b.current_troops || 0) - (a.current_troops || 0)))

const relationClass = (val) => {
	if ((val || 0) >= 20) return 'bg-success'
	if ((val || 0) <= -20) return 'bg-danger'
	return 'bg-secondary'
}

const fatigueClass = (val) => {
	if ((val || 0) >= 70) return 'critical'
	if ((val || 0) >= 40) return 'high'
	return ''
}

const formatTime = (ts) => {
	if (!ts) return ''
	const d = new Date(ts)
	return d.toLocaleDateString([], { month: 'short', day: 'numeric' }) + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const loadCampaigns = async () => {
	loadingCampaigns.value = true
	try {
		const res = await bannerlordAPI.getCampaigns()
		campaigns.value = res?.data || []
		// Auto-select: remembered choice, else most recent campaign
		if (selectedCampaign.value && !campaigns.value.some((c) => c.id === selectedCampaign.value)) {
			selectedCampaign.value = ''
		}
		if (!selectedCampaign.value && campaigns.value.length > 0) {
			selectedCampaign.value = campaigns.value[0].id
		}
	} catch {
		campaigns.value = []
	} finally {
		loadingCampaigns.value = false
	}
}

const loadOverview = async () => {
	if (!selectedCampaign.value) return
	loadingOverview.value = true
	try {
		const res = await bannerlordAPI.getCampaignOverview(selectedCampaign.value)
		overview.value = res?.data || null
	} catch {
		overview.value = null
	} finally {
		loadingOverview.value = false
	}
}

const toggleWatch = async (npc) => {
	try {
		if (npc.watched) {
			if (npc.watch_id) await aiAPI.removeRoleplayWatch(npc.watch_id)
		} else {
			await aiAPI.addRoleplayWatch(npc.file_path)
		}
		await loadOverview()
	} catch { /* ignore */ }
}

const refresh = async () => {
	await Promise.all([loadCampaigns(), loadOverview()])
}

watch(selectedCampaign, (id) => {
	if (id) {
		localStorage.setItem('bannerlord_selected_campaign', id)
		loadOverview()
	}
})

onMounted(loadCampaigns)
</script>

<style scoped>
.bannerlord-panel {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.bl-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	flex-wrap: wrap;
	gap: 0.75rem;
}

.bl-title {
	display: flex;
	align-items: center;
	gap: 0.6rem;
	font-size: 1.25rem;
	font-weight: 700;
	color: #00d9ff;
}

.bl-controls {
	display: flex;
	gap: 0.5rem;
	align-items: center;
}

.bl-select {
	min-width: 320px;
	background: rgba(0, 0, 0, 0.3);
	border: 1px solid rgba(255, 255, 255, 0.15);
	color: #fff;
}

.bl-empty {
	text-align: center;
	padding: 3rem 1rem;
	color: rgba(255, 255, 255, 0.4);
}

.bl-empty-sm {
	font-size: 0.8rem;
	color: rgba(255, 255, 255, 0.35);
	padding: 0.4rem 0;
}

.bl-stats {
	display: flex;
	gap: 0.75rem;
	flex-wrap: wrap;
}

.bl-stat {
	background: rgba(255, 255, 255, 0.04);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 10px;
	padding: 0.6rem 1.2rem;
	display: flex;
	flex-direction: column;
	align-items: center;
	min-width: 90px;
}

.bl-stat.watched {
	border-color: rgba(0, 217, 255, 0.4);
}

.bl-stat-value {
	font-size: 1.4rem;
	font-weight: 700;
	color: #fff;
}

.bl-stat.watched .bl-stat-value {
	color: #00d9ff;
}

.bl-stat-label {
	font-size: 0.7rem;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	color: rgba(255, 255, 255, 0.5);
}

.bl-grid {
	display: grid;
	grid-template-columns: 7fr 5fr;
	gap: 1rem;
	align-items: start;
}

@media (max-width: 1199.98px) {
	.bl-grid {
		grid-template-columns: 1fr;
	}
}

.bl-section {
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 1rem;
	margin-bottom: 1rem;
}

.bl-section-header {
	font-weight: 700;
	color: #00d9ff;
	margin-bottom: 0.75rem;
	display: flex;
	align-items: baseline;
	gap: 0.6rem;
}

.bl-section-sub {
	font-size: 0.72rem;
	font-weight: 400;
	color: rgba(255, 255, 255, 0.4);
}

/* NPC roster */
.bl-npc-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
	gap: 0.6rem;
}

.bl-npc-card {
	background: rgba(0, 0, 0, 0.25);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-radius: 10px;
	padding: 0.6rem 0.75rem;
	transition: border-color 0.2s;
}

.bl-npc-card.watched {
	border-color: rgba(0, 217, 255, 0.45);
	background: rgba(0, 217, 255, 0.04);
}

.bl-npc-top {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 0.35rem;
}

.bl-npc-name {
	font-weight: 700;
	color: #fff;
	font-size: 0.9rem;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.bl-watch-btn {
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.3);
	cursor: pointer;
	font-size: 0.95rem;
	padding: 0 0.15rem;
	transition: color 0.2s;
}

.bl-watch-btn:hover {
	color: rgba(255, 255, 255, 0.7);
}

.bl-watch-btn.on {
	color: #00d9ff;
}

.bl-npc-badges {
	display: flex;
	flex-wrap: wrap;
	gap: 0.3rem;
	margin-bottom: 0.35rem;
}

.bl-npc-task {
	font-size: 0.72rem;
	color: rgba(255, 255, 255, 0.5);
	margin-bottom: 0.3rem;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.bl-npc-response {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.65);
	margin: 0;
	display: -webkit-box;
	-webkit-line-clamp: 3;
	-webkit-box-orient: vertical;
	overflow: hidden;
}

.bl-npc-error {
	font-size: 0.7rem;
	color: #dc3545;
}

/* Diplomacy side */
.bl-side {
	display: flex;
	flex-direction: column;
}

.bl-event {
	background: rgba(0, 0, 0, 0.25);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-left: 3px solid rgba(255, 255, 255, 0.2);
	border-radius: 8px;
	padding: 0.55rem 0.75rem;
	margin-bottom: 0.5rem;
}

.bl-event-top {
	display: flex;
	align-items: center;
	gap: 0.4rem;
	margin-bottom: 0.3rem;
}

.bl-badge-type {
	font-size: 0.65rem;
	font-weight: 700;
	text-transform: uppercase;
	letter-spacing: 0.04em;
	padding: 0.1rem 0.4rem;
	border-radius: 4px;
	background: rgba(255, 255, 255, 0.1);
	color: rgba(255, 255, 255, 0.7);
}

.bl-badge-type.type-political { background: rgba(111, 66, 193, 0.25); color: #b794f4; }
.bl-badge-type.type-military { background: rgba(220, 53, 69, 0.2); color: #ff8a8a; }
.bl-badge-type.type-economic { background: rgba(255, 193, 7, 0.15); color: #ffd54f; }
.bl-badge-type.type-social { background: rgba(40, 167, 69, 0.15); color: #7ee2a8; }
.bl-badge-type.type-rumor { background: rgba(255, 255, 255, 0.08); color: rgba(255, 255, 255, 0.6); }

.bl-badge-imp {
	font-size: 0.7rem;
	color: rgba(255, 255, 255, 0.45);
}

.bl-badge-imp.high {
	color: #ffc107;
	font-weight: 700;
}

.bl-event-title {
	font-weight: 600;
	color: #fff;
	font-size: 0.85rem;
}

.bl-event-desc {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.55);
	margin-top: 0.2rem;
	display: -webkit-box;
	-webkit-line-clamp: 3;
	-webkit-box-orient: vertical;
	overflow: hidden;
}

.bl-event-kingdoms {
	display: flex;
	flex-wrap: wrap;
	gap: 0.3rem;
	margin-top: 0.35rem;
}

.bl-kingdom-chip {
	font-size: 0.68rem;
	background: rgba(0, 217, 255, 0.1);
	border: 1px solid rgba(0, 217, 255, 0.25);
	color: #00d9ff;
	border-radius: 8px;
	padding: 0.1rem 0.45rem;
}

.bl-statement {
	background: rgba(0, 0, 0, 0.25);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-radius: 8px;
	padding: 0.55rem 0.75rem;
	margin-bottom: 0.5rem;
}

.bl-statement-top {
	display: flex;
	align-items: center;
	flex-wrap: wrap;
	gap: 0.4rem;
	margin-bottom: 0.3rem;
}

.bl-statement-text {
	font-size: 0.78rem;
	color: rgba(255, 255, 255, 0.65);
	margin: 0;
	display: -webkit-box;
	-webkit-line-clamp: 3;
	-webkit-box-orient: vertical;
	overflow: hidden;
}

/* Wars table */
.bl-wars-table {
	width: 100%;
	font-size: 0.78rem;
	border-collapse: collapse;
}

.bl-wars-table th {
	text-align: left;
	font-size: 0.68rem;
	text-transform: uppercase;
	letter-spacing: 0.04em;
	color: rgba(255, 255, 255, 0.4);
	padding: 0.3rem 0.4rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.bl-wars-table td {
	padding: 0.35rem 0.4rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.05);
	color: rgba(255, 255, 255, 0.8);
	font-variant-numeric: tabular-nums;
}

.bl-war-kingdom {
	font-weight: 600;
	color: #fff;
	text-transform: capitalize;
}

.bl-fatigue {
	padding: 0.05rem 0.4rem;
	border-radius: 4px;
	background: rgba(40, 167, 69, 0.15);
	color: #7ee2a8;
}

.bl-fatigue.high {
	background: rgba(255, 193, 7, 0.15);
	color: #ffd54f;
}

.bl-fatigue.critical {
	background: rgba(220, 53, 69, 0.2);
	color: #ff8a8a;
	font-weight: 700;
}
</style>
