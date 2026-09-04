'use client';

import { useEffect, useState } from 'react';
import {
  AlertTriangle,
  Bell,
  CheckCircle2,
  Clock3,
  Database,
  Info,
  KeyRound,
  Loader2,
  RefreshCw,
  RotateCcw,
  Save,
  Server,
  Settings,
  SlidersHorizontal,
  Trash2,
} from 'lucide-react';
import {
  useCleanupSettingsData,
  useResetSettings,
  useSettings,
  useSettingsStatus,
  useUpdateSettings,
} from '@/hooks/use-settings';
import { useAuth, useUpdateAccount } from '@/hooks/use-auth';
import type { SettingsStatus, SystemSettings, UpdateSystemSettingsInput } from '@/types/api';
import { cn } from '@/lib/utils';

type SettingsTab = 'account' | 'general' | 'sync' | 'alerts' | 'data' | 'about';
type SettingsForm = UpdateSystemSettingsInput;

const DEFAULT_FORM: SettingsForm = {
  systemName: 'PT Manager',
  timezone: 'Asia/Shanghai',
  defaultPageSize: 20,
  syncEnabled: true,
  syncIntervalMinutes: 30,
  alertEnabled: true,
  alertRetentionDays: 90,
  syncLogRetentionDays: 30,
  snapshotRetentionDays: 365,
};

const TABS: Array<{ id: SettingsTab; label: string; icon: typeof Settings }> = [
  { id: 'account', label: '登录账号', icon: KeyRound },
  { id: 'general', label: '通用设置', icon: SlidersHorizontal },
  { id: 'sync', label: '同步设置', icon: RefreshCw },
  { id: 'alerts', label: '告警设置', icon: Bell },
  { id: 'data', label: '数据维护', icon: Database },
  { id: 'about', label: '关于系统', icon: Info },
];

const TIMEZONES = [
  { value: 'Asia/Shanghai', label: '中国标准时间（UTC+8）' },
  { value: 'UTC', label: '协调世界时（UTC）' },
  { value: 'Asia/Tokyo', label: '日本标准时间（UTC+9）' },
  { value: 'America/Los_Angeles', label: '太平洋时间（UTC-8/-7）' },
  { value: 'Europe/London', label: '伦敦时间（UTC+0/+1）' },
];

const PAGE_SIZES = [20, 50, 100];
const SYNC_INTERVALS = [5, 15, 30, 60];
const RETENTION_OPTIONS = [7, 30, 90, 180, 365, 730];
const SNAPSHOT_RETENTION_OPTIONS = [30, 90, 180, 365, 730, 1095];

