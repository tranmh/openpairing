import { vi } from 'vitest';

/**
 * Mock API client for testing.
 * Replace actual API calls with controlled responses.
 */
export const mockApi = {
  setAccessToken: vi.fn(),
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
};

/**
 * Mock authApi for testing authentication flows.
 */
export const mockAuthApi = {
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
  refreshToken: vi.fn(),
  getMe: vi.fn(),
};

/**
 * Creates a successful API response.
 */
export function createApiResponse<T>(data: T, status = 200) {
  return {
    data,
    status,
    error: undefined,
  };
}

/**
 * Creates an error API response.
 */
export function createApiError(
  message: string,
  status = 400,
  code?: string,
  details?: Record<string, string[]>
) {
  return {
    data: undefined,
    status,
    error: {
      message,
      code,
      details,
    },
  };
}

/**
 * Resets all API mocks between tests.
 */
export function resetApiMocks() {
  Object.values(mockApi).forEach((mock) => mock.mockReset());
  Object.values(mockAuthApi).forEach((mock) => mock.mockReset());
}

/**
 * Setup mock for a successful login response.
 */
export function mockSuccessfulLogin(user = mockUser, accessToken = 'test-token') {
  mockAuthApi.login.mockResolvedValue(
    createApiResponse({ user, accessToken })
  );
}

/**
 * Setup mock for a failed login response.
 */
export function mockFailedLogin(message = 'Invalid credentials') {
  mockAuthApi.login.mockResolvedValue(createApiError(message, 401));
}

// Default mock user for testing
export const mockUser = {
  id: 'user-1',
  email: 'test@example.com',
  role: 'player' as const,
  preferredLanguage: 'de',
  createdAt: '2024-01-01T00:00:00Z',
};

// Mock admin user
export const mockAdminUser = {
  id: 'admin-1',
  email: 'admin@example.com',
  role: 'admin' as const,
  preferredLanguage: 'de',
  createdAt: '2024-01-01T00:00:00Z',
};

// Mock arbiter user
export const mockArbiterUser = {
  id: 'arbiter-1',
  email: 'arbiter@example.com',
  role: 'arbiter' as const,
  preferredLanguage: 'de',
  createdAt: '2024-01-01T00:00:00Z',
};
