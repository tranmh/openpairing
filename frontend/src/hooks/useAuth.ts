'use client';

import { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore } from '@/store/auth';

/**
 * Hook for authentication state and actions
 */
export function useAuth() {
  const store = useAuthStore();

  return {
    user: store.user,
    token: store.accessToken,
    isAuthenticated: store.isAuthenticated,
    isLoading: store.isLoading,
    isHydrated: store.isHydrated,
    error: store.error,
    login: store.login,
    register: store.register,
    logout: store.logout,
    clearError: store.clearError,
  };
}

/**
 * Hook that redirects to login if not authenticated
 */
export function useRequireAuth(redirectTo: string = '/login') {
  const router = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, isLoading, isHydrated } = useAuthStore();

  useEffect(() => {
    // Wait for hydration before making auth decisions
    if (!isHydrated || isLoading) return;

    if (!isAuthenticated) {
      // Store the attempted URL for redirect after login
      const returnUrl = encodeURIComponent(pathname);
      router.push(`${redirectTo}?returnUrl=${returnUrl}`);
    }
  }, [isAuthenticated, isLoading, isHydrated, router, pathname, redirectTo]);

  return { isAuthenticated, isLoading, isHydrated };
}

/**
 * Hook that redirects away if already authenticated
 */
export function useRedirectIfAuthenticated(redirectTo: string = '/tournaments') {
  const router = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, isLoading, isHydrated } = useAuthStore();

  useEffect(() => {
    // Wait for hydration before making auth decisions
    if (!isHydrated || isLoading) return;

    // Prevent redirect if already on the target page
    if (pathname === redirectTo) return;

    if (isAuthenticated) {
      router.push(redirectTo);
    }
  }, [isAuthenticated, isLoading, isHydrated, router, redirectTo, pathname]);

  return { isAuthenticated, isLoading, isHydrated };
}

/**
 * Hook to check if user has required role
 */
export function useRequireRole(allowedRoles: string[], redirectTo: string = '/tournaments') {
  const router = useRouter();
  const { user, isAuthenticated, isLoading, isHydrated } = useAuthStore();

  useEffect(() => {
    // Wait for hydration before making auth decisions
    if (!isHydrated || isLoading) return;

    if (!isAuthenticated) {
      router.push('/login');
      return;
    }

    if (user && !allowedRoles.includes(user.role)) {
      router.push(redirectTo);
    }
  }, [user, isAuthenticated, isLoading, isHydrated, router, allowedRoles, redirectTo]);

  const hasRole = user ? allowedRoles.includes(user.role) : false;

  return { hasRole, isLoading, isHydrated };
}