function toForm(settings: SystemSettings): SettingsForm {
  return {
    systemName: settings.systemName,
    timezone: settings.timezone,
    defaultPageSize: settings.defaultPageSize,
    syncEnabled: settings.syncEnabled,
    syncIntervalMinutes: settings.syncIntervalMinutes,
    alertEnabled: settings.alertEnabled,
    alertRetentionDays: settings.alertRetentionDays,
    syncLogRetentionDays: settings.syncLogRetentionDays,
    snapshotRetentionDays: settings.snapshotRetentionDays,
  };
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '请求失败，请稍后重试';
}

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const [form, setForm] = useState<SettingsForm>(DEFAULT_FORM);
  const [cleanupMessage, setCleanupMessage] = useState<string | null>(null);
  const { data, isLoading, isError, refetch } = useSettings();
  const { data: status } = useSettingsStatus();
  const update = useUpdateSettings();
  const reset = useResetSettings();
  const cleanup = useCleanupSettingsData();

  useEffect(() => {
    if (data) setForm(toForm(data));
  }, [data]);

  const isDirty = data ? JSON.stringify(form) !== JSON.stringify(toForm(data)) : false;
  const isSaving = update.isPending || reset.isPending;

  function updateField<Key extends keyof SettingsForm>(key: Key, value: SettingsForm[Key]) {
    setForm((current) => ({ ...current, [key]: value }));
    setCleanupMessage(null);
  }

  function handleSave() {
    update.mutate(form);
  }

  function handleReset() {
    if (!window.confirm('确定恢复所有系统设置的默认值吗？')) return;
    reset.mutate();
  }

  function handleCleanup() {
    if (!window.confirm('确定按当前保留策略清理历史数据吗？此操作不可撤销。')) return;
    setCleanupMessage(null);
    cleanup.mutate(undefined, {
      onSuccess: (result) => {
        setCleanupMessage(
          `清理完成：同步任务 ${result.syncJobs} 条、统计快照 ${result.snapshots} 条、已忽略告警 ${result.alerts} 条。`,
        );
      },
    });
  }

  if (isLoading) return <SettingsSkeleton />;

  if (isError || !data) {
    return (
      <div className="flex min-h-[420px] flex-col items-center justify-center gap-3 text-center">
        <AlertTriangle className="h-8 w-8 text-danger" />
        <p className="text-sm text-fg-subtle">系统设置加载失败</p>
        <button
          onClick={() => void refetch()}
          className="btn-toolbar"
        >
          重试
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="page-header">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <Settings className="h-5 w-5 shrink-0 text-accent" />
            <h2 className="text-lg font-semibold text-fg">系统设置</h2>
          </div>
          <p className="mt-0.5 text-sm text-fg-subtle">管理同步、告警和历史数据保留策略</p>
        </div>
        {activeTab !== 'about' && activeTab !== 'account' && (
          <div className="flex shrink-0 items-center gap-2">
            <button
              onClick={handleReset}
              disabled={isSaving}
              className="btn-toolbar flex-1 sm:flex-none"
            >
              <RotateCcw className="h-3.5 w-3.5" />
              恢复默认
            </button>
            <button
              onClick={handleSave}
              disabled={!isDirty || isSaving}
              className="btn-accent flex-1 sm:flex-none"
            >
              {update.isPending ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Save className="h-3.5 w-3.5" />
              )}
              保存设置
            </button>
          </div>
        )}
      </div>

      {(update.isError || reset.isError) && (
        <div className="rounded-lg border border-danger/20 bg-danger/10 px-3 py-2 text-xs text-danger">
          {errorMessage(update.error ?? reset.error)}
        </div>
      )}
      {update.isSuccess && !isDirty && (
        <div className="flex items-center gap-2 rounded-lg border border-success/20 bg-success/10 px-3 py-2 text-xs text-success">
          <CheckCircle2 className="h-3.5 w-3.5" />
          设置已保存，同步服务已应用新的配置。
        </div>
      )}

      <div className="flex flex-col items-start gap-6 lg:flex-row">
        <nav className="scrollbar-none flex w-full shrink-0 gap-1 overflow-x-auto rounded-xl border border-border bg-bg-card p-1 lg:w-52 lg:flex-col">
          {TABS.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => setActiveTab(id)}
              className={cn(
                'flex min-h-[40px] shrink-0 items-center gap-2 whitespace-nowrap rounded-lg px-3 py-2 text-left text-xs font-medium transition-colors lg:min-h-0 lg:w-full',
                activeTab === id
                  ? 'bg-accent/10 text-accent'
                  : 'text-fg-muted hover:bg-bg-elevated hover:text-fg',
              )}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {label}
            </button>
          ))}
        </nav>

        <div className="min-w-0 flex-1 space-y-4">
          {activeTab === 'account' && <AccountSettings />}
          {activeTab === 'general' && <GeneralSettings form={form} updateField={updateField} />}
          {activeTab === 'sync' && <SyncSettings form={form} updateField={updateField} />}
          {activeTab === 'alerts' && <AlertSettings form={form} updateField={updateField} />}
          {activeTab === 'data' && (
            <DataSettings
              form={form}
              updateField={updateField}
              onCleanup={handleCleanup}
              isCleaning={cleanup.isPending}
              cleanupMessage={cleanupMessage}
              cleanupError={cleanup.error}
            />
          )}
          {activeTab === 'about' && <AboutSettings status={status} />}
        </div>
      </div>
    </div>
  );
}

