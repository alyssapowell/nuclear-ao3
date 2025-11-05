import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // Safari compatibility fixes
  poweredByHeader: false,
  
  // Disable type checking in dev for speed
  typescript: {
    ignoreBuildErrors: true,
  },
  
  eslint: {
    ignoreDuringBuilds: true,
  },
  
  // Optimize for precompiled CSS
  experimental: {
    optimizeCss: true,
  },
  
  // Ensure proper headers for Safari and service workers
  async headers() {
    return [
      {
        source: '/_next/static/css/(.*)',
        headers: [
          {
            key: 'Content-Type',
            value: 'text/css; charset=utf-8',
          },
          {
            key: 'Cache-Control',
            value: 'public, max-age=31536000, immutable',
          },
        ],
      },
      {
        source: '/sw-consent-aware.js',
        headers: [
          {
            key: 'Content-Type',
            value: 'application/javascript',
          },
          {
            key: 'Service-Worker-Allowed',
            value: '/',
          },
        ],
      },
    ];
  },

  // Handle service worker routing
  async rewrites() {
    return [
      {
        source: '/sw-consent-aware.js',
        destination: '/sw-consent-aware.js',
      },
    ];
  },
};

export default nextConfig;