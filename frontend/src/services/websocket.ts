import type { ChatMessage, WebSocketMessage } from '../types';

type MessageHandler = (message: ChatMessage) => void;
type TypingHandler = (sessionId: number, isTyping: boolean) => void;
type WhatsAppQRHandler = (qrImage: string) => void;
type WhatsAppStatusHandler = (status: Record<string, string>) => void;
type SessionUpdateHandler = (payload: Record<string, unknown>) => void;

class WebSocketService {
  private socket: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private messageHandlers: Set<MessageHandler> = new Set();
  private typingHandlers: Set<TypingHandler> = new Set();
  private whatsappQRHandlers: Set<WhatsAppQRHandler> = new Set();
  private whatsappStatusHandlers: Set<WhatsAppStatusHandler> = new Set();
  private sessionUpdateHandlers: Set<SessionUpdateHandler> = new Set();
  private subscribedSessions: Set<number> = new Set();

  connect(userId?: number): void {
    const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8080';
    const url = userId ? `${wsUrl}/ws?user_id=${userId}` : `${wsUrl}/ws`;

    this.socket = new WebSocket(url);

    this.socket.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;
      // Re-subscribe to sessions
      this.subscribedSessions.forEach((sessionId) => {
        this.subscribe(sessionId);
      });
    };

    this.socket.onclose = () => {
      console.log('WebSocket disconnected');
      this.attemptReconnect(userId);
    };

    this.socket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.socket.onmessage = (event) => {
      try {
        const data: WebSocketMessage = JSON.parse(event.data);
        this.handleMessage(data);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };
  }

  private attemptReconnect(userId?: number): void {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
      console.log(`Attempting to reconnect in ${delay}ms...`);
      setTimeout(() => this.connect(userId), delay);
    }
  }

  private handleMessage(data: WebSocketMessage): void {
    switch (data.type) {
      case 'message':
        if (data.payload) {
          this.messageHandlers.forEach((handler) => {
            handler(data.payload as ChatMessage);
          });
        }
        break;
      case 'typing':
        if (data.session_id !== undefined) {
          this.typingHandlers.forEach((handler) => {
            handler(data.session_id as number, true);
          });
        }
        break;
      case 'read':
        break;
      case 'whatsapp_qr':
        if (data.payload) {
          const payload = data.payload as Record<string, string>;
          this.whatsappQRHandlers.forEach((handler) => {
            handler(payload.qr_image);
          });
        }
        break;
      case 'whatsapp_status':
        if (data.payload) {
          this.whatsappStatusHandlers.forEach((handler) => {
            handler(data.payload as Record<string, string>);
          });
        }
        break;
      case 'session_update':
        if (data.payload) {
          this.sessionUpdateHandlers.forEach((handler) => {
            handler(data.payload as Record<string, unknown>);
          });
        }
        break;
    }
  }

  disconnect(): void {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
    this.subscribedSessions.clear();
  }

  subscribe(sessionId: number): void {
    this.subscribedSessions.add(sessionId);
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.send({
        type: 'subscribe',
        session_id: sessionId,
      });
    }
  }

  unsubscribe(sessionId: number): void {
    this.subscribedSessions.delete(sessionId);
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.send({
        type: 'unsubscribe',
        session_id: sessionId,
      });
    }
  }

  sendTyping(sessionId: number): void {
    this.send({
      type: 'typing',
      session_id: sessionId,
    });
  }

  sendRead(sessionId: number): void {
    this.send({
      type: 'read',
      session_id: sessionId,
    });
  }

  private send(message: WebSocketMessage): void {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    }
  }

  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => {
      this.messageHandlers.delete(handler);
    };
  }

  onTyping(handler: TypingHandler): () => void {
    this.typingHandlers.add(handler);
    return () => {
      this.typingHandlers.delete(handler);
    };
  }

  onWhatsAppQR(handler: WhatsAppQRHandler): () => void {
    this.whatsappQRHandlers.add(handler);
    return () => {
      this.whatsappQRHandlers.delete(handler);
    };
  }

  onWhatsAppStatus(handler: WhatsAppStatusHandler): () => void {
    this.whatsappStatusHandlers.add(handler);
    return () => {
      this.whatsappStatusHandlers.delete(handler);
    };
  }

  onSessionUpdate(handler: SessionUpdateHandler): () => void {
    this.sessionUpdateHandlers.add(handler);
    return () => {
      this.sessionUpdateHandlers.delete(handler);
    };
  }

  get isConnected(): boolean {
    return this.socket?.readyState === WebSocket.OPEN;
  }
}

export const wsService = new WebSocketService();
export default wsService;