function AccountSettings() {
  const { data: user } = useAuth();
  const update = useUpdateAccount();
  const [email, setEmail] = useState(user?.email ?? '');
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  useEffect(() => {
    if (user?.email) setEmail(user.email);
  }, [user?.email]);

  const emailChanged = email.trim().toLowerCase() !== (user?.email ?? '').toLowerCase();
  const passwordChanged = newPassword.length > 0;
  const canSubmit = currentPassword.length > 0 && (emailChanged || passwordChanged) && !update.isPending;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!canSubmit) return;
    if (passwordChanged && newPassword.length < 8) return;
    if (passwordChanged && newPassword !== confirmPassword) return;
    update.mutate(
      {
        currentPassword,
        email: emailChanged ? email.trim() : undefined,
        newPassword: passwordChanged ? newPassword : undefined,
      },
      {
        onSuccess: () => {
          setCurrentPassword('');
          setNewPassword('');
          setConfirmPassword('');
        },
      },
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <SettingCard title="登录账号" description="修改系统登录邮箱和密码。改密码不影响 M-Team API Key。">
        <SettingRow label="登录邮箱" description="用来登录本系统，不是站点账号。">
          <input
            type="email"
            autoComplete="username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="field bg-bg-base sm:py-2 lg:w-64"
          />
        </SettingRow>
        <SettingRow label="当前密码" description="验证身份后才能修改。">
          <input
            type="password"
            autoComplete="current-password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            className="field bg-bg-base sm:py-2 lg:w-64"
          />
        </SettingRow>
        <SettingRow label="新密码" description="至少 8 位；不改密码请留空。">
          <input
            type="password"
            autoComplete="new-password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            className="field bg-bg-base sm:py-2 lg:w-64"
          />
        </SettingRow>
        <SettingRow label="确认新密码" description="与新密码保持一致。">
          <input
            type="password"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            className="field bg-bg-base sm:py-2 lg:w-64"
          />
        </SettingRow>
      </SettingCard>

      {passwordChanged && newPassword.length < 8 && (
        <p className="text-xs text-danger">新密码至少 8 位</p>
      )}
      {passwordChanged && newPassword.length >= 8 && confirmPassword && newPassword !== confirmPassword && (
        <p className="text-xs text-danger">两次输入的新密码不一致</p>
      )}
      {update.isError && (
        <p className="text-xs text-danger">{errorMessage(update.error)}</p>
      )}
      {update.isSuccess && (
        <p className="flex items-center gap-2 text-xs text-success">
          <CheckCircle2 className="h-3.5 w-3.5" />
          账号已更新，下次请用新邮箱/密码登录。
        </p>
      )}

      <button
        type="submit"
        disabled={!canSubmit || (passwordChanged && (newPassword.length < 8 || newPassword !== confirmPassword))}
        className="btn-accent"
      >
        {update.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />}
        保存账号
      </button>
    </form>
  );
}

