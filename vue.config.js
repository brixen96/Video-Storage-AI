const { defineConfig } = require('@vue/cli-service')
const { purgeCSSPlugin } = require('@fullhuman/postcss-purgecss')

module.exports = defineConfig({
	transpileDependencies: true,
	devServer: {
		port: 8081,
		proxy: {
			'/api': {
				target: 'http://localhost:8080',
				changeOrigin: true,
			},
		},
	},
	css: {
		loaderOptions: {
			postcss: {
				postcssOptions: {
					plugins: process.env.NODE_ENV === 'production' ? [
						purgeCSSPlugin({
							content: [
								'./src/**/*.vue',
								'./src/**/*.js',
								'./src/**/*.html',
								'./public/**/*.html',
							],
							safelist: {
								// Keep Bootstrap dynamic classes
								standard: [
									/^btn-/,
									/^bg-/,
									/^text-/,
									/^badge-/,
									/^alert-/,
									/^border-/,
									/^rounded-/,
									/^shadow-/,
									/^d-/,
									/^flex-/,
									/^justify-/,
									/^align-/,
									/^col-/,
									/^row-/,
									/^g-/,
									/^gap-/,
									/^m[tblrxy]?-/,
									/^p[tblrxy]?-/,
									/^w-/,
									/^h-/,
									/^position-/,
									/^top-/,
									/^bottom-/,
									/^start-/,
									/^end-/,
									/^modal/,
									/^fade/,
									/^show/,
									/^collapse/,
									/^dropdown/,
									/^nav/,
									/^active/,
								],
								// Keep FontAwesome classes
								deep: [/^fa-/, /^svg-inline/],
								// Keep animation classes
								greedy: [/^slide-/, /^fade-/, /^scale-/],
							},
						}),
					] : [],
				},
			},
		},
	},
	// Performance optimizations for production
	productionSourceMap: false,
	configureWebpack: {
		optimization: {
			splitChunks: {
				chunks: 'all',
				cacheGroups: {
					vendor: {
						test: /[\/]node_modules[\/]/,
						name: 'chunk-vendors',
						priority: 10,
					},
					common: {
						minChunks: 2,
						priority: 5,
						reuseExistingChunk: true,
					},
				},
			},
		},
	},
})
