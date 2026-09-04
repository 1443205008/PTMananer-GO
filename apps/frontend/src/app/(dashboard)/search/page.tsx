'use client';

import { useEffect, useState, useRef } from 'react';
import { Search, Download, Users, Upload, Loader2, AlertTriangle, Zap, ArrowUpDown, ArrowUp, ArrowDown, Sprout, ChevronDown, Check, ListFilter } from 'lucide-react';
import { useSearchTorrents, useGenDlToken, useTeamList } from '@/hooks/use-search';
import { useStartFakeSeed } from '@/hooks/use-fake-seed';
import { useAccounts } from '@/hooks/use-accounts';
import { useSettings } from '@/hooks/use-settings';
import { useLocalStorage } from '@/hooks/use-local-storage';
import { formatBytes, formatRelativeTime } from '@/lib/formatters';
import { cn } from '@/lib/utils';
import type { TorrentSearchItem, SearchMode, SearchDiscount, SearchSortField, SearchTeam } from '@/types/api';

const DISCOUNT_LABELS: Record<string, { label: string; cls: string }> = {
  FREE:        { label: '免费', cls: 'bg-success/15 text-success border-success/20' },
  _2X_FREE:    { label: '2x免费', cls: 'bg-success/15 text-success border-success/20' },
  PERCENT_50:  { label: '50%', cls: 'bg-warning/15 text-warning border-warning/20' },
  _2X:         { label: '2x', cls: 'bg-accent/15 text-accent border-accent/20' },
  _2X_PERCENT_50: { label: '2x 50%', cls: 'bg-accent/15 text-accent border-accent/20' },
};

const MODE_OPTIONS: { value: SearchMode; label: string }[] = [
  { value: 'normal', label: '综合' },
  { value: 'movie',  label: '电影' },
  { value: 'tvshow', label: '剧集' },
  { value: 'music',  label: '音乐' },
  { value: 'anime',  label: '动漫' },
  { value: 'adult',  label: '成人' },
];

const DISCOUNT_OPTIONS: { value: SearchDiscount | ''; label: string }[] = [
  { value: '',           label: '全部优惠' },
  { value: 'FREE',       label: '免费' },
  { value: '_2X_FREE',   label: '2x免费' },
  { value: 'PERCENT_50', label: '50%' },
  { value: '_2X',        label: '2x上传' },
];

const SORT_OPTIONS: { value: SearchSortField; label: string }[] = [
  { value: 'CREATED_DATE',    label: '发布时间' },
  { value: 'SEEDERS',         label: '做种数' },
  { value: 'LEECHERS',        label: '下载数' },
  { value: 'TIMES_COMPLETED', label: '完成数' },
  { value: 'SIZE',            label: '文件大小' },
  { value: 'NAME',            label: '名称' },
];

// localStorage key 前缀，避免与其他模块冲突
const LS = 'search:';

