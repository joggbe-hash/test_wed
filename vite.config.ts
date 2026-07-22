import { defineConfig, loadEnv, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

const metadataBlockPattern = /\s*<!-- site-metadata:start -->[\s\S]*?<!-- site-metadata:end -->/

function normalizeBasePath(value: string | undefined) {
  const trimmed = value?.trim()
  if (!trimmed || trimmed === '/') return '/'
  return `/${trimmed.replace(/^\/+|\/+$/g, '')}/`
}

function siteMetadataPlugin(siteUrl: string | undefined): Plugin {
  return {
    name: 'type-wsp-site-metadata',
    transformIndexHtml(html) {
      const trimmedSiteUrl = siteUrl?.trim()
      if (!trimmedSiteUrl) {
        return html.replace(metadataBlockPattern, '')
      }

      const canonicalUrl = new URL(trimmedSiteUrl)
      if (!canonicalUrl.pathname.endsWith('/')) {
        canonicalUrl.pathname += '/'
      }
      const socialImageUrl = new URL('picture/meme_background.jpg', canonicalUrl)

      return html
        .replaceAll('__TYPE_WSP_SITE_URL__', canonicalUrl.toString())
        .replaceAll('__TYPE_WSP_SOCIAL_IMAGE_URL__', socialImageUrl.toString())
    },
  }
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '.', 'VITE_')

  return {
    plugins: [
      vue(),
      tailwindcss(),
      siteMetadataPlugin(env.VITE_SITE_URL),
    ],
    base: normalizeBasePath(env.VITE_BASE_PATH),
  }
})
