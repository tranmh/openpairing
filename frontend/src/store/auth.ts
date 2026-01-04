import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { api, authApi, User } from '@/lib/api';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

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
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

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
        set({
          user: null,
          accessToken: null,
          isAuthenticated: false,
          error: null,
        });
      },

      refreshToken: async () => {
        const response = await authApi.refreshToken();

        if (response.error) {
          // Token refresh failed, logout user
          get().logout();
          return false;
        }

        if (response.data) {
          const { accessToken } = response.data;
          api.setAccessToken(accessToken);
          set({ accessToken });
          return true;
        }

        return false;
      },

      clearError: () => set({ error: null }),

      setUser: (user) => set({ user }),
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
      },
    }
  )
);