export default function SearchPage() {
  const [keyword, setKeyword] = useLocalStorage(`${LS}keyword`, '');
  const [submittedKeyword, setSubmittedKeyword] = useLocalStorage(`${LS}submittedKeyword`, '');
  const [hasSearched, setHasSearched] = useLocalStorage(
    `${LS}hasSearched`,
    submittedKeyword.length > 0,
  );
  const [mode, setMode] = useLocalStorage<SearchMode>(`${LS}mode`, 'normal');
  const [discount, setDiscount] = useLocalStorage<SearchDiscount | ''>(`${LS}discount`, '');
  const [teamIds, setTeamIds] = useLocalStorage<number[]>(`${LS}teamIds`, []);
  const [sortField, setSortField] = useLocalStorage<SearchSortField>(`${LS}sortField`, 'CREATED_DATE');
  const [sortDirection, setSortDirection] = useLocalStorage<'ASC' | 'DESC'>(`${LS}sortDirection`, 'DESC');
  const [page, setPage] = useState(1); // 页码不持久化，每次打开从第 1 页开始
  const inputRef = useRef<HTMLInputElement>(null);
  const { data: settings } = useSettings();

  const params = {
    keyword: submittedKeyword, mode,
    discount: discount || undefined,
    teams: teamIds.length ? teamIds : undefined,
    sortField, sortDirection,
    page, pageSize: settings?.defaultPageSize ?? 20,
  };
  const { data, isLoading, isError, isFetching, error, refetch } = useSearchTorrents(params, hasSearched);
  const { data: teams } = useTeamList();
  const { data: accounts } = useAccounts();
  const defaultAccountId = accounts?.find(
    (account) => account.isEnabled && account.status === 'ACTIVE',
  )?.id ?? '';

  function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    const kw = keyword.trim();
    setSubmittedKeyword(kw);
    setHasSearched(true);
    setPage(1);
  }

  function handleSortField(f: SearchSortField) {
    if (f === sortField) {
      // 同一字段再点 → 切换方向
      setSortDirection((d) => d === 'DESC' ? 'ASC' : 'DESC');
    } else {
      setSortField(f);
      setSortDirection('DESC');
    }
    setPage(1);
  }

  const totalPages = data ? Math.ceil(data.total / data.pageSize) : 1;

  return (
    <div className="space-y-5">
      {/* 标题 + 搜索框 */}
      <div>
        <h2 className="text-lg font-semibold text-fg">站内搜索</h2>
        <p className="mt-0.5 text-sm text-fg-subtle">搜索 M-Team 站内种子</p>
      </div>

      <form onSubmit={handleSearch} className="flex gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle" />
          <input
            ref={inputRef}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="输入关键词、IMDb、DMM 番号…"
            className="field pl-9 pr-3 sm:py-2"
          />
        </div>
        <button type="submit" disabled={isFetching}
          className="btn-accent px-4 text-sm sm:min-h-[36px]">
          {isFetching ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}
          搜索
        </button>
      </form>

      {/* 过滤栏 */}
      <div className="flex flex-wrap items-center gap-2">
        {/* 分类模式 —— 窄屏横向滚动，不再挤压 */}
        <div className="seg-bar max-w-full">
          {MODE_OPTIONS.map((opt) => (
            <button key={opt.value} onClick={() => { setMode(opt.value); setPage(1); }}
              className={cn('seg-btn',
                mode === opt.value ? 'seg-btn-active' : 'seg-btn-idle')}>
              {opt.label}
            </button>
          ))}
        </div>

        {/* 优惠筛选 */}
        <select value={discount} onChange={(e) => { setDiscount(e.target.value as SearchDiscount | ''); setPage(1); }}
          className="field sm:w-auto sm:text-xs">
          {DISCOUNT_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>

        {/* 制作组筛选 */}
        <TeamMultiSelect
          teams={teams}
          selectedIds={teamIds}
          onChange={(ids) => { setTeamIds(ids); setPage(1); }}
        />

        {/* 排序 —— 6 个按钮在窄屏必然溢出，改为横向滚动 */}
        <div className="seg-bar max-w-full">
          <ArrowUpDown className="ml-1.5 h-3 w-3 text-fg-subtle shrink-0" />
          {SORT_OPTIONS.map((opt) => (
            <button key={opt.value} onClick={() => handleSortField(opt.value)}
              className={cn('seg-btn',
                sortField === opt.value ? 'seg-btn-active' : 'seg-btn-idle')}>
              {opt.label}
              {sortField === opt.value && (
                sortDirection === 'DESC'
                  ? <ArrowDown className="h-2.5 w-2.5" />
                  : <ArrowUp className="h-2.5 w-2.5" />
              )}
            </button>
          ))}
        </div>

        {data && (
          <span className="ml-auto text-xs text-fg-subtle">共 {data.total.toLocaleString()} 条结果</span>
        )}
      </div>

      {/* 结果区 */}
      {!hasSearched ? (
        <EmptyPrompt />
      ) : isLoading ? (
        <SearchSkeleton />
      ) : isError ? (
        <ErrorState
          message={error instanceof Error ? error.message : undefined}
          onRetry={() => void refetch()}
        />
      ) : !data || data.data.length === 0 ? (
        <NoResults keyword={submittedKeyword} />
      ) : (
        <>
          <div className="space-y-2">
            {data.data.map((t) => <TorrentRow key={t.id} torrent={t} accountId={defaultAccountId} />)}
          </div>
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2">
              <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}
                className="btn-toolbar disabled:opacity-40">上一页</button>
              <span className="text-xs text-fg-subtle">{page} / {totalPages}</span>
              <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}
                className="btn-toolbar disabled:opacity-40">下一页</button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function TeamMultiSelect({
  teams,
  selectedIds,
  onChange,
}: {
  teams?: SearchTeam[];
  selectedIds: number[];
  onChange: (ids: number[]) => void;
}) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const allIds = teams?.map((team) => team.id) ?? [];

  useEffect(() => {
    if (!open) return;
    function handleOutsideClick(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener('mousedown', handleOutsideClick);
    return () => document.removeEventListener('mousedown', handleOutsideClick);
  }, [open]);

  function toggleTeam(id: number) {
    onChange(
      selectedIds.includes(id)
        ? selectedIds.filter((selectedId) => selectedId !== id)
        : [...selectedIds, id],
    );
  }

  return (
    <div ref={containerRef} className="relative w-full sm:w-auto">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        className={cn(
          'flex w-full min-h-[40px] items-center gap-1.5 rounded-lg border px-2.5 text-xs outline-none transition-colors sm:w-auto sm:min-h-[30px] sm:py-1.5',
          selectedIds.length
            ? 'border-accent/30 bg-accent/10 text-accent'
            : 'border-border bg-bg-card text-fg-muted hover:bg-bg-elevated hover:text-fg',
        )}
      >
        <ListFilter className="h-3.5 w-3.5 shrink-0" />
        {selectedIds.length ? `制作组（${selectedIds.length}）` : '全部制作组'}
        <ChevronDown className={cn('ml-auto h-3.5 w-3.5 shrink-0 transition-transform sm:ml-0', open && 'rotate-180')} />
      </button>

      {open && (
        // 面板宽度在窄屏收敛到视口内（主内容区左右各 16px 内边距），避免右边缘溢出
        <div className="absolute left-0 top-full z-30 mt-1 w-[min(16rem,calc(100vw_-_2rem))] max-w-[calc(100vw_-_2rem)] overflow-hidden rounded-lg border border-border bg-bg-card shadow-xl sm:w-64">
          <div className="flex items-center justify-between gap-2 border-b border-border px-3 py-2">
            <span className="min-w-0 truncate text-xs font-medium text-fg">选择制作组</span>
            <div className="flex shrink-0 items-center gap-2 text-[11px]">
              <button
                type="button"
                onClick={() => onChange(allIds)}
                className="inline-flex min-h-[40px] items-center px-2 text-accent hover:underline sm:min-h-0 sm:px-0"
              >
                全选
              </button>
              <button
                type="button"
                onClick={() => onChange([])}
                className="inline-flex min-h-[40px] items-center px-2 text-fg-subtle hover:text-fg sm:min-h-0 sm:px-0"
              >
                清空
              </button>
            </div>
          </div>
          <div className="max-h-64 overflow-y-auto p-1.5">
            {!teams ? (
              <p className="px-2 py-3 text-xs text-fg-subtle">制作组加载中…</p>
            ) : teams.length === 0 ? (
              <p className="px-2 py-3 text-xs text-fg-subtle">暂无制作组</p>
            ) : (
              teams.map((team) => {
                const checked = selectedIds.includes(team.id);
                return (
                  <button
                    key={team.id}
                    type="button"
                    onClick={() => toggleTeam(team.id)}
                    className="flex min-h-[40px] w-full cursor-pointer items-center gap-2 rounded-md px-2 text-xs text-fg-muted hover:bg-bg-elevated hover:text-fg sm:min-h-[26px] sm:py-1.5"
                  >
                    <span
                      className={cn(
                        'flex h-4 w-4 shrink-0 items-center justify-center rounded border sm:h-3.5 sm:w-3.5',
                        checked
                          ? 'border-accent bg-accent text-white'
                          : 'border-border bg-bg-base',
                      )}
                    >
                      {checked && <Check className="h-3 w-3" />}
                    </span>
                    <span className="min-w-0 truncate">{team.name}</span>
                  </button>
                );
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ── 子组件 ─────────────────────────────────────────────────────────────────────

function TorrentRow({ torrent, accountId }: { torrent: TorrentSearchItem; accountId: string }) {
  const dlMutation = useGenDlToken();
  const seedMutation = useStartFakeSeed();
  const discountCfg = torrent.discount ? DISCOUNT_LABELS[torrent.discount] : null;

  async function handleDownload() {
    try {
      const { url } = await dlMutation.mutateAsync(torrent.id);
      if (url) window.open(url, '_blank');
    } catch { /* 静默失败 */ }
  }

  async function handleSeed() {
    if (!accountId) return;
    await seedMutation.mutateAsync({ accountId, torrentId: torrent.id, torrentName: torrent.name });
  }

  const seedDone = seedMutation.isSuccess || torrent.isSeeding;
  const seedError = seedMutation.error as Error | null;

  return (
    <div className="bento-card py-3">
      {/* 窄屏竖排：种子名独占整行宽度，操作按钮落到下一行 */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
      <div className="flex-1 min-w-0">
        <div className="flex flex-wrap items-center gap-1.5 mb-1">
          {discountCfg && (
            <span className={cn('rounded border px-1.5 py-0.5 text-[10px] font-semibold', discountCfg.cls)}>
              {discountCfg.label}
            </span>
          )}
          <p className="text-sm font-medium text-fg leading-snug line-clamp-2">{torrent.name}</p>
        </div>
        {torrent.smallDescr && (
          <p className="text-xs text-fg-subtle truncate mb-1.5">{torrent.smallDescr}</p>
        )}
        <div className="flex flex-wrap gap-3 text-xs text-fg-subtle">
          <span>{formatBytes(BigInt(torrent.size))}</span>
          <span className="flex items-center gap-1 text-success">
            <Upload className="h-3 w-3" />{torrent.seeders}
          </span>
          <span className="flex items-center gap-1 text-blue-400">
            <Download className="h-3 w-3" />{torrent.leechers}
          </span>
          <span className="flex items-center gap-1">
            <Users className="h-3 w-3" />{torrent.timesCompleted}
          </span>
          {torrent.createdDate && <span>{formatRelativeTime(torrent.createdDate)}</span>}
          {torrent.imdb && (
            <a href={`https://www.imdb.com/title/${torrent.imdb}`} target="_blank" rel="noreferrer"
              className="text-accent hover:underline">IMDb</a>
          )}
        </div>
      </div>
      <div className="flex gap-1.5 sm:shrink-0">
        {/* 保种按钮 */}
        <button onClick={() => void handleSeed()}
          disabled={seedMutation.isPending || seedDone || !accountId}
          title={torrent.isSeeding ? '该账号正在做种' : seedMutation.isSuccess ? '保种任务已创建' : '开始保种'}
          className={cn(
            'btn-toolbar flex-1 sm:flex-none',
            seedDone
              ? 'border-success/30 bg-success/10 text-success hover:bg-success/10 hover:text-success'
              : 'hover:border-success/30 hover:bg-success/10 hover:text-success',
          )}>
          {seedMutation.isPending
            ? <Loader2 className="h-3.5 w-3.5 animate-spin" />
            : <Sprout className="h-3.5 w-3.5" />}
          {torrent.isSeeding ? '做种中' : seedMutation.isSuccess ? '已保种' : '保种'}
        </button>
        {/* 下载按钮 */}
        <button onClick={() => void handleDownload()} disabled={dlMutation.isPending}
          title="获取下载链接"
          className="btn-toolbar flex-1 sm:flex-none">
          {dlMutation.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Download className="h-3.5 w-3.5" />}
          下载
        </button>
      </div>
      </div>
      {/* 保种失败提示 —— 独占一行，不再和标题/操作列抢宽度 */}
      {seedError && (
        <p className="break-anywhere mt-2 text-[10px] text-danger" title={seedError.message}>
          保种失败：{seedError.message}
        </p>
      )}
    </div>
  );
}

function EmptyPrompt() {
  return (
    <div className="flex min-h-[300px] flex-col items-center justify-center gap-3 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-accent/10">
        <Search className="h-6 w-6 text-accent" />
      </div>
      <p className="text-sm font-medium text-fg">搜索 M-Team 站内种子</p>
      <p className="text-xs text-fg-subtle">支持关键词、IMDb 编号、DMM 番号，也可以直接搜索全部种子</p>
    </div>
  );
}

function NoResults({ keyword }: { keyword: string }) {
  return (
    <div className="flex min-h-[200px] flex-col items-center justify-center gap-2 text-center">
      <Zap className="h-8 w-8 text-fg-subtle" />
      <p className="text-sm text-fg">
        {keyword ? `没有找到「${keyword}」相关的种子` : '没有找到符合筛选条件的种子'}
      </p>
      <p className="text-xs text-fg-subtle">尝试换个关键词或切换分类</p>
    </div>
  );
}

function ErrorState({ message, onRetry }: { message?: string; onRetry: () => void }) {
  return (
    <div className="flex min-h-[200px] flex-col items-center justify-center gap-2 text-center">
      <AlertTriangle className="h-8 w-8 text-danger" />
      <p className="text-sm text-fg">搜索失败</p>
      <p className="max-w-md text-xs text-fg-subtle">{message || '请稍后重试'}</p>
      <button type="button" onClick={onRetry} className="btn-toolbar mt-1">
        重试
      </button>
    </div>
  );
}

function SearchSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="h-20 animate-pulse rounded-xl bg-bg-card" />
      ))}
    </div>
  );
}
