import axios from 'axios';
import type { AxiosInstance, InternalAxiosRequestConfig } from 'axios';
import type {
  AuthResponse, ChatListResponse, ChatMessage, ChatSession, ChatStats,
  LeadRow, MessagesResponse, Product, ProductImage, User,
} from '../types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

// Feature flags exposed by the backend so the UI can render conditionally
// (which login buttons to show, whether email verification / billing are on).
export interface PublicConfig {
  googleAuth: boolean;
  facebookAuth: boolean;
  emailVerification: boolean;
  billingEnabled: boolean;
  aiEnabled: boolean;
}

class ApiService {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: `${API_BASE_URL}/api`,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add auth token to requests (skip if Authorization already set, e.g. admin calls)
    this.client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
      if (config.headers && !config.headers.Authorization) {
        const token = localStorage.getItem('token');
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
      }
      return config;
    });

    // Handle auth and subscription errors (skip redirects for admin routes)
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        const isAdminRoute = error.config?.url?.startsWith('/admin');
        if (error.response?.status === 401 && !isAdminRoute) {
          localStorage.removeItem('token');
          window.location.href = '/login';
        } else if (error.response?.status === 401 && isAdminRoute) {
          localStorage.removeItem('admin_token');
          window.location.href = '/admin/login';
        } else if (error.response?.status === 402) {
          // Avoid redirect loop if we're already on the billing/expired page
          const alreadyOnBilling =
            window.location.pathname === '/settings' &&
            window.location.search.includes('tab=billing') &&
            window.location.search.includes('expired=true');
          if (!alreadyOnBilling) {
            window.location.href = '/settings?tab=billing&expired=true';
          }
        }
        return Promise.reject(error);
      }
    );
  }

  // Public server metadata (no auth required)
  async getPublicConfig(): Promise<PublicConfig> {
    const response = await this.client.get('/config');
    return response.data;
  }

  // Auth endpoints
  // When the server has no SMTP configured, registration auto-verifies the
  // account and returns { token, user } for an immediate login. Otherwise it
  // returns { message } and the user must verify via emailed code.
  async register(
    email: string,
    password: string,
    name: string,
  ): Promise<{ message?: string; token?: string; user?: User }> {
    const response = await this.client.post('/auth/register', { email, password, name });
    return response.data;
  }

  async verifyEmail(email: string, code: string): Promise<AuthResponse> {
    const response = await this.client.post('/auth/verify-email', { email, code });
    return response.data;
  }

  async resendCode(email: string): Promise<{ message: string }> {
    const response = await this.client.post('/auth/resend-code', { email });
    return response.data;
  }

  async emailLogin(email: string, password: string): Promise<AuthResponse> {
    const response = await this.client.post('/auth/login', { email, password });
    return response.data;
  }

  async getGoogleAuthUrl(): Promise<{ url: string }> {
    const response = await this.client.get('/auth/google');
    return response.data;
  }

  async googleCallback(code: string, redirectUri: string): Promise<AuthResponse> {
    const response = await this.client.post('/auth/google/callback', {
      code,
      redirect_uri: redirectUri,
    });
    return response.data;
  }

  async getFacebookAuthUrl(): Promise<{ url: string }> {
    const response = await this.client.get('/auth/facebook');
    return response.data;
  }

  async facebookCallback(code: string, redirectUri: string): Promise<AuthResponse> {
    const response = await this.client.post('/auth/facebook/callback', {
      code,
      redirect_uri: redirectUri,
    });
    return response.data;
  }

  async getMe(): Promise<User> {
    const response = await this.client.get('/auth/me');
    return response.data;
  }

  // Chat endpoints
  async getChats(params?: {
    page?: number;
    page_size?: number;
    filter?: string;
    platform?: string;
  }): Promise<ChatListResponse> {
    const response = await this.client.get('/chats', { params });
    return response.data;
  }

  async getChat(id: number): Promise<ChatSession> {
    const response = await this.client.get(`/chats/${id}`);
    return response.data;
  }

  async getChatMessages(
    id: number,
    params?: { page?: number; page_size?: number }
  ): Promise<MessagesResponse> {
    const response = await this.client.get(`/chats/${id}/messages`, { params });
    return response.data;
  }

  async sendMessage(
    sessionId: number,
    content: string,
    messageType: string = 'text',
    mediaUrl?: string
  ): Promise<ChatMessage> {
    const response = await this.client.post(`/chats/${sessionId}/messages`, {
      content,
      message_type: messageType,
      media_url: mediaUrl,
    });
    return response.data;
  }

  async markAsRead(sessionId: number): Promise<void> {
    await this.client.post(`/chats/${sessionId}/read`);
  }

  async sendImageMessage(sessionId: number, file: File): Promise<ChatMessage> {
    const formData = new FormData();
    formData.append('file', file);
    const response = await this.client.post(`/chats/${sessionId}/upload`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return response.data;
  }

  async acceptConversation(sessionId: number): Promise<{ status: string }> {
    const response = await this.client.post(`/chats/${sessionId}/accept`);
    return response.data;
  }

  async blacklistCustomer(sessionId: number): Promise<{ status: string }> {
    const response = await this.client.post(`/chats/${sessionId}/blacklist`);
    return response.data;
  }

  async getBlacklisted(): Promise<{ sessions: import('../types').ChatSession[]; total: number }> {
    const response = await this.client.get('/chats/blacklisted');
    return response.data;
  }

  async unblacklistCustomer(sessionId: number): Promise<{ status: string }> {
    const response = await this.client.post(`/chats/${sessionId}/unblacklist`);
    return response.data;
  }

  async getChatStats(): Promise<ChatStats> {
    const response = await this.client.get('/chats/stats');
    return response.data;
  }

  // Tag endpoints
  async listTags(): Promise<{ id: number; name: string; color: string }[]> {
    const response = await this.client.get('/tags');
    return response.data;
  }

  async createTag(name: string, color: string = 'blue'): Promise<{ id: number; name: string; color: string }> {
    const response = await this.client.post('/tags', { name, color });
    return response.data;
  }

  async deleteTag(id: number): Promise<void> {
    await this.client.delete(`/tags/${id}`);
  }

  async getSessionTags(sessionId: number): Promise<{ id: number; name: string; color: string }[]> {
    const response = await this.client.get(`/chats/${sessionId}/tags`);
    return response.data;
  }

  async addTagToSession(sessionId: number, tagId: number): Promise<void> {
    await this.client.post(`/chats/${sessionId}/tags`, { tag_id: tagId });
  }

  async removeTagFromSession(sessionId: number, tagId: number): Promise<void> {
    await this.client.delete(`/chats/${sessionId}/tags/${tagId}`);
  }

  async getReportData(): Promise<Record<string, unknown>> {
    const response = await this.client.get('/chats/reports');
    return response.data;
  }

  // WhatsApp endpoints
  async connectWhatsApp(): Promise<{ message: string; status: string }> {
    const response = await this.client.post('/whatsapp/connect');
    return response.data;
  }

  async disconnectWhatsApp(): Promise<{ status: string }> {
    const response = await this.client.post('/whatsapp/disconnect');
    return response.data;
  }

  async disconnectWhatsAppDevice(deviceId: number): Promise<{ status: string }> {
    const response = await this.client.post(`/whatsapp/disconnect/${deviceId}`);
    return response.data;
  }

  async getWhatsAppStatus(): Promise<Record<string, string>> {
    const response = await this.client.get('/whatsapp/status');
    return response.data;
  }

  async getWhatsAppDevices(): Promise<{ devices: import('../types').WhatsAppDevice[] }> {
    const response = await this.client.get('/whatsapp/devices');
    return response.data;
  }

  // Knowledge Base / Q&A endpoints
  async getKnowledgeBase(): Promise<import('../types').KnowledgeBase> {
    const response = await this.client.get('/knowledge');
    return response.data;
  }

  async updateKnowledgeBase(data: { auto_reply_enabled?: boolean; system_prompt?: string }): Promise<import('../types').KnowledgeBase> {
    const response = await this.client.put('/knowledge', data);
    return response.data;
  }

  async getQAItems(): Promise<{ items: import('../types').QAItem[]; total: number }> {
    const response = await this.client.get('/knowledge/qa');
    return response.data;
  }

  async createQAItem(data: { question: string; answer: string; category?: string }): Promise<import('../types').QAItem> {
    const response = await this.client.post('/knowledge/qa', data);
    return response.data;
  }

  async updateQAItem(id: number, data: { question?: string; answer?: string; category?: string }): Promise<import('../types').QAItem> {
    const response = await this.client.put(`/knowledge/qa/${id}`, data);
    return response.data;
  }

  async deleteQAItem(id: number): Promise<void> {
    await this.client.delete(`/knowledge/qa/${id}`);
  }

  async deleteAllQAItems(): Promise<{ status: string; deleted: number }> {
    const response = await this.client.delete('/knowledge/qa');
    return response.data;
  }

  async downloadTemplate(): Promise<Blob> {
    const response = await this.client.get('/knowledge/template', { responseType: 'blob' });
    return response.data;
  }

  async syncKnowledgeBase(): Promise<{ status: string }> {
    const response = await this.client.post('/knowledge/sync');
    return response.data;
  }

  async uploadKnowledgeFile(file: File): Promise<{ status: string; filename: string; count?: number }> {
    const formData = new FormData();
    formData.append('file', file);
    const response = await this.client.post('/knowledge/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return response.data;
  }

  async testKnowledgeQuery(question: string): Promise<{ question: string; answer: string }> {
    const response = await this.client.post('/knowledge/test', { question });
    return response.data;
  }

  async testChat(
    message: string,
    history: { role: 'user' | 'bot'; content: string }[],
  ): Promise<{ reply: string }> {
    const response = await this.client.post('/knowledge/test/chat', { message, history });
    return response.data;
  }

  // Products catalog
  async listProducts(params?: { search?: string; category?: string; active?: boolean }): Promise<{
    products: Product[];
    total: number;
  }> {
    const response = await this.client.get('/products', { params });
    return response.data;
  }

  async createProduct(data: Partial<Product>): Promise<Product> {
    const response = await this.client.post('/products', data);
    return response.data;
  }

  async updateProduct(id: number, data: Partial<Product>): Promise<Product> {
    const response = await this.client.put(`/products/${id}`, data);
    return response.data;
  }

  async deleteProduct(id: number): Promise<void> {
    await this.client.delete(`/products/${id}`);
  }

  async addProductImage(productId: number, url: string, isPrimary = false): Promise<ProductImage> {
    const response = await this.client.post(`/products/${productId}/images`, { url, is_primary: isPrimary });
    return response.data;
  }

  async deleteProductImage(productId: number, imageId: number): Promise<void> {
    await this.client.delete(`/products/${productId}/images/${imageId}`);
  }

  async importProducts(file: File): Promise<{ status: string; created: number; total: number }> {
    const formData = new FormData();
    formData.append('file', file);
    const response = await this.client.post('/products/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return response.data;
  }

  async downloadProductTemplate(): Promise<Blob> {
    const response = await this.client.get('/products/template', { responseType: 'blob' });
    return response.data;
  }

  async listLeads(): Promise<{ leads: LeadRow[]; total: number }> {
    const response = await this.client.get('/leads');
    return response.data;
  }

  // Admin endpoints
  private getAdminHeaders() {
    const token = localStorage.getItem('admin_token');
    return token ? { Authorization: `Bearer ${token}` } : {};
  }

  async adminLogin(username: string, password: string): Promise<{ token: string }> {
    const response = await this.client.post('/admin/login', { username, password });
    return response.data;
  }

  async getAdminDashboard(): Promise<Record<string, unknown>> {
    const response = await this.client.get('/admin/dashboard', { headers: this.getAdminHeaders() });
    return response.data;
  }

  async getAdminUsers(params?: {
    page?: number;
    page_size?: number;
    search?: string;
    plan?: string;
    status?: string;
    provider?: string;
  }): Promise<Record<string, unknown>> {
    const response = await this.client.get('/admin/users', { params, headers: this.getAdminHeaders() });
    return response.data;
  }

  async getAdminUser(id: number): Promise<Record<string, unknown>> {
    const response = await this.client.get(`/admin/users/${id}`, { headers: this.getAdminHeaders() });
    return response.data;
  }

  async deleteAdminUser(id: number): Promise<{ message: string }> {
    const response = await this.client.delete(`/admin/users/${id}`, { headers: this.getAdminHeaders() });
    return response.data;
  }

  async updateAdminUser(id: number, data: {
    name?: string;
    subscription_plan?: string;
    subscription_status?: string;
    max_devices?: number;
    trial_ends_at?: string | null;
    email_verified?: boolean;
  }): Promise<Record<string, unknown>> {
    const response = await this.client.patch(`/admin/users/${id}`, data, { headers: this.getAdminHeaders() });
    return response.data;
  }

  async resetAdminUserPassword(id: number, password?: string): Promise<{
    message: string;
    generated: boolean;
    password?: string;
  }> {
    const response = await this.client.post(`/admin/users/${id}/reset-password`,
      password ? { password } : {},
      { headers: this.getAdminHeaders() });
    return response.data;
  }

  async suspendAdminUser(id: number, reason?: string): Promise<Record<string, unknown>> {
    const response = await this.client.post(`/admin/users/${id}/suspend`,
      { reason: reason || '' },
      { headers: this.getAdminHeaders() });
    return response.data;
  }

  async unsuspendAdminUser(id: number): Promise<Record<string, unknown>> {
    const response = await this.client.post(`/admin/users/${id}/unsuspend`, {},
      { headers: this.getAdminHeaders() });
    return response.data;
  }

  async getAdminUserKnowledge(id: number): Promise<{
    id?: number;
    user_id: number;
    auto_reply_enabled: boolean;
    system_prompt: string;
    last_synced_at?: string;
    qa_count?: number;
  }> {
    const response = await this.client.get(`/admin/users/${id}/knowledge`, { headers: this.getAdminHeaders() });
    return response.data;
  }

  async updateAdminUserKnowledge(id: number, data: {
    auto_reply_enabled?: boolean;
    system_prompt?: string;
  }): Promise<Record<string, unknown>> {
    const response = await this.client.put(`/admin/users/${id}/knowledge`, data, { headers: this.getAdminHeaders() });
    return response.data;
  }

  async getAdminDeviceChats(deviceId: number, params?: {
    page?: number;
    page_size?: number;
    search?: string;
  }): Promise<Record<string, unknown>> {
    const response = await this.client.get(`/admin/devices/${deviceId}/chats`, {
      params,
      headers: this.getAdminHeaders(),
    });
    return response.data;
  }

  async getAdminChatMessages(sessionId: number, params?: {
    page?: number;
    page_size?: number;
  }): Promise<Record<string, unknown>> {
    const response = await this.client.get(`/admin/chats/${sessionId}/messages`, {
      params,
      headers: this.getAdminHeaders(),
    });
    return response.data;
  }

  async getAdminDevices(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    search?: string;
  }): Promise<Record<string, unknown>> {
    const response = await this.client.get('/admin/devices', { params, headers: this.getAdminHeaders() });
    return response.data;
  }

  async getAdminSubscriptions(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    plan?: string;
  }): Promise<Record<string, unknown>> {
    const response = await this.client.get('/admin/subscriptions', { params, headers: this.getAdminHeaders() });
    return response.data;
  }

  async getAdminChatAnalytics(): Promise<Record<string, unknown>> {
    const response = await this.client.get('/admin/chats/analytics', { headers: this.getAdminHeaders() });
    return response.data;
  }

  async getAdminSystemInfo(): Promise<Record<string, unknown>> {
    const response = await this.client.get('/admin/system', { headers: this.getAdminHeaders() });
    return response.data;
  }

  async getAdminBotPrompt(): Promise<{
    value: string;
    default: string;
    is_custom: boolean;
    updated_at?: string;
    updated_by?: string;
  }> {
    const response = await this.client.get('/admin/bot-prompt', { headers: this.getAdminHeaders() });
    return response.data;
  }

  async updateAdminBotPrompt(value: string): Promise<{
    value: string;
    default: string;
    is_custom: boolean;
    updated_at?: string;
    updated_by?: string;
  }> {
    const response = await this.client.put('/admin/bot-prompt', { value }, { headers: this.getAdminHeaders() });
    return response.data;
  }

  async getAdminAIReplyDelay(): Promise<{
    min_seconds: number;
    max_seconds: number;
    default_min_seconds: number;
    default_max_seconds: number;
    is_custom: boolean;
    updated_at?: string;
    updated_by?: string;
  }> {
    const response = await this.client.get('/admin/ai-reply-delay', { headers: this.getAdminHeaders() });
    return response.data;
  }

  async updateAdminAIReplyDelay(minSeconds: number, maxSeconds: number): Promise<{
    min_seconds: number;
    max_seconds: number;
    default_min_seconds: number;
    default_max_seconds: number;
    is_custom: boolean;
  }> {
    const response = await this.client.put(
      '/admin/ai-reply-delay',
      { min_seconds: minSeconds, max_seconds: maxSeconds },
      { headers: this.getAdminHeaders() },
    );
    return response.data;
  }

  async resetAdminAIReplyDelay(): Promise<{
    min_seconds: number;
    max_seconds: number;
    default_min_seconds: number;
    default_max_seconds: number;
    is_custom: boolean;
  }> {
    const response = await this.client.post(
      '/admin/ai-reply-delay/reset',
      {},
      { headers: this.getAdminHeaders() },
    );
    return response.data;
  }

  // Billing endpoints
  async createCheckout(quantity: number = 1): Promise<{ checkout_url: string; session_id: string }> {
    const response = await this.client.post('/billing/checkout', { quantity });
    return response.data;
  }

  async createPortal(): Promise<{ portal_url: string }> {
    const response = await this.client.post('/billing/portal');
    return response.data;
  }

  async updateQuantity(quantity: number): Promise<{ max_devices: number; status: string; message: string }> {
    const response = await this.client.put('/billing/quantity', { quantity });
    return response.data;
  }

  async getSubscription(): Promise<Record<string, unknown>> {
    const response = await this.client.get('/billing/subscription');
    return response.data;
  }

  async syncSubscription(): Promise<{ synced: boolean; max_devices: number; plan?: string; status?: string }> {
    const response = await this.client.post('/billing/sync');
    return response.data;
  }
}

export const api = new ApiService();
export default api;

