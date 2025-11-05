import { createRouter, createWebHistory } from 'vue-router'

const routes = [
	{
		path: '/',
		name: 'Home',
		component: () => import('@/views/HomePage.vue'),
	},
	{
		path: '/performers',
		name: 'Performers',
		component: () => import('@/views/PerformersPage.vue'),
	},
	{
		path: '/performers/:id',
		name: 'PerformerDetails',
		component: () => import('@/views/PerformerDetailsPage.vue'),
	},
	{
		path: '/videos',
		name: 'Videos',
		component: () => import('@/views/VideosPage.vue'),
	},
	{
		path: '/libraries',
		name: 'Libraries',
		component: () => import('@/views/LibrariesPage.vue'),
	},
	{
		path: '/browser',
		name: 'Browser',
		component: () => import('@/views/BrowserPage.vue'),
	},
	{
		path: '/studios',
		name: 'Studios',
		component: () => import('@/views/StudiosPage.vue'),
	},
	{
		path: '/tags',
		name: 'Tags',
		component: () => import('@/views/TagsPage.vue'),
	},
	{
		path: '/activity',
		name: 'Activity',
		component: () => import('@/views/ActivityPage.vue'),
	},
	{
		path: '/settings',
		name: 'Settings',
		component: () => import('@/views/SettingsPage.vue'),
	},
]

const router = createRouter({
	history: createWebHistory(process.env.BASE_URL),
	routes,
})

export default router
