/** @type {import('next').NextConfig} */
const nextConfig = {
  // Removed standalone output to fix static file serving
  trailingSlash: false,
  async headers() {
    return [
      {
        source: '/sw-consent-aware.js',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=86400',
          },
          {
            key: 'Content-Type',
            value: 'application/javascript',
          },
        ],
      },
      {
        source: '/manifest.json',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=86400',
          },
          {
            key: 'Content-Type',
            value: 'application/manifest+json',
          },
        ],
      },
    ];
  },
}

module.exports = nextConfig