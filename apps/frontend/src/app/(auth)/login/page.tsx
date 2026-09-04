'use client';

import { FormEvent, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { LockKeyhole, Loader2, LogIn, ShieldCheck } from 'lucide-react';
import { useLogin } from '@/hooks/use-auth';

function getErrorMessage(error: unknown): string {
  const response = (error as { response?: { data?: { message?: string | string[] } } })?.response;
  const message = response?.data?.message;
  if (Array.isArray(message)) return message.join('、');
  if (message) return message;
  return '登录失败，请检查后端服务是否正常运行';
}

export default function LoginPage() {
  const router = useRouter();
  const loginMutation = useLogin();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  useEffect(() => {
    if (loginMutation.isSuccess) router.replace('/');
  }, [loginMutation.isSuccess, router]);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    loginMutation.mutate({ email: email.trim(), password });
  }

  const systemName = 'PT Manager';

  return (
    <main className="flex min-h-dvh items-center justify-center bg-bg-base px-4 py-8">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-accent text-white shadow-lg shadow-accent/20">
            <ShieldCheck className="h-7 w-7" />
          </div>
          <h1 className="mt-4 text-xl font-semibold text-fg">{systemName}</h1>
          <p className="mt-1 text-sm text-fg-subtle">登录后继续使用管理平台</p>
        </div>

        <form onSubmit={handleSubmit} className="bento-card space-y-4">
          <div>
            <label htmlFor="email" className="mb-1.5 block text-xs font-medium text-fg-muted">
              管理员邮箱
            </label>
            <input
              id="email"
              type="email"
              autoComplete="username"
              required
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="admin@pt-manager.local"
              className="field bg-bg-base"
            />
          </div>

          <div>
            <label htmlFor="password" className="mb-1.5 block text-xs font-medium text-fg-muted">
              登录密码
            </label>
            <div className="relative">
              <LockKeyhole className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle" />
              <input
                id="password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                className="field bg-bg-base pl-9"
              />
            </div>
          </div>

          {loginMutation.isError && (
            <p className="rounded-lg border border-danger/20 bg-danger/10 px-3 py-2 text-xs text-danger">
              {getErrorMessage(loginMutation.error)}
            </p>
          )}

          <button
            type="submit"
            disabled={loginMutation.isPending}
            className="btn-accent w-full"
          >
            {loginMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <LogIn className="h-4 w-4" />
            )}
            登录
          </button>
        </form>

        <p className="mt-5 text-center text-xs text-fg-subtle">
          登录凭据由服务端环境变量配置
        </p>
      </div>
    </main>
  );
}
