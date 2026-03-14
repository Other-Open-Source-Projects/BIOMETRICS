import fs from 'node:fs/promises'
import path from 'node:path'
import matter from 'gray-matter'

const pagesDir = path.join(process.cwd(), 'pages')

async function listMDX(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...(await listMDX(fullPath)))
      continue
    }
    if (entry.name.endsWith('.mdx')) {
      files.push(fullPath)
    }
  }
  return files
}

const files = await listMDX(pagesDir)
const failures = []
for (const file of files) {
  const source = await fs.readFile(file, 'utf8')
  const parsed = matter(source)
  if (!parsed.data.title || !parsed.data.description) {
    failures.push(path.relative(process.cwd(), file))
  }
}

if (failures.length > 0) {
  console.error('Missing required frontmatter (title/description):')
  for (const failure of failures) {
    console.error(`- ${failure}`)
  }
  process.exit(1)
}

console.log(`frontmatter check passed (${files.length} mdx files)`)
