import fs from 'node:fs/promises'
import path from 'node:path'

const root = process.cwd()
const pagesDir = path.join(root, 'pages')
const publicDir = path.join(root, 'public')

async function listFiles(dir, predicate) {
  const entries = await fs.readdir(dir, { withFileTypes: true })
  const output = []
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      output.push(...(await listFiles(fullPath, predicate)))
      continue
    }
    if (predicate(entry.name)) {
      output.push(fullPath)
    }
  }
  return output
}

function routeCandidates(routePath) {
  const cleanPath = routePath.replace(/[#?].*$/, '')
  if (cleanPath === '/' || cleanPath === '') {
    return [path.join(pagesDir, 'index.mdx')]
  }
  const normalized = cleanPath.replace(/^\//, '')
  return [
    path.join(pagesDir, `${normalized}.mdx`),
    path.join(pagesDir, normalized, 'index.mdx'),
    path.join(publicDir, normalized),
    path.join(publicDir, `${normalized}.xml`),
    path.join(publicDir, `${normalized}.txt`),
    path.join(publicDir, `${normalized}.json`),
    path.join(publicDir, `${normalized}.yaml`)
  ]
}

const mdxFiles = await listFiles(pagesDir, (name) => name.endsWith('.mdx'))
const failures = []

for (const file of mdxFiles) {
  const source = await fs.readFile(file, 'utf8')
  const matches = [
    ...source.matchAll(/\[[^\]]+\]\(([^)]+)\)/g),
    ...source.matchAll(/href="([^"]+)"/g)
  ]

  for (const match of matches) {
    const link = match[1]
    if (!link || /^(https?:|mailto:|#)/.test(link)) {
      continue
    }

    const candidates = link.startsWith('/')
      ? routeCandidates(link)
      : routeCandidates(path.posix.join(path.posix.dirname(path.relative(pagesDir, file).replace(/\.mdx$/, '')), link))

    let exists = false
    for (const candidate of candidates) {
      try {
        await fs.access(candidate)
        exists = true
        break
      } catch {
        // continue
      }
    }

    if (!exists) {
      failures.push(`${path.relative(root, file)} -> ${link}`)
    }
  }
}

if (failures.length > 0) {
  console.error('Broken internal links detected:')
  for (const failure of failures) {
    console.error(`- ${failure}`)
  }
  process.exit(1)
}

console.log(`link check passed (${mdxFiles.length} mdx files)`)
