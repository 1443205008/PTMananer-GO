import axios from 'axios';

/**
 * 统一 API Client
 * 所有请求指向 PT Manager Backend，绝不直接请求 M-Team API
 */
const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:4000/api',
  // 搜索等接口要等 M-Team 上游，15s 会在后端还没返回时先失败
  timeout: 45_000,
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
});

function extractErrorMessage(err: unknown): string {
  const axiosErr = err as {
    code?: string;
    message?: string;
    response?: { data?: { message?: string | string[] } };
  };
  if (axiosErr.code === 'ECONNABORTED') return '请求超时，请稍后重试';
  const raw = axiosErr.response?.data?.message;
  if (Array.isArray(raw) && raw.length) return raw.join('；');
  if (typeof raw === 'string' && raw.trim()) return raw;
  return axiosErr.message || '请求失败';
}

// ─── Response Interceptor ─────────────────────────────────────────────────
apiClient.interceptors.response.use(
  (res) => res,
  (err) => {
    const status = err.response?.status;
    const message = extractErrorMessage(err);
    console.error(`API error [${status}]: ${message}`);
    return Promise.reject(Object.assign(new Error(message), { status, cause: err }));
  },
);

export default apiClient;