function GeneralSettings({
  form,
  updateField,
}: {
  form: SettingsForm;
  updateField: <Key extends keyof SettingsForm>(key: Key, value: SettingsForm[Key]) => void;
}) {
  return (
    <SettingCard title="通用设置" description="设置系统显示名称、时区和列表默认分页数量。">
      <SettingRow label="系统名称" description="显示在系统标题和页面顶部。">
        <input
          value={form.systemName}
          onChange={(event) => updateField('systemName', event.target.value)}
          maxLength={50}
          className="field bg-bg-base sm:py-2 lg:w-64"
        />
      </SettingRow>
      <SettingRow label="时区" description="用于显示时间和数据维护截止时间。">
        <select
          value={form.timezone}
          onChange={(event) => updateField('timezone', event.target.value)}
          className="field bg-bg-base sm:py-2 lg:w-64"
        >
          {TIMEZONES.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </SettingRow>
      <SettingRow label="默认分页数量" description="种子、告警等列表首次打开时的默认展示数量。">
        <select
          value={form.defaultPageSize}
          onChange={(event) => updateField('defaultPageSize', Number(event.target.value))}
          className="field bg-bg-base sm:py-2 lg:w-64"
        >
          {PAGE_SIZES.map((size) => (
            <option key={size} value={size}>
              {size} 条
            </option>
          ))}
        </select>
      </SettingRow>
    </SettingCard>
  );
}

function SyncSettings({
  form,
  updateField,
}: {
  form: SettingsForm;
  updateField: <Key extends keyof SettingsForm>(key: Key, value: SettingsForm[Key]) => void;
}) {
  return (
    <SettingCard title="同步设置" description="控制系统自动把启用账号加入同步队列的频率。">
      <SettingRow label="启用定时同步" description="关闭后仍可在站点页面手动触发同步。">
        <Toggle checked={form.syncEnabled} onChange={(value) => updateField('syncEnabled', value)} />
      </SettingRow>
      <SettingRow label="同步周期" description="每轮定时同步会处理所有启用且允许同步的账号。">
        <select
          value={form.syncIntervalMinutes}
          onChange={(event) => updateField('syncIntervalMinutes', Number(event.target.value))}
          disabled={!form.syncEnabled}
          className="field bg-bg-base sm:py-2 lg:w-64"
        >
          {SYNC_INTERVALS.map((minutes) => (
            <option key={minutes} value={minutes}>
              每 {minutes} 分钟
            </option>
          ))}
        </select>
      </SettingRow>
      <div className="flex items-start gap-2 rounded-lg bg-info/10 px-3 py-2.5 text-xs text-info">
        <Clock3 className="mt-0.5 h-3.5 w-3.5 shrink-0" />
        修改并保存后，后端调度器会立即应用新的周期，无需重启容器。
      </div>
    </SettingCard>
  );
}

function AlertSettings({
  form,
  updateField,
}: {
  form: SettingsForm;
  updateField: <Key extends keyof SettingsForm>(key: Key, value: SettingsForm[Key]) => void;
}) {
  return (
    <SettingCard title="告警设置" description="控制同步失败等系统告警是否写入告警中心。">
      <SettingRow label="启用告警" description="关闭后不会创建新的同步失败告警，已有告警不受影响。">
        <Toggle checked={form.alertEnabled} onChange={(value) => updateField('alertEnabled', value)} />
      </SettingRow>
      <SettingRow label="告警保留天数" description="数据维护时只清理已忽略且超过期限的告警。">
        <RetentionSelect
          value={form.alertRetentionDays}
          options={RETENTION_OPTIONS}
          onChange={(value) => updateField('alertRetentionDays', value)}
        />
      </SettingRow>
    </SettingCard>
  );
}

function DataSettings({
  form,
  updateField,
  onCleanup,
  isCleaning,
  cleanupMessage,
  cleanupError,
}: {
  form: SettingsForm;
  updateField: <Key extends keyof SettingsForm>(key: Key, value: SettingsForm[Key]) => void;
  onCleanup: () => void;
  isCleaning: boolean;
  cleanupMessage: string | null;
  cleanupError: unknown;
}) {
  return (
    <>
      <SettingCard title="数据保留策略" description="设置历史数据保留时间，保存后点击下方按钮执行清理。">
        <SettingRow label="同步任务保留天数" description="同步任务及其详细日志的保留时间。">
          <RetentionSelect
            value={form.syncLogRetentionDays}
            options={RETENTION_OPTIONS}
            onChange={(value) => updateField('syncLogRetentionDays', value)}
          />
        </SettingRow>
        <SettingRow label="统计快照保留天数" description="用于趋势图的每日统计快照保留时间。">
          <RetentionSelect
            value={form.snapshotRetentionDays}
            options={SNAPSHOT_RETENTION_OPTIONS}
            onChange={(value) => updateField('snapshotRetentionDays', value)}
          />
        </SettingRow>
      </SettingCard>

      <SettingCard title="数据清理" description="只会清理超过保留期限的历史记录，以及已经忽略的旧告警。">
        <div className="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-center">
          <div>
            <p className="text-sm font-medium text-fg">立即执行清理</p>
            <p className="mt-0.5 text-xs text-fg-subtle">清理前请先保存上方的保留策略。</p>
          </div>
          <button
            onClick={onCleanup}
            disabled={isCleaning}
            className="btn-danger w-full sm:w-auto"
          >
            {isCleaning ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
            执行清理
          </button>
        </div>
        {cleanupMessage && (
          <p className="mt-3 flex items-center gap-2 text-xs text-success">
            <CheckCircle2 className="h-3.5 w-3.5" />
            {cleanupMessage}
          </p>
        )}
        {cleanupError != null ? (
          <p className="mt-3 text-xs text-danger">{errorMessage(cleanupError)}</p>
        ) : null}
      </SettingCard>
    </>
  );
}

function AboutSettings({ status }: { status?: SettingsStatus }) {
  return (
    <>
      <SettingCard title="关于系统" description="查看当前 PT Manager 的版本和运行环境。">
        <InfoRow label="版本" value={status?.version ?? '加载中'} />
        <InfoRow label="运行环境" value={status?.environment ?? '加载中'} />
        <InfoRow label="最近检查" value={status ? new Date(status.checkedAt).toLocaleString() : '加载中'} />
      </SettingCard>
      <SettingCard title="服务状态" description="设置页加载成功代表后端 API 当前可用。">
        <StatusRow icon={Database} label="数据库" status={status?.database} />
        <StatusRow icon={Server} label="Redis 队列" status={status?.redis} />
      </SettingCard>
    </>
  );
}

function SettingCard({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <section className="bento-card overflow-hidden">
      <div className="border-b border-border px-4 py-3">
        <h3 className="text-sm font-semibold text-fg">{title}</h3>
        <p className="mt-0.5 text-xs text-fg-subtle">{description}</p>
      </div>
      <div className="divide-y divide-border px-4">{children}</div>
    </section>
  );
}

function SettingRow({
  label,
  description,
  children,
}: {
  label: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <p className="text-sm font-medium text-fg">{label}</p>
        <p className="mt-0.5 text-xs text-fg-subtle">{description}</p>
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (value: boolean) => void }) {
  // 命中区在移动端撑到 40px，开关本身仍是 24×44
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className="flex h-10 w-11 shrink-0 items-center sm:h-6"
    >
      <span
        className={cn(
          'relative block h-6 w-11 rounded-full transition-colors',
          checked ? 'bg-accent' : 'bg-bg-elevated',
        )}
      >
        <span
          className={cn(
            'absolute top-1 h-4 w-4 rounded-full bg-white transition-transform',
            checked ? 'translate-x-6' : 'translate-x-1',
          )}
        />
      </span>
    </button>
  );
}

function RetentionSelect({
  value,
  options,
  onChange,
}: {
  value: number;
  options: number[];
  onChange: (value: number) => void;
}) {
  return (
    <select
      value={value}
      onChange={(event) => onChange(Number(event.target.value))}
      className="field bg-bg-base sm:py-2 lg:w-40"
    >
      {options.map((days) => (
        <option key={days} value={days}>
          {days} 天
        </option>
      ))}
    </select>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 py-3 text-sm">
      <span className="shrink-0 text-fg-muted">{label}</span>
      <span className="break-anywhere min-w-0 text-right text-fg">{value}</span>
    </div>
  );
}

function StatusRow({
  icon: Icon,
  label,
  status,
}: {
  icon: typeof Database;
  label: string;
  status?: 'connected' | 'error';
}) {
  const connected = status === 'connected';
  return (
    <div className="flex items-center justify-between gap-4 py-3 text-sm">
      <span className="flex items-center gap-2 text-fg-muted">
        <Icon className="h-4 w-4" />
        {label}
      </span>
      <span className={cn('flex items-center gap-1.5 text-xs', connected ? 'text-success' : 'text-danger')}>
        <span className={cn('h-1.5 w-1.5 rounded-full', connected ? 'bg-success' : 'bg-danger')} />
        {status ? (connected ? '已连接' : '异常') : '检查中'}
      </span>
    </div>
  );
}

function SettingsSkeleton() {
  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="h-10 w-64 animate-pulse rounded-lg bg-bg-card" />
      <div className="flex gap-6">
        <div className="hidden h-56 w-52 animate-pulse rounded-xl bg-bg-card lg:block" />
        <div className="h-72 flex-1 animate-pulse rounded-xl bg-bg-card" />
      </div>
    </div>
  );
}
