import fs from 'node:fs/promises'
import path from 'node:path'

const robotsPath = path.join(process.cwd(), 'public', 'robots.txt')
const sitemapPath = path.join(process.cwd(), 'public', 'sitemap.xml')

const robots = await fs.readFile(robotsPath, 'utf8')
const sitemap = await fs.readFile(sitemapPath, 'utf8')

if (!robots.includes('Sitemap:')) {
  console.error('robots.txt must include Sitemap declaration')
  process.exit(1)
}

for (const required of ['<loc>https://biometrics.dev/</loc>', '<loc>https://biometrics.dev/quickstart</loc>', '<loc>https://biometrics.dev/docs</loc>']) {
  if (!sitemap.includes(required)) {
    console.error(`sitemap.xml missing required route entry: ${required}`)
    process.exit(1)
  }
}

console.log('sitemap/robots check passed')
