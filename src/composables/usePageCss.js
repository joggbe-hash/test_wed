import { onMounted, watch } from 'vue'

const PAGE_STYLE_ID = 'vue-page-style'
const MATERIAL_STYLE_ID = 'material-symbols-style'
const materialHref =
  'https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@20..48,100..700,0..1,-50..200&icon_names=explore'
const preloadedCss = new Set()

export function preloadPageCss(fileNames) {
  fileNames.forEach((fileName) => {
    const href = `/css/${fileName}`
    if (preloadedCss.has(href)) return

    const link = document.createElement('link')
    link.rel = 'preload'
    link.as = 'style'
    link.href = href
    document.head.appendChild(link)
    preloadedCss.add(href)
  })
}

export function usePageCss(fileName, options = {}) {
  const { materialIcons = false } = options

  const apply = () => {
    preloadPageCss([fileName])

    let link = document.getElementById(PAGE_STYLE_ID)
    if (!link) {
      link = document.createElement('link')
      link.id = PAGE_STYLE_ID
      link.rel = 'stylesheet'
      document.head.appendChild(link)
    }
    link.href = `/css/${fileName}`

    if (materialIcons && !document.getElementById(MATERIAL_STYLE_ID)) {
      const materialLink = document.createElement('link')
      materialLink.id = MATERIAL_STYLE_ID
      materialLink.rel = 'stylesheet'
      materialLink.href = materialHref
      document.head.appendChild(materialLink)
    }
  }

  onMounted(apply)
  watch(() => fileName, apply)
}
