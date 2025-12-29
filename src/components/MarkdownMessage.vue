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
@import 'highlight.js/styles/atom-one-dark.css';

.markdown-message {
	color: #fff;
	line-height: 1.6;
	font-size: 0.9rem;
}

/* Typography */
.markdown-message :deep(h1),
.markdown-message :deep(h2),
.markdown-message :deep(h3),
.markdown-message :deep(h4),
.markdown-message :deep(h5),
.markdown-message :deep(h6) {
	margin-top: 1rem;
	margin-bottom: 0.5rem;
	font-weight: 600;
	color: #00d9ff;
}

.markdown-message :deep(h1) {
	font-size: 1.5rem;
	border-bottom: 2px solid rgba(0, 217, 255, 0.3);
	padding-bottom: 0.3rem;
}

.markdown-message :deep(h2) {
	font-size: 1.3rem;
}

.markdown-message :deep(h3) {
	font-size: 1.1rem;
}

.markdown-message :deep(p) {
	margin-bottom: 0.75rem;
	color: rgba(255, 255, 255, 0.9);
}

.markdown-message :deep(p:last-child) {
	margin-bottom: 0;
}

/* Lists */
.markdown-message :deep(ul),
.markdown-message :deep(ol) {
	margin: 0.5rem 0;
	padding-left: 1.5rem;
}

.markdown-message :deep(li) {
	margin: 0.25rem 0;
	color: rgba(255, 255, 255, 0.9);
}

.markdown-message :deep(li::marker) {
	color: #00d9ff;
}

/* Links */
.markdown-message :deep(a) {
	color: #00d9ff;
	text-decoration: none;
	border-bottom: 1px solid rgba(0, 217, 255, 0.3);
	transition: all 0.2s ease;
}

.markdown-message :deep(a:hover) {
	color: #00ffff;
	border-bottom-color: #00ffff;
}

/* Inline Code */
.markdown-message :deep(code:not(pre code)) {
	background: rgba(0, 217, 255, 0.15);
	color: #00ffff;
	padding: 0.15rem 0.4rem;
	border-radius: 4px;
	font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
	font-size: 0.85em;
	border: 1px solid rgba(0, 217, 255, 0.3);
}

/* Code Blocks */
.markdown-message :deep(.code-block-wrapper) {
	position: relative;
	margin: 0.75rem 0;
	border-radius: 8px;
	overflow: hidden;
	background: #282c34;
	border: 1px solid rgba(0, 217, 255, 0.2);
}

.markdown-message :deep(.code-block-header) {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 0.5rem 0.75rem;
	background: rgba(0, 0, 0, 0.3);
	border-bottom: 1px solid rgba(0, 217, 255, 0.2);
}

.markdown-message :deep(.code-lang) {
	font-size: 0.75rem;
	font-weight: 600;
	text-transform: uppercase;
	color: #00d9ff;
	letter-spacing: 0.5px;
}

.markdown-message :deep(.code-copy-btn) {
	display: flex;
	align-items: center;
	gap: 0.3rem;
	background: rgba(0, 217, 255, 0.1);
	border: 1px solid rgba(0, 217, 255, 0.3);
	color: #00d9ff;
	padding: 0.3rem 0.6rem;
	border-radius: 4px;
	font-size: 0.75rem;
	cursor: pointer;
	transition: all 0.2s ease;
	font-weight: 500;
}

.markdown-message :deep(.code-copy-btn:hover) {
	background: rgba(0, 217, 255, 0.2);
	border-color: #00d9ff;
	transform: translateY(-1px);
}

.markdown-message :deep(.code-copy-btn.copied) {
	background: rgba(0, 255, 127, 0.2);
	border-color: #00ff7f;
	color: #00ff7f;
}

.markdown-message :deep(pre) {
	margin: 0;
	padding: 1rem;
	overflow-x: auto;
	background: #282c34;
	font-size: 0.85rem;
}

.markdown-message :deep(pre code) {
	background: transparent;
	color: #abb2bf;
	padding: 0;
	border: none;
	font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
	line-height: 1.5;
}

/* Blockquotes */
.markdown-message :deep(blockquote) {
	margin: 0.75rem 0;
	padding: 0.5rem 0.75rem;
	border-left: 4px solid #00d9ff;
	background: rgba(0, 217, 255, 0.1);
	border-radius: 0 6px 6px 0;
	color: rgba(255, 255, 255, 0.85);
	font-style: italic;
}

.markdown-message :deep(blockquote p) {
	margin: 0;
}

/* Tables */
.markdown-message :deep(table) {
	width: 100%;
	border-collapse: collapse;
	margin: 0.75rem 0;
	background: rgba(0, 0, 0, 0.2);
	border-radius: 6px;
	overflow: hidden;
}

.markdown-message :deep(th),
.markdown-message :deep(td) {
	padding: 0.5rem 0.75rem;
	border: 1px solid rgba(0, 217, 255, 0.2);
	text-align: left;
}

.markdown-message :deep(th) {
	background: rgba(0, 217, 255, 0.2);
	color: #00d9ff;
	font-weight: 600;
}

.markdown-message :deep(tr:hover) {
	background: rgba(0, 217, 255, 0.05);
}

/* Horizontal Rule */
.markdown-message :deep(hr) {
	border: none;
	border-top: 2px solid rgba(0, 217, 255, 0.3);
	margin: 1rem 0;
}

/* Strong/Bold */
.markdown-message :deep(strong) {
	font-weight: 600;
	color: #fff;
}

/* Emphasis/Italic */
.markdown-message :deep(em) {
	font-style: italic;
	color: rgba(255, 255, 255, 0.95);
}

/* Scrollbar for code blocks */
.markdown-message :deep(pre::-webkit-scrollbar) {
	height: 8px;
}

.markdown-message :deep(pre::-webkit-scrollbar-track) {
	background: rgba(0, 0, 0, 0.3);
	border-radius: 4px;
}

.markdown-message :deep(pre::-webkit-scrollbar-thumb) {
	background: rgba(0, 217, 255, 0.3);
	border-radius: 4px;
}

.markdown-message :deep(pre::-webkit-scrollbar-thumb:hover) {
	background: rgba(0, 217, 255, 0.5);
}
</style>
