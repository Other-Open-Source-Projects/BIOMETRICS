import { expect, test } from '@playwright/test'

test('landing to quickstart flow works', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByTestId('landing-hero')).toBeVisible()
  await page.getByTestId('cta-quickstart').click()
  await expect(page).toHaveURL(/\/quickstart$/)
  await expect(page.getByTestId('command-tabs')).toBeVisible()
  await page.getByTestId('tab-linux').click()
  await expect(page.getByTestId('command-block')).toContainText('./biometrics-onboard')
})

test('install hub links to OS-specific guides', async ({ page }) => {
  await page.goto('/install')
  await expect(page.getByRole('link', { name: 'macOS install path' })).toBeVisible()
  await page.getByRole('link', { name: 'Windows (WSL) install path' }).click()
  await expect(page).toHaveURL(/\/install\/windows-wsl$/)
})

test('docs fallback search returns results', async ({ page }) => {
  await page.goto('/docs')
  await page.getByTestId('docs-search-input').fill('runbook')
  await page.getByTestId('docs-search-button').click()
  await expect(page.getByRole('link', { name: 'Runbook: Production Operations' })).toBeVisible()
})

test('locale switch to german critical pages', async ({ page }) => {
  await page.goto('/')
  await page.getByTestId('switch-de').click()
  await expect(page).toHaveURL(/\/de$/)
  await page.getByTestId('cta-quickstart-de').click()
  await expect(page).toHaveURL(/\/de\/quickstart$/)
})
