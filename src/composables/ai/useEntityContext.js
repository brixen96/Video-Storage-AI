/**
 * useEntityContext — detects page context (performer/video) and builds greeting cards.
 */
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { performersAPI, videosAPI } from '@/services/api'
import { PAGE_LABELS } from './constants'

export function useEntityContext() {
	const router = useRouter()
	const entityContext = ref(null) // string like "Jessica Jones (12 videos)"
	const entityCard = ref(null)    // { type, icon, name, fields: [{icon, value}] }

	const _sessionGreeted = new Set(
		JSON.parse(sessionStorage.getItem('ai_entity_greeted') || '[]'),
	)

	const _fmtDuration = (secs) => {
		const s = Math.floor(secs)
		const h = Math.floor(s / 3600)
		const m = Math.floor((s % 3600) / 60)
		const sec = s % 60
		return h > 0 ? `${h}h ${m}m` : `${m}:${sec.toString().padStart(2, '0')}`
	}

	const _ageFromDate = (dateStr) => {
		if (!dateStr) return null
		const age = Math.floor(
			(Date.now() - new Date(dateStr).getTime()) / (365.25 * 24 * 60 * 60 * 1000),
		)
		return age > 0 && age < 120 ? age : null
	}

	const loadEntityContext = async (route) => {
		entityContext.value = null
		entityCard.value = null
		const { name, params } = route
		if (!params?.id) return
		try {
			if (name === 'PerformerDetails') {
				const res = await performersAPI.getById(params.id)
				const p = res?.data?.data ?? res?.data
				if (!p?.name) return
				const count = p.video_count ?? null
				entityContext.value = count != null ? `${p.name} (${count} videos)` : p.name
				const fields = []
				if (count != null) fields.push({ icon: ['fas', 'film'], value: `${count} videos` })
				const age = _ageFromDate(p.metadata?.birthdate)
				if (age) fields.push({ icon: ['fas', 'birthday-cake'], value: `${age} yrs` })
				if (p.metadata?.ethnicity) fields.push({ icon: ['fas', 'globe'], value: p.metadata.ethnicity })
				if (p.metadata?.hair_color) fields.push({ icon: ['fas', 'user'], value: p.metadata.hair_color })
				if (p.metadata?.height) fields.push({ icon: ['fas', 'arrows-alt'], value: p.metadata.height })
				entityCard.value = { type: 'performer', icon: ['fas', 'user-circle'], name: p.name, fields }
			} else if (name === 'VideoPlayer') {
				const res = await videosAPI.getById(params.id)
				const v = res?.data?.data ?? res?.data
				if (!v?.title) return
				entityContext.value = `"${v.title}"`
				const fields = []
				if (v.duration) fields.push({ icon: ['fas', 'clock'], value: _fmtDuration(v.duration) })
				if (v.resolution) fields.push({ icon: ['fas', 'tv'], value: v.resolution })
				if (v.fps) fields.push({ icon: ['fas', 'film'], value: `${Math.round(v.fps)} fps` })
				if (v.rating > 0) fields.push({ icon: ['fas', 'star'], value: '★'.repeat(v.rating) })
				if (v.performers?.length > 0) {
					fields.push({ icon: ['fas', 'users'], value: v.performers.slice(0, 2).map((p) => p.name).join(', ') })
				}
				entityCard.value = { type: 'video', icon: ['fas', 'film'], name: v.title, fields }
			}
		} catch { /* silently ignore — missing context is better than broken context */ }
	}

	/** Build a page context string for the current route */
	const buildPageContext = () => {
		const route = router.currentRoute.value
		const label = PAGE_LABELS[route.name] ?? String(route.name ?? 'App')
		const parts = [label]
		if (entityContext.value) {
			parts.push(entityContext.value)
		} else if (route.params.id) {
			parts.push(`#${route.params.id}`)
		}
		if (route.query.search) parts.push(`search: "${route.query.search}"`)
		return parts.join(' · ')
	}

	/** Generate a proactive greeting message for the entity card */
	const maybeShowEntityGreeting = (card, chatHistoryRef, unreadCountRef, isOpenRef) => {
		const route = router.currentRoute.value
		const key = `${card.type}-${route.params?.id}`
		if (_sessionGreeted.has(key)) return
		_sessionGreeted.add(key)
		sessionStorage.setItem('ai_entity_greeted', JSON.stringify([..._sessionGreeted]))

		let greeting = ''
		if (card.type === 'performer') {
			const vidField = card.fields.find((f) => String(f.value).includes('video'))
			const extras = card.fields.filter((f) => f !== vidField).map((f) => f.value).join(' · ')
			greeting = `I see you're on **${card.name}**'s page${vidField ? ` — ${vidField.value}` : ''}${extras ? ` (${extras})` : ''}. I can summarise their catalog, compare video quality, or find content by tag or rating. What would you like to know?`
		} else if (card.type === 'video') {
			const dur = card.fields.find((f) => f.icon[1] === 'clock')?.value ?? ''
			const res = card.fields.find((f) => f.icon[1] === 'tv')?.value ?? ''
			const cast = card.fields.find((f) => f.icon[1] === 'users')?.value ?? ''
			greeting = `Now playing **"${card.name}"**${dur ? ` · ${dur}` : ''}${res ? ` · ${res}` : ''}${cast ? ` · ${cast}` : ''}. I can suggest similar videos, manage tags, or compare quality with another file. Ask me anything!`
		}
		if (!greeting) return

		chatHistoryRef.value.push({
			role: 'proactive',
			content: greeting,
			timestamp: new Date(),
			severity: 'info',
			source: 'entity_greeting',
		})
		if (!isOpenRef.value) unreadCountRef.value++
	}

	// Watch route changes and entity card resolution
	const setupWatchers = (chatHistoryRef, unreadCountRef, isOpenRef) => {
		watch(() => router.currentRoute.value, (route) => loadEntityContext(route), { immediate: true })
		watch(entityCard, (card) => {
			if (card) maybeShowEntityGreeting(card, chatHistoryRef, unreadCountRef, isOpenRef)
		})
	}

	return {
		entityContext,
		entityCard,
		buildPageContext,
		setupWatchers,
		loadEntityContext,
	}
}
