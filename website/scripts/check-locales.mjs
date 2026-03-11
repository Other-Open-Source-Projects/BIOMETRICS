import fs from 'node:fs/promises'
import path from 'node:path'

const pagesDir = path.join(process.cwd(), 'pages')

const requiredEN = ['index.mdx', 'quickstart.mdx', 'install/index.mdx', 'install/macos.mdx', 'install/linux.mdx', 'install/windows-wsl.mdx']
const requiredDE = ['de/index.mdx', 'de/quickstart.mdx', 'de/install.mdx']

async function ensureFiles(paths) {
  for (const relativePath of paths) {
    const target = path.join(pagesDir, relativePath)
    try {
      await fs.access(target)
    } catch {
      throw new Error(`Missing locale-critical page: ${relativePath}`)
    }
  }
}

await ensureFiles(requiredEN)
await ensureFiles(requiredDE)
console.log('locale route generation check passed')
