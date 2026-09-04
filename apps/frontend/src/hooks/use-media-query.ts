'use client';

import { useEffect, useState } from 'react';

/**
 * SSR 安全的媒体查询。首帧一律返回 false（服务端没有视口），
 * 挂载后再同步真实值，所以只用于 JS 必须知道断点的场合
 * （例如 recharts 的数值型 props）；纯样式请优先用 Tailwind 断点。
 */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(false);

  useEffect(() => {
    const mql = window.matchMedia(query);
    setMatches(mql.matches);

    const onChange = (e: MediaQueryListEvent) => setMatches(e.matches);
    mql.addEventListener('change', onChange);
    return () => mql.removeEventListener('change', onChange);
  }, [query]);

  return matches;
}

/** 对齐 Tailwind 的 sm 断点（640px） */
export function useIsMobile(): boolean {
  return useMediaQuery('(max-width: 639px)');
}
