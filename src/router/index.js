import { createRouter, createWebHistory } from 'vue-router'

const routes = [
	{
		path: '/',
		name: 'Home',
		component: () => import(/* webpackChunkName: "home" */ '@/views/HomePage.vue'),
	},
	{
		path: '/performers',
		name: 'Performers',
		component: () => import(/* webpackChunkName: "performers" */ '@/views/PerformersPage.vue'),
	},
	{
		path: '/performers/:id',
		name: 'PerformerDetails',
		component: () => import(/* webpackChunkName: "performer-details" */ '@/views/PerformerDetailsPage.vue'),
	},
	{
		path: '/videos',
		name: 'Videos',
		component: () => import(/* webpackChunkName: "videos" */ '@/views/VideosPage.vue'),
	},
	{
		path: '/libraries',
		name: 'Libraries',
		component: () => import(/* webpackChunkName: "libraries" */ '@/views/LibrariesPage.vue'),
	},
	{
		path: '/browser',
		name: 'Browser',
		component: () => import(/* webpackChunkName: "browser" */ '@/views/BrowserPage.vue'),
	},
	{
		path: '/studios',
		name: 'Studios',
		component: () => import(/* webpackChunkName: "studios" */ '@/views/StudiosPage.vue'),
	},
	{
		path: '/tags',
		name: 'Tags',
		component: () => import(/* webpackChunkName: "tags" */ '@/views/TagsPage.vue'),
	},
	{
		path: '/activity',
		name: 'Activity',
		component: () => import(/* webpackChunkName: "activity" */ '@/views/ActivityPage.vue'),
	},
	{
		path: '/ai',
		name: 'AI',
		component: () => import(/* webpackChunkName: "ai" */ '@/views/AIPage.vue'),
	},
	{
		path: '/settings',
		name: 'Settings',
		component: () => import(/* webpackChunkName: "settings" */ '@/views/SettingsPage.vue'),
	},
	{
		path: '/tasks',
		name: 'Tasks',
		component: () => import(/* webpackChunkName: "tasks" */ '@/views/TasksPage.vue'),
	},
]

const router = createRouter({
	history: createWebHistory(process.env.BASE_URL),
	routes,
	scrollBehavior(to, from, savedPosition) {
		if (savedPosition) {
			return savedPosition
		}
		return { top: 0 }
	},
})

export default router
