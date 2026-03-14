import type { DocsThemeConfig } from 'nextra-theme-docs'
import { useRouter } from 'next/router'

const config: DocsThemeConfig = {
  logo: (
    <span className="brand-logo">
      BIOMETRICS <em>control plane</em>
    </span>
  ),
  project: {
    link: 'https://github.com/biometrics/biometrics'
  },
  docsRepositoryBase: 'https://github.com/biometrics/biometrics/tree/main/website/pages',
  search: {
    placeholder: 'Search BIOMETRICS docs'
  },
  footer: {
    text: `BIOMETRICS Public Website ${new Date().getFullYear()}`
  },
  useNextSeoProps() {
    const { asPath } = useRouter()
    if (asPath === '/') {
      return {
        titleTemplate: 'BIOMETRICS'
      }
    }
    return {
      titleTemplate: '%s – BIOMETRICS'
    }
  },
  head: () => (
    <>
      <meta name="description" content="Install, operate, and harden BIOMETRICS with production-ready guidance." />
      <meta name="og:title" content="BIOMETRICS" />
      <meta name="og:description" content="Public product and docs site for BIOMETRICS." />
      <meta name="og:type" content="website" />
      <meta name="twitter:card" content="summary_large_image" />
    </>
  )
}

export default config
