/** @type {import('next').NextConfig} */
const nextConfig = {
  // Use the Webpack scripts for consistent builds in restricted dev environments.
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.MEGHATRON_API_URL || 'http://127.0.0.1:8080'}/api/:path*`,
      },
    ];
  },
};

module.exports = nextConfig;
