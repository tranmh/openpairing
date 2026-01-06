import { vi } from 'vitest';
import type { User } from '@/lib/api';

/**
 * Creates a mock auth store state for testing.
 */
export function createMockAuthState(overrides?: Partial<MockAuthState>): MockAuthState {
  return {
    user: null,
    accessToken: null,
    isAuthenticated: false,
    isLoading: false,
    isHydrated: true,
    error: null,
    _refreshInProgress: null,
    login: vi.fn().mockResolvedValue(true),
    register: vi.fn().mockResolvedValue(true),
    logout: vi.fn().mockResolvedValue(undefined),
    refreshToken: vi.fn().mockResolvedValue(true),
    clearError: vi.fn(),
    setUser: vi.fn(),
    setHydrated: vi.fn(),
    ...overrides,
  };
}

/**
 * Creates a mock authenticated state.
 */
export function createAuthenticatedState(user?: User): MockAuthState {
  return createMockAuthState({
    user: user ?? mockUser,
    accessToken: 'test-access-token',
    isAuthenticated: true,
    isHydrated: true,
  });
}

/**
 * Creates a mock loading state.
 */
export function createLoadingState(): MockAuthState {
  return createMockAuthState({
    isLoading: true,
  });
}

/**
 * Creates a mock error state.
 */
export function createErrorState(error: string): MockAuthState {
  return createMockAuthState({
    error,
    isAuthenticated: false,
  });
}

// Type for the mock auth state
interface MockAuthState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  isHydrated: boolean;
  error: string | null;
  _refreshInProgress: Promise<boolean> | null;
  login: ReturnType<typeof vi.fn>;
  register: ReturnType<typeof vi.fn>;
  logout: ReturnType<typeof vi.fn>;
  refreshToken: ReturnType<typeof vi.fn>;
  clearError: ReturnType<typeof vi.fn>;
  setUser: ReturnType<typeof vi.fn>;
  setHydrated: ReturnType<typeof vi.fn>;
}

// Default mock user
const mockUser: User = {
  id: 'user-1',
  email: 'test@example.com',
  role: 'player',
  preferredLanguage: 'de',
  createdAt: '2024-01-01T00:00:00Z',
};

/**
 * Mock the useAuthStore hook.
 * Usage in tests:
 *
 * vi.mock('@/store/auth', () => ({
 *   useAuthStore: vi.fn(() => createAuthenticatedState()),
 * }));
 */
export function mockUseAuthStore(state: MockAuthState) {
  return vi.fn(() => state);
}
