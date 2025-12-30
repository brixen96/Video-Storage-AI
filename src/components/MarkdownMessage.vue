<template>
	<div class="markdown-message" v-html="renderedMarkdown"></div>
</template>

<script setup>
/* eslint-disable no-undef */
import { computed, onMounted, nextTick } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import hljs from 'highlight.js/lib/core'

// Import common languages for syntax highlighting
import javascript from 'highlight.js/lib/languages/javascript'
import python from 'highlight.js/lib/languages/python'
import json from 'highlight.js/lib/languages/json'
import sql from 'highlight.js/lib/languages/sql'
import bash from 'highlight.js/lib/languages/bash'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import go from 'highlight.js/lib/languages/go'

// Register languages
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('python', python)
hljs.registerLanguage('json', json)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('css', css)
hljs.registerLanguage('go', go)

const props = defineProps({
	content: {
		type: String,
		required: true,
	},
})

// Configure marked with syntax highlighting
marked.setOptions({
	highlight: function (code, lang) {
		if (lang && hljs.getLanguage(lang)) {
			try {
				return hljs.highlight(code, { language: lang }).value
			} catch (err) {
				console.error('Highlighting error:', err)
			}
		}
		return hljs.highlightAuto(code).value
	},
	breaks: true,
	gfm: true,
})

// Create custom renderer to add copy button to code blocks
const renderer = new marked.Renderer()
const originalCodeRenderer = renderer.code.bind(renderer)

renderer.code = function (code, language) {
	const lang = language || 'plaintext'
	const highlighted = originalCodeRenderer(code, language)

	// Wrap code block with a container that has a copy button
	return `
		<div class="code-block-wrapper">
			<div class="code-block-header">
				<span class="code-lang">${lang}</span>
				<button class="code-copy-btn" data-code="${escapeHtml(code)}">
					<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
						<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
					</svg>
					Copy
				</button>
			</div>
			${highlighted}
		</div>
	`
}

// Helper to escape HTML for data attribute
function escapeHtml(text) {
	const div = document.createElement('div')
	div.textContent = text
	return div.innerHTML
}

marked.use({ renderer })

const renderedMarkdown = computed(() => {
	try {
		const html = marked.parse(props.content)
		// Sanitize HTML to prevent XSS attacks
		return DOMPurify.sanitize(html, {
			ADD_ATTR: ['target', 'data-code'],
			ADD_TAGS: ['button'],
		})
	} catch (error) {
		console.error('Markdown parsing error:', error)
		return `<p>${escapeHtml(props.content)}</p>`
	}
})

// Add copy functionality after component mounts
onMounted(async () => {
	await nextTick()
	setupCopyButtons()
})

const setupCopyButtons = () => {
	const copyButtons = document.querySelectorAll('.code-copy-btn')
	copyButtons.forEach((button) => {
		button.addEventListener('click', async () => {
			const code = button.getAttribute('data-code')
			if (code) {
				try {
					await navigator.clipboard.writeText(code)
					const originalText = button.innerHTML
					button.innerHTML = `
						<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<polyline points="20 6 9 17 4 12"></polyline>
						</svg>
						Copied!
					`
					button.classList.add('copied')

					setTimeout(() => {
						button.innerHTML = originalText
						button.classList.remove('copied')
					}, 2000)
				} catch (err) {
					console.error('Failed to copy code:', err)
				}
			}
		})
	})
}
</script>

<style scoped>
@import '@/styles/components/markdown_message.css';
</style>
