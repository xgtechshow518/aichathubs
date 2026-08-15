import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import api from '../services/api';

interface AdminState {
  adminToken: string | null;
  isAdminAuthenticated: boolean;
  isLoading: boolean;
  adminLogin: (username: string, password: string) => Promise<void>;
  adminLogout: () => void;
  checkAdmin: () => void;
}

export const useAdminStore = create<AdminState>()(
  persist(
    (set) => ({
      adminToken: null,
      isAdminAuthenticated: false,
      isLoading: true,

      adminLogin: async (username: string, password: string) => {
        const data = await api.adminLogin(username, password);
        localStorage.setItem('admin_token', data.token);
        set({ adminToken: data.token, isAdminAuthenticated: true, isLoading: false });
      },

      adminLogout: () => {
        localStorage.removeItem('admin_token');
        set({ adminToken: null, isAdminAuthenticated: false, isLoading: false });
      },

      checkAdmin: () => {
        const token = localStorage.getItem('admin_token');
        if (token) {
          set({ adminToken: token, isAdminAuthenticated: true, isLoading: false });
        } else {
          set({ isLoading: false, isAdminAuthenticated: false });
        }
      },
    }),
    {
      name: 'admin-storage',
      partialize: (state) => ({ adminToken: state.adminToken }),
    }
  )
);
