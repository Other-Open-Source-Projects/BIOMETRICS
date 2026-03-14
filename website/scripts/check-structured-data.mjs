import fs from 'node:fs/promises'
import path from 'node:path'

const landing = await fs.readFile(path.join(process.cwd(), 'pages', 'index.mdx'), 'utf8')
if (!landing.includes('<LandingSchema />')) {
  console.error('Landing page must include <LandingSchema /> for JSON-LD structured data')
  process.exit(1)
}
console.log('structured data check passed')
