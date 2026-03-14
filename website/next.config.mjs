import nextra from 'nextra'

const isDev = process.env.NODE_ENV !== 'production'
const isCloudflarePages = process.env.CF_PAGES === '1' || process.env.CLOUDFLARE_PAGES === '1'

const cspDirectives = [
  "default-src 'self'",
  "base-uri 'self'",
  "font-src 'self' https: data:",
  "form-action 'self'",
  "frame-ancestors 'none'",
  "img-src 'self' data: blob: https:",
  "object-src 'none'",
  `script-src 'self' 'unsafe-inline'${isDev ? " 'unsafe-eval'" : ''} https:`,
  "style-src 'self' 'unsafe-inline' https:",
  "connect-src 'self' https:",
  "worker-src 'self' blob:"
]

if (!isDev) {
  cspDirectives.push('upgrade-insecure-requests')
}

const securityHeaders = [
  { key: 'Content-Security-Policy', value: cspDirectives.join('; ') },
  { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
  { key: 'X-Content-Type-Options', value: 'nosniff' },
  { key: 'X-Frame-Options', value: 'DENY' },
  { key: 'Permissions-Policy', value: 'camera=(), geolocation=(), microphone=(), payment=(), usb=()' },
  { key: 'Cross-Origin-Opener-Policy', value: 'same-origin' },
  { key: 'Cross-Origin-Resource-Policy', value: 'same-origin' },
  { key: 'Origin-Agent-Cluster', value: '?1' },
  { key: 'X-DNS-Prefetch-Control', value: 'off' },
  { key: 'X-Permitted-Cross-Domain-Policies', value: 'none' }
]

if (!isDev) {
  securityHeaders.push({
    key: 'Strict-Transport-Security',
    value: 'max-age=63072000; includeSubDomains; preload'
  })
}

const withNextra = nextra({
  theme: 'nextra-theme-docs',
  themeConfig: './theme.config.tsx',
  flexsearch: {
    codeblocks: false
  }
})

const nextConfig = {
  reactStrictMode: true,
  output: isCloudflarePages ? 'export' : 'standalone',
  trailingSlash: isCloudflarePages,
  images: isCloudflarePages
    ? {
        unoptimized: true
      }
    : undefined
}

if (!isCloudflarePages) {
  nextConfig.headers = async () => [
    {
      source: '/:path*',
      headers: securityHeaders
    }
  ]
}

export default withNextra(nextConfig)
