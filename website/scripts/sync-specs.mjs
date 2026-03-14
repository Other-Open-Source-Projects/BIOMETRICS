import fs from 'node:fs/promises'
import path from 'node:path'

const root = process.cwd()
const repoRoot = path.resolve(root, '..')
const openapiSource = path.join(repoRoot, 'docs', 'api', 'openapi-v3-controlplane.yaml')
const contractsSource = path.join(repoRoot, 'docs', 'specs', 'contracts')

const publicSpecs = path.join(root, 'public', 'specs')
const publicContracts = path.join(publicSpecs, 'contracts')

await fs.mkdir(publicContracts, { recursive: true })
await fs.copyFile(openapiSource, path.join(publicSpecs, 'openapi-v3-controlplane.yaml'))

const contracts = await fs.readdir(contractsSource)
for (const file of contracts) {
  if (!file.endsWith('.json')) {
    continue
  }
  await fs.copyFile(path.join(contractsSource, file), path.join(publicContracts, file))
}

console.log(`synced openapi + ${contracts.filter((name) => name.endsWith('.json')).length} contracts`)
