import { Download } from 'lucide-react';
import { ComingSoon } from '@/components/layout/coming-soon';

export default function DownloadersPage() {
  return (
    <ComingSoon
      title="下载器"
      description="接入 qBittorrent / Transmission 等下载器，关联做种数据并支持远程状态查看。"
      icon={Download}
      phase="Phase 9"
    />
  );
}
