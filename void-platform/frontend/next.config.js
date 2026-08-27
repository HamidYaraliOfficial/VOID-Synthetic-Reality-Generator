/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  env: {
    NEXT_PUBLIC_VOID_API_BASE: process.env.NEXT_PUBLIC_VOID_API_BASE || 'http://localhost:8080',
  },
};
module.exports = nextConfig;
