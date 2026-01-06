import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useAuth, useRequireAuth, useRedirectIfAuthenticated, useRequireRole } from './useAuth';

// Mock Next.js navigation
const mockPush = vi.fn();
const mockPathname = '/current-page';

vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => mockPathname,
}));

// Mock the auth store
const mockStore = {
  user: null as { id: string; email: string; role: string } | null,
  accessToken: null as string | null,
  isAuthenticated: false,
  isLoading: false,
  isHydrated: true,
  error: null as string | null,
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
  clearError: vi.fn(),
};

vi.mock('@/store/auth', () => ({
  useAuthStore: () => mockStore,
}));

describe('useAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStore.user = null;
    mockStore.accessToken = null;
    mockStore.isAuthenticated = false;
    mockStore.isLoading = false;
    mockStore.isHydrated = true;
    mockStore.error = null;
  });

  it('returns auth state from store', () => {
    mockStore.user = { id: '1', email: 'test@example.com', role: 'player' };
    mockStore.accessToken = 'token';
    mockStore.isAuthenticated = true;

    const { result } = renderHook(() => useAuth());

    expect(result.current.user).toEqual(mockStore.user);
    expect(result.current.token).toBe('token');
    expect(result.current.isAuthenticated).toBe(true);
  });

  it('returns auth actions from store', () => {
    const { result } = renderHook(() => useAuth());

    expect(result.current.login).toBe(mockStore.login);
    expect(result.current.register).toBe(mockStore.register);
    expect(result.current.logout).toBe(mockStore.logout);
    expect(result.current.clearError).toBe(mockStore.clearError);
  });

  it('returns loading state', () => {
    mockStore.isLoading = true;

    const { result } = renderHook(() => useAuth());

    expect(result.current.isLoading).toBe(true);
  });

  it('returns error state', () => {
    mockStore.error = 'Login failed';

    const { result } = renderHook(() => useAuth());

    expect(result.current.error).toBe('Login failed');
  });
});

describe('useRequireAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStore.user = null;
    mockStore.isAuthenticated = false;
    mockStore.isLoading = false;
    mockStore.isHydrated = true;
  });

  it('redirects to login when not authenticated', async () => {
    mockStore.isAuthenticated = false;
    mockStore.isHydrated = true;

    renderHook(() => useRequireAuth());

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith(
        `/login?returnUrl=${encodeURIComponent(mockPathname)}`
      );
    });
  });

  it('does not redirect when authenticated', async () => {
    mockStore.isAuthenticated = true;
    mockStore.isHydrated = true;

    renderHook(() => useRequireAuth());

    // Wait a tick to ensure useEffect has run
    await waitFor(() => {
      expect(mockPush).not.toHaveBeenCalled();
    });
  });

  it('does not redirect while loading', async () => {
    mockStore.isAuthenticated = false;
    mockStore.isLoading = true;
    mockStore.isHydrated = true;

    renderHook(() => useRequireAuth());

    await waitFor(() => {
      expect(mockPush).not.toHaveBeenCalled();
    });
  });

  it('does not redirect before hydration', async () => {
    mockStore.isAuthenticated = false;
    mockStore.isHydrated = false;

    renderHook(() => useRequireAuth());

    await waitFor(() => {
      expect(mockPush).not.toHaveBeenCalled();
    });
  });

  it('uses custom redirect path', async () => {
    mockStore.isAuthenticated = false;
    mockStore.isHydrated = true;

    renderHook(() => useRequireAuth('/custom-login'));

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith(
        `/custom-login?returnUrl=${encodeURIComponent(mockPathname)}`
      );
    });
  });

  it('returns auth state', () => {
    mockStore.isAuthenticated = true;
    mockStore.isLoading = false;
    mockStore.isHydrated = true;

    const { result } = renderHook(() => useRequireAuth());

    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.isHydrated).toBe(true);
  });
});

describe('useRedirectIfAuthenticated', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStore.isAuthenticated = false;
    mockStore.isLoading = false;
    mockStore.isHydrated = true;
  });

  it('redirects when authenticated', async () => {
    mockStore.isAuthenticated = true;
    mockStore.isHydrated = true;

    renderHook(() => useRedirectIfAuthenticated());

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/tournaments');
    });
  });

  it('does not redirect when not authenticated', async () => {
    mockStore.isAuthenticated = false;
    mockStore.isHydrated = true;

    renderHook(() => useRedirectIfAuthenticated());

    await waitFor(() => {
      expect(mockPush).not.toHaveBeenCalled();
    });
  });

  it('does not redirect while loading', async () => {
    mockStore.isAuthenticated = true;
    mockStore.isLoading = true;
    mockStore.isHydrated = true;

    renderHook(() => useRedirectIfAuthenticated());

    await waitFor(() => {
      expect(mockPush).not.toHaveBeenCalled();
    });
  });

  it('does not redirect before hydration', async () => {
    mockStore.isAuthenticated = true;
    mockStore.isHydrated = false;

    renderHook(() => useRedirectIfAuthenticated());

    await waitFor(() => {
      expect(mockPush).not.toHaveBeenCalled();
    });
  });

  it('uses custom redirect path', async () => {
    mockStore.isAuthenticated = true;
    mockStore.isHydrated = true;

    renderHook(() => useRedirectIfAuthenticated('/dashboard'));

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/dashboard');
    });
  });
});

describe('useRequireRole', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStore.user = null;
    mockStore.isAuthenticated = false;
    mockStore.isLoading = false;
    mockStore.isHydrated = true;
  });

  it('redirects to login when not authenticated', async () => {
    mockStore.isAuthenticated = false;
    mockStore.isHydrated = true;

    renderHook(() => useRequireRole(['admin']));

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/login');
    });
  });

  it('redirects when user lacks required role', async () => {
    mockStore.isAuthenticated = true;
    mockStore.isHydrated = true;
    mockStore.user = { id: '1', email: 'test@example.com', role: 'player' };

    renderHook(() => useRequireRole(['admin', 'arbiter']));

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/tournaments');
    });
  });

  it('does not redirect when user has required role', async () => {
    mockStore.isAuthenticated = true;
    mockStore.isHydrated = true;
    mockStore.user = { id: '1', email: 'test@example.com', role: 'admin' };

    renderHook(() => useRequireRole(['admin', 'arbiter']));

    await waitFor(() => {
      expect(mockPush).not.toHaveBeenCalled();
    });
  });

  it('uses custom redirect path', async () => {
    mockStore.isAuthenticated = true;
    mockStore.isHydrated = true;
    mockStore.user = { id: '1', email: 'test@example.com', role: 'player' };

    renderHook(() => useRequireRole(['admin'], '/unauthorized'));

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/unauthorized');
    });
  });

  it('returns hasRole correctly', () => {
    mockStore.isAuthenticated = true;
    mockStore.user = { id: '1', email: 'test@example.com', role: 'admin' };

    const { result } = renderHook(() => useRequireRole(['admin', 'arbiter']));

    expect(result.current.hasRole).toBe(true);
  });

  it('returns hasRole false when user lacks role', () => {
    mockStore.isAuthenticated = true;
    mockStore.user = { id: '1', email: 'test@example.com', role: 'player' };

    const { result } = renderHook(() => useRequireRole(['admin']));

    expect(result.current.hasRole).toBe(false);
  });

  it('returns hasRole false when user is null', () => {
    mockStore.isAuthenticated = false;
    mockStore.user = null;

    const { result } = renderHook(() => useRequireRole(['admin']));

    expect(result.current.hasRole).toBe(false);
  });
});
