import { Sidebar } from '@/components/layout/sidebar';
import { Header } from '@/components/layout/header';
import { MobileNav } from '@/components/layout/mobile-nav';
import { AuthGuard } from '@/components/auth/auth-guard';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGuard>
      {/* .app-shell = 100dvh（带 100vh 兜底）：地址栏收放时不裁掉底部一截 */}
      <div className="app-shell flex overflow-hidden bg-bg-base">
        <Sidebar />
        {/* min-w-0：长种子名/hash 不能把这一列顶宽，否则整页横向溢出 */}
        <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
          <Header />
          <main className="main-scroll">{children}</main>
          {/* 底部标签栏是 flex 子项而非 fixed，内容区无需补偿内边距 */}
          <MobileNav />
        </div>
      </div>
    </AuthGuard>
  );
}
