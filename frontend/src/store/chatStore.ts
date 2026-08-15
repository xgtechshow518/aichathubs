import { create } from 'zustand';
import type { ChatMessage, ChatSession, ChatStats } from '../types';
import api from '../services/api';

interface ChatState {
  sessions: ChatSession[];
  currentSession: ChatSession | null;
  messages: ChatMessage[];
  stats: ChatStats | null;
  isLoading: boolean;
  filter: string;
  
  // Actions
  fetchSessions: (filter?: string) => Promise<void>;
  fetchSession: (id: number) => Promise<void>;
  fetchMessages: (sessionId: number) => Promise<void>;
  fetchStats: () => Promise<void>;
  sendMessage: (sessionId: number, content: string) => Promise<void>;
  markAsRead: (sessionId: number) => Promise<void>;
  setFilter: (filter: string) => void;
  addMessage: (message: ChatMessage) => void;
  updateSessionFromWS: (payload: Record<string, unknown>) => void;
  setCurrentSession: (session: ChatSession | null) => void;
}

export const useChatStore = create<ChatState>((set, get) => ({
  sessions: [],
  currentSession: null,
  messages: [],
  stats: null,
  isLoading: false,
  filter: 'all',

  fetchSessions: async (filter?: string) => {
    set({ isLoading: true });
    try {
      const response = await api.getChats({ filter: filter || get().filter });
      set({ sessions: response.chats, isLoading: false });
    } catch (error) {
      console.error('Failed to fetch sessions:', error);
      set({ isLoading: false });
    }
  },

  fetchSession: async (id: number) => {
    try {
      const session = await api.getChat(id);
      set({ currentSession: session });
    } catch (error) {
      console.error('Failed to fetch session:', error);
    }
  },

  fetchMessages: async (sessionId: number) => {
    set({ isLoading: true });
    try {
      const response = await api.getChatMessages(sessionId, { page_size: 100 });
      set({ messages: response.messages || [], isLoading: false });
    } catch (error) {
      console.error('Failed to fetch messages:', error);
      set({ isLoading: false });
    }
  },

  fetchStats: async () => {
    try {
      const stats = await api.getChatStats();
      set({ stats });
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    }
  },

  sendMessage: async (sessionId: number, content: string) => {
    try {
      await api.sendMessage(sessionId, content);
      // Message will be added via WebSocket broadcast (addMessage)
    } catch (error) {
      console.error('Failed to send message:', error);
    }
  },

  markAsRead: async (sessionId: number) => {
    try {
      await api.markAsRead(sessionId);
      set((state) => ({
        sessions: state.sessions.map((s) =>
          s.id === sessionId ? { ...s, unread_count: 0 } : s
        ),
      }));
    } catch (error) {
      console.error('Failed to mark as read:', error);
    }
  },

  setFilter: (filter: string) => {
    set({ filter });
    get().fetchSessions(filter);
  },

  addMessage: (message: ChatMessage) => {
    set((state) => {
      if (state.currentSession?.id === message.session_id) {
        // Skip if message already exists (prevents duplicates from API + WebSocket)
        if (state.messages.some((m) => m.id === message.id)) {
          return state;
        }
        return { messages: [...state.messages, message] };
      }
      return state;
    });

    // Update session's last message
    set((state) => ({
      sessions: state.sessions.map((s) =>
        s.id === message.session_id
          ? {
              ...s,
              last_message: message.content,
              last_message_at: message.created_at,
              unread_count: s.unread_count + 1,
            }
          : s
      ),
    }));
  },

  updateSessionFromWS: (payload: Record<string, unknown>) => {
    const sessionId = payload.session_id as number;
    if (!sessionId) return;

    set((state) => {
      const exists = state.sessions.some((s) => s.id === sessionId);
      if (exists) {
        return {
          sessions: state.sessions.map((s) =>
            s.id === sessionId
              ? {
                  ...s,
                  last_message: (payload.last_message as string) ?? s.last_message,
                  unread_count: (payload.unread_count as number) ?? s.unread_count,
                  customer_name: (payload.customer_name as string) ?? s.customer_name,
                }
              : s
          ),
        };
      }
      return state;
    });
  },

  setCurrentSession: (session: ChatSession | null) => {
    set({ currentSession: session, messages: [] });
  },
}));

