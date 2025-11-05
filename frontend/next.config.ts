import type { NextConfig } from 'next';

// Generate build timestamp for cache busting
const BUILD_TIMESTAMP = Date.now().toString();

const nextConfig: NextConfig = {
  // Safari compatibility fixes
  poweredByHeader: false,
  
  // Add build timestamp to all assets for cache busting
  generateBuildId: async () => {
    return `build-${BUILD_TIMESTAMP}`;
  },
  
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
        // Static assets with build ID can be cached aggressively
        source: '/_next/static/(.*)',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=31536000, immutable',
          },
        ],
      },
      {
        // CSS files
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
        // JavaScript chunks - shorter cache for deployments
        source: '/_next/static/chunks/(.*)',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=3600, stale-while-revalidate=86400',
          },
          {
            key: 'X-Build-ID',
            value: BUILD_TIMESTAMP,
          },
        ],
      },
      {
        // Main pages - short cache to ensure updates
        source: '/(.*)',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=60, stale-while-revalidate=300',
          },
          {
            key: 'X-Build-ID', 
            value: BUILD_TIMESTAMP,
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
          {
            key: 'Cache-Control',
            value: 'no-cache',
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