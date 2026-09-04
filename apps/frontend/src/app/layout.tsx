import type { Metadata, Viewport } from 'next';
import { Geist, Geist_Mono } from 'next/font/google';
import '../styles/globals.css';
import { QueryProvider } from '@/components/providers/query-provider';
import { ThemeProvider } from '@/components/providers/theme-provider';

const geistSans = Geist({ subsets: ['latin'], variable: '--font-geist-sans' });
const geistMono = Geist_Mono({ subsets: ['latin'], variable: '--font-geist-mono' });

export const metadata: Metadata = {
  title: 'PT Manager',
  description: '私人 PT 账号聚合管理平台',
  // 加到主屏幕时按独立 App 打开，顶栏跟随主题色
  appleWebApp: { capable: true, statusBarStyle: 'black-translucent', title: 'PT Manager' },
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  // 不锁死缩放：无障碍上放大是必要能力
  maximumScale: 5,
  userScalable: true,
  // 让 env(safe-area-inset-*) 生效，配合 .safe-b / .main-scroll 避让刘海和手势条
  viewportFit: 'cover',
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#FFFFFF' },
    { media: '(prefers-color-scheme: dark)', color: '#09090B' },
  ],
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <body className={`${geistSans.variable} ${geistMono.variable} antialiased`}>
        <ThemeProvider
          attribute="class"
          defaultTheme="dark"
          enableSystem={false}
          disableTransitionOnChange
        >
          <QueryProvider>{children}</QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
