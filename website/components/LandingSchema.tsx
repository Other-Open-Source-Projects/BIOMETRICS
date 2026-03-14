const schema = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'BIOMETRICS',
  applicationCategory: 'DeveloperApplication',
  operatingSystem: 'macOS, Linux, Windows (WSL)',
  offers: {
    '@type': 'Offer',
    price: '0',
    priceCurrency: 'USD'
  },
  description: 'BIOMETRICS control-plane for orchestrated coding runs with production-grade docs and runbooks.'
}

export function LandingSchema() {
  return <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(schema) }} />
}
