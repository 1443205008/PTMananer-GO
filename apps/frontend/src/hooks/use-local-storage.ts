import { useState, useCallback } from 'react';

/**
 * 与 localStorage 同步的 useState 替代品。
 * - 初始值从 localStorage 读取，写入时同步回存
 * - SSR 安全（window 不存在时回退到 initialValue）
 * - JSON 序列化失败时静默回退，不抛出异常
 */
export function useLocalStorage<T>(key: string, initialValue: T) {
  const [storedValue, setStoredValue] = useState<T>(() => {
    if (typeof window === 'undefined') return initialValue;
    try {
      const item = window.localStorage.getItem(key);
      return item !== null ? (JSON.parse(item) as T) : initialValue;
    } catch {
      return initialValue;
    }
  });

  const setValue = useCallback(
    (value: T | ((prev: T) => T)) => {
      setStoredValue((prev) => {
        const next = typeof value === 'function' ? (value as (p: T) => T)(prev) : value;
        try {
          window.localStorage.setItem(key, JSON.stringify(next));
        } catch {
          // 存储空间不足或隐私模式时静默忽略
        }
        return next;
      });
    },
    [key],
  );

  return [storedValue, setValue] as const;
}
