import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // Enable Turbopack for faster dev builds
  experimental: {
    turbo: {
      rules: {
        '*.tsx': {
          loaders: ['swc-loader'],
          as: '*.js',
        },
      },
    },
    optimizeCss: true,
    optimizePackageImports: ['@/components', '@/lib'],
  },

  // Disable type checking in dev for speed
  typescript: {
    ignoreBuildErrors: true,
  },
  
  eslint: {
    ignoreDuringBuilds: true,
  },

  // Optimize for dev performance
  swcMinify: true,
  compress: false, // Disable compression in dev
  poweredByHeader: false,

  // Fast webpack config for dev
  webpack: (config, { dev }) => {
    if (dev) {
      // Speed up dev builds
      config.optimization = {
        ...config.optimization,
        removeAvailableModules: false,
        removeEmptyChunks: false,
        splitChunks: false,
      };
      
      // Reduce file watching overhead
      config.watchOptions = {
        poll: false,
        ignored: /node_modules/,
      };
    }
    return config;
  },

  // Headers for performance
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=0, must-revalidate',
          },
        ],
      },
    ];
  },
};

export default nextConfig;