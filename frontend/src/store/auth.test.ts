import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act } from '@testing-library/react';
import { useAuthStore } from './auth';

// Mock the API module
vi.mock('@/lib/api', () => ({
  api: {
    setAccessToken: vi.fn(),
  },
  authApi: {
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    refreshToken: vi.fn(),
  },
}));

// Mock the query-client module
vi.mock('@/lib/query-client', () => ({
  clearQueryCache: vi.fn(),
}));

// Import mocked modules for assertions
import { api, authApi } from '@/lib/api';
import { clearQueryCache } from '@/lib/query-client';

const mockUser = {
  id: 'user-1',
  email: 'test@example.com',
  role: 'player' as const,
  preferredLanguage: 'de',
  createdAt: '2024-01-01T00:00:00Z',
};

describe('useAuthStore', () => {
  beforeEach(() => {
    // Reset the store to initial state
    useAuthStore.setState({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      isLoading: false,
      isHydrated: false,
      error: null,
      _refreshInProgress: null,
    });
    vi.clearAllMocks();
  });

  describe('initial state', () => {
    it('has correct initial values', () => {
      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.accessToken).toBeNull();
      expect(state.isAuthenticated).toBe(false);
      expect(state.isLoading).toBe(false);
      expect(state.error).toBeNull();
    });
  });

  describe('login', () => {
    it('sets loading state while logging in', async () => {
      vi.mocked(authApi.login).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      const loginPromise = useAuthStore.getState().login('test@example.com', 'password');

      expect(useAuthStore.getState().isLoading).toBe(true);
      expect(useAuthStore.getState().error).toBeNull();

      // Clean up
      vi.mocked(authApi.login).mockResolvedValue({ status: 200 });
    });

    it('sets user and token on successful login', async () => {
      vi.mocked(authApi.login).mockResolvedValue({
        data: { user: mockUser, accessToken: 'test-token' },
        status: 200,
      });

      const result = await useAuthStore.getState().login('test@example.com', 'password');

      expect(result).toBe(true);
      expect(useAuthStore.getState().user).toEqual(mockUser);
      expect(useAuthStore.getState().accessToken).toBe('test-token');
      expect(useAuthStore.getState().isAuthenticated).toBe(true);
      expect(useAuthStore.getState().isLoading).toBe(false);
      expect(useAuthStore.getState().error).toBeNull();
      expect(api.setAccessToken).toHaveBeenCalledWith('test-token');
    });

    it('sets error on failed login', async () => {
      vi.mocked(authApi.login).mockResolvedValue({
        error: { message: 'Invalid credentials' },
        status: 401,
      });

      const result = await useAuthStore.getState().login('test@example.com', 'wrong');

      expect(result).toBe(false);
      expect(useAuthStore.getState().user).toBeNull();
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
      expect(useAuthStore.getState().isLoading).toBe(false);
      expect(useAuthStore.getState().error).toBe('Invalid credentials');
    });

    it('clears error before new login attempt', async () => {
      useAuthStore.setState({ error: 'Previous error' });

      vi.mocked(authApi.login).mockResolvedValue({
        data: { user: mockUser, accessToken: 'token' },
        status: 200,
      });

      await useAuthStore.getState().login('test@example.com', 'password');

      expect(useAuthStore.getState().error).toBeNull();
    });

    it('returns false when response has no data or error', async () => {
      vi.mocked(authApi.login).mockResolvedValue({ status: 200 });

      const result = await useAuthStore.getState().login('test@example.com', 'password');

      expect(result).toBe(false);
      expect(useAuthStore.getState().isLoading).toBe(false);
    });
  });

  describe('register', () => {
    const registerData = {
      email: 'new@example.com',
      password: 'password123',
      firstName: 'Test',
      lastName: 'User',
    };

    it('sets loading state while registering', async () => {
      vi.mocked(authApi.register).mockImplementation(
        () => new Promise(() => {})
      );

      useAuthStore.getState().register(registerData);

      expect(useAuthStore.getState().isLoading).toBe(true);
    });

    it('returns true on successful registration', async () => {
      vi.mocked(authApi.register).mockResolvedValue({
        data: { message: 'Registration successful' },
        status: 201,
      });

      const result = await useAuthStore.getState().register(registerData);

      expect(result).toBe(true);
      expect(useAuthStore.getState().isLoading).toBe(false);
      expect(useAuthStore.getState().error).toBeNull();
    });

    it('sets error on failed registration', async () => {
      vi.mocked(authApi.register).mockResolvedValue({
        error: { message: 'Email already exists' },
        status: 409,
      });

      const result = await useAuthStore.getState().register(registerData);

      expect(result).toBe(false);
      expect(useAuthStore.getState().isLoading).toBe(false);
      expect(useAuthStore.getState().error).toBe('Email already exists');
    });
  });

  describe('logout', () => {
    beforeEach(() => {
      useAuthStore.setState({
        user: mockUser,
        accessToken: 'test-token',
        isAuthenticated: true,
      });
    });

    it('clears user state on logout', async () => {
      vi.mocked(authApi.logout).mockResolvedValue({ status: 200 });

      await useAuthStore.getState().logout();

      expect(useAuthStore.getState().user).toBeNull();
      expect(useAuthStore.getState().accessToken).toBeNull();
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
      expect(useAuthStore.getState().error).toBeNull();
    });

    it('clears access token from API client', async () => {
      vi.mocked(authApi.logout).mockResolvedValue({ status: 200 });

      await useAuthStore.getState().logout();

      expect(api.setAccessToken).toHaveBeenCalledWith(null);
    });

    it('clears query cache on logout', async () => {
      vi.mocked(authApi.logout).mockResolvedValue({ status: 200 });

      await useAuthStore.getState().logout();

      expect(clearQueryCache).toHaveBeenCalled();
    });
  });

  describe('refreshToken', () => {
    it('updates access token on successful refresh', async () => {
      vi.mocked(authApi.refreshToken).mockResolvedValue({
        data: { accessToken: 'new-token' },
        status: 200,
      });

      const result = await useAuthStore.getState().refreshToken();

      expect(result).toBe(true);
      expect(useAuthStore.getState().accessToken).toBe('new-token');
      expect(api.setAccessToken).toHaveBeenCalledWith('new-token');
    });

    it('logs out user on failed refresh', async () => {
      useAuthStore.setState({
        user: mockUser,
        accessToken: 'old-token',
        isAuthenticated: true,
      });

      vi.mocked(authApi.refreshToken).mockResolvedValue({
        error: { message: 'Refresh token expired' },
        status: 401,
      });
      vi.mocked(authApi.logout).mockResolvedValue({ status: 200 });

      const result = await useAuthStore.getState().refreshToken();

      expect(result).toBe(false);
      // logout() is called which clears the state
      expect(authApi.logout).toHaveBeenCalled();
    });

    it('prevents concurrent refresh requests', async () => {
      let resolveRefresh: (value: unknown) => void;
      vi.mocked(authApi.refreshToken).mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveRefresh = resolve;
          })
      );

      // Start first refresh
      const promise1 = useAuthStore.getState().refreshToken();
      // Start second refresh while first is in progress
      const promise2 = useAuthStore.getState().refreshToken();

      // Only one API call should be made (the key assertion)
      expect(authApi.refreshToken).toHaveBeenCalledTimes(1);

      // Resolve the refresh
      resolveRefresh!({ data: { accessToken: 'new-token' }, status: 200 });

      const [result1, result2] = await Promise.all([promise1, promise2]);
      expect(result1).toBe(true);
      expect(result2).toBe(true);
    });

    it('returns false when response has no data', async () => {
      vi.mocked(authApi.refreshToken).mockResolvedValue({ status: 200 });

      const result = await useAuthStore.getState().refreshToken();

      expect(result).toBe(false);
    });
  });

  describe('clearError', () => {
    it('clears the error state', () => {
      useAuthStore.setState({ error: 'Some error' });

      useAuthStore.getState().clearError();

      expect(useAuthStore.getState().error).toBeNull();
    });
  });

  describe('setUser', () => {
    it('sets the user', () => {
      useAuthStore.getState().setUser(mockUser);

      expect(useAuthStore.getState().user).toEqual(mockUser);
    });

    it('can set user to null', () => {
      useAuthStore.setState({ user: mockUser });

      useAuthStore.getState().setUser(null);

      expect(useAuthStore.getState().user).toBeNull();
    });
  });

  describe('setHydrated', () => {
    it('sets hydrated to true', () => {
      expect(useAuthStore.getState().isHydrated).toBe(false);

      useAuthStore.getState().setHydrated();

      expect(useAuthStore.getState().isHydrated).toBe(true);
    });
  });
});
