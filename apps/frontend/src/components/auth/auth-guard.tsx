'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Loader2 } from 'lucide-react';
import { useAuth } from '@/hooks/use-auth';

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { data: user, isLoading, isError } = useAuth();

  useEffect(() => {
    if (isError) router.replace('/login');
  }, [isError, router]);

  if (isLoading || isError || !user) {
    return (
      <div className="flex min-h-dvh items-center justify-center bg-bg-base text-fg-subtle">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    );
  }

  return <>{children}</>;
}
