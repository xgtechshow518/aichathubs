// User type
export interface User {
  id: number;
  email: string;
  name: string;
  avatar_url: string;
  provider: string;
  provider_id: string;
  subscription_plan: string;
  subscription_status: string;
  trial_ends_at?: string;
  max_devices: number;
  created_at: string;
  updated_at: string;
}

// Chat session type
export interface ChatSession {
  id: number;
  customer_name: string;
  customer_avatar: string;
  customer_phone: string;
  platform: 'whatsapp' | 'facebook' | 'telegram' | 'instagram' | 'line' | 'email';
  unread_count: number;
  last_message: string;
  last_message_at: string;
  last_sender_type?: 'customer' | 'agent' | 'bot';
  device_id?: number;
  status: 'active' | 'closed';
  assigned_to_id?: number;
  assigned_to?: User;
  created_at: string;
  updated_at: string;
}

// Chat message type
export interface ChatMessage {
  id: number;
  session_id: number;
  sender_type: 'customer' | 'agent' | 'bot';
  sender_id?: number;
  content: string;
  message_type: 'text' | 'image' | 'file';
  media_url?: string;
  is_read: boolean;
  created_at: string;
}

// API response types
export interface ChatListResponse {
  chats: ChatSession[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface MessagesResponse {
  messages: ChatMessage[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface ChatStats {
  total_chats: number;
  unread_chats: number;
  active_chats: number;
}

// WhatsApp types
export interface WhatsAppDevice {
  id: number;
  user_id: number;
  jid: string;
  push_name: string;
  phone: string;
  status: 'connected' | 'disconnected' | 'scanning';
  connected_at?: string;
  created_at: string;
  updated_at: string;
}

export interface WhatsAppStatus {
  status: 'connected' | 'disconnected' | 'scanning' | 'timeout';
  jid?: string;
  push_name?: string;
  phone?: string;
}

// Knowledge Base types
export interface KnowledgeBase {
  id: number;
  user_id: number;
  gemini_store_id: string;
  gemini_store_name: string;
  auto_reply_enabled: boolean;
  system_prompt: string;
  last_synced_at?: string;
  created_at: string;
  updated_at: string;
}

export interface QAItem {
  id: number;
  user_id: number;
  question: string;
  answer: string;
  category: string;
  created_at: string;
  updated_at: string;
}

export interface ProductImage {
  id: number;
  product_id: number;
  url: string;
  is_primary: boolean;
  sort_order: number;
}

export interface Product {
  id: number;
  user_id: number;
  sku: string;
  name: string;
  description: string;
  price: number;
  currency: string;
  stock: number;
  category: string;
  tags: string;
  product_url?: string;
  checkout_url?: string;
  active: boolean;
  images?: ProductImage[];
  created_at: string;
  updated_at: string;
}

export interface LeadRow {
  id: number;
  user_id: number;
  session_id: number;
  sku: string;
  quantity: number;
  notes: string;
  customer_name?: string;
  customer_phone?: string;
  created_at: string;
}

// WebSocket message types
export interface WebSocketMessage {
  type: 'message' | 'typing' | 'read' | 'subscribe' | 'unsubscribe' | 'whatsapp_qr' | 'whatsapp_status' | 'session_update';
  session_id?: number;
  payload?: unknown;
}

