import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { api, authApi, User } from '@/lib/api';
import { clearQueryCache } from '@/lib/query-client';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  isHydrated: boolean;
  error: string | null;
  _refreshInProgress: Promise<boolean> | null;

  // Actions
  login: (email: string, password: string) => Promise<boolean>;
  register: (data: {
    email: string;
    password: string;
    firstName: string;
    lastName: string;
  }) => Promise<boolean>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<boolean>;
  clearError: () => void;
  setUser: (user: User | null) => void;
  setHydrated: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      isLoading: false,
      isHydrated: false,
      error: null,
      _refreshInProgress: null,

      login: async (email: string, password: string) => {
        set({ isLoading: true, error: null });

        const response = await authApi.login(email, password);

        if (response.error) {
          set({
            isLoading: false,
            error: response.error.message,
            isAuthenticated: false,
          });
          return false;
        }

        if (response.data) {
          const { user, accessToken } = response.data;
          api.setAccessToken(accessToken);
          set({
            user,
            accessToken,
            isAuthenticated: true,
            isLoading: false,
            error: null,
          });
          return true;
        }

        set({ isLoading: false });
        return false;
      },

      register: async (data) => {
        set({ isLoading: true, error: null });

        const response = await authApi.register(data);

        if (response.error) {
          set({
            isLoading: false,
            error: response.error.message,
          });
          return false;
        }

        set({ isLoading: false, error: null });
        return true;
      },

      logout: async () => {
        await authApi.logout();
        api.setAccessToken(null);
        // Clear React Query cache to remove any user-specific data
        clearQueryCache();
        set({
          user: null,
          accessToken: null,
          isAuthenticated: false,
          error: null,
        });
      },

      refreshToken: async () => {
        // Prevent concurrent refresh requests
        const existingRefresh = get()._refreshInProgress;
        if (existingRefresh) {
          return existingRefresh;
        }

        const refreshPromise = (async () => {
          const response = await authApi.refreshToken();

          if (response.error) {
            // Token refresh failed, logout user
            set({ _refreshInProgress: null });
            get().logout();
            return false;
          }

          if (response.data) {
            const { accessToken } = response.data;
            api.setAccessToken(accessToken);
            set({ accessToken, _refreshInProgress: null });
            return true;
          }

          set({ _refreshInProgress: null });
          return false;
        })();

        set({ _refreshInProgress: refreshPromise });
        return refreshPromise;
      },

      clearError: () => set({ error: null }),

      setUser: (user) => set({ user }),

      setHydrated: () => set({ isHydrated: true }),
    }),
    {
      name: 'openpairing-auth',
      partialize: (state) => ({
        user: state.user,
        accessToken: state.accessToken,
        isAuthenticated: state.isAuthenticated,
      }),
      onRehydrateStorage: () => (state) => {
        // Restore access token to API client after rehydration
        if (state?.accessToken) {
          api.setAccessToken(state.accessToken);
        }
        // Mark hydration as complete
        state?.setHydrated();
      },
    }
  )
);
