'use client';

import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { useIsMobile } from '@/hooks/use-media-query';

export interface TrendChartPoint {
  date: string;
  value: number;
}

/** 窄屏 Y 轴刻度：12.4T / 8.1G / 350M / 1.2K，保证 34px 内放得下 */
function compactNumber(v: number): string {
  const abs = Math.abs(v);
  const units: [number, string][] = [
    [1e12, 'T'],
    [1e9, 'G'],
    [1e6, 'M'],
    [1e3, 'K'],
  ];
  for (const [size, suffix] of units) {
    if (abs >= size) {
      const n = v / size;
      return `${n >= 100 ? Math.round(n) : Number(n.toFixed(1))}${suffix}`;
    }
  }
  return String(Math.round(v));
}

interface TrendChartProps {
  data: TrendChartPoint[];
  /** 颜色，默认 accent indigo */
  color?: string;
  /** Y 轴 / tooltip 值格式化 */
  formatValue?: (v: number) => string;
  /** 图表唯一 id（用于渐变 defs，避免多个图冲突） */
  gradientId: string;
}

/**
 * 通用趋势面积图（dark theme）
 *
 * 用于仪表盘上传趋势 / 分享率趋势等。
 * 空数据时显示占位文案，避免 recharts 报错。
 */
export function TrendChart({
  data,
  color = '#6366F1',
  formatValue = (v) => String(v),
  gradientId,
}: TrendChartProps) {
  // recharts 的尺寸是数值 props，拿不到 CSS 断点，只能在 JS 里判断
  const isMobile = useIsMobile();

  if (!data || data.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center px-2 text-center">
        <p className="text-xs text-fg-subtle">暂无历史数据，完成首次同步后显示</p>
      </div>
    );
  }

  // 手机上 52px 的 Y 轴要吃掉 ~17% 卡片宽度，收窄并简写刻度
  const axisWidth = isMobile ? 34 : 52;
  const axisFontSize = isMobile ? 10 : 11;

  return (
    <ResponsiveContainer width="100%" height={isMobile ? 150 : 180}>
      <AreaChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
        <defs>
          <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.35} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#27272A" vertical={false} />
        <XAxis
          dataKey="date"
          tick={{ fill: '#71717A', fontSize: axisFontSize }}
          tickLine={false}
          axisLine={{ stroke: '#27272A' }}
          tickFormatter={(d: string) => d.slice(5)} // MM-DD
          minTickGap={isMobile ? 32 : 24}
        />
        <YAxis
          tick={{ fill: '#71717A', fontSize: axisFontSize }}
          tickLine={false}
          axisLine={false}
          width={axisWidth}
          tickFormatter={(v: number) => (isMobile ? compactNumber(v) : formatValue(v))}
        />
        <Tooltip
          contentStyle={{
            background: '#18181B',
            border: '1px solid #27272A',
            borderRadius: 8,
            fontSize: 12,
            maxWidth: '60vw',
          }}
          labelStyle={{ color: '#A1A1AA' }}
          itemStyle={{ color: '#FAFAFA' }}
          formatter={(v: number) => [formatValue(v), '']}
        />
        <Area
          type="monotone"
          dataKey="value"
          stroke={color}
          strokeWidth={2}
          fill={`url(#${gradientId})`}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}
