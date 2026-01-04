/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,

  // Enable standalone output for Docker
  output: 'standalone',

  // Configure image domains
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'ratings.fide.com',
      },
    ],
  },

  // Internationalization
  i18n: {
    locales: ['de', 'en'],
    defaultLocale: 'de',
  },

  // Environment variables exposed to browser
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
    NEXT_PUBLIC_WS_URL: process.env.NEXT_PUBLIC_WS_URL,
  },

  // Rewrites for API proxy in development
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.API_URL || 'http://localhost:8080'}/api/:path*`,
      },
      {
        source: '/ws/:path*',
        destination: `${process.env.WS_URL || 'http://localhost:8081'}/ws/:path*`,
      },
    ];
  },
};

module.exports = nextConfig;
