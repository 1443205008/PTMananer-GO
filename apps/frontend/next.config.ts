import path from 'node:path';
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  output: 'export',
  // The app lives in a pnpm monorepo, so include only traced files while
  // keeping the standalone directory layout rooted at the repository root.
  outputFileTracingRoot: path.join(__dirname, '../..'),
  // 静态导出：API 由 Go 后端同源服务（NEXT_PUBLIC_API_BASE_URL=/api）
  // 引用 shared package 的源码（不需要单独编译）
  transpilePackages: ['@pt-manager/shared'],
  images: {
    unoptimized: true,
    // M-Team CDN 域（⚠️ 实际 CDN 域名需验证后更新）
    remotePatterns: [
      { protocol: 'https', hostname: '**.m-team.cc' },
      { protocol: 'https', hostname: '**.m-team.io' },
    ],
  },
};

export default nextConfig;
