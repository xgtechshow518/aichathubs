import { useEffect, useRef, useState } from 'react';
import {
  Layout,
  Menu,
  List,
  Avatar,
  Input,
  Button,
  Badge,
  Typography,
  Tag,
  Spin,
  Card,
  Divider,
  Popover,
  Popconfirm,
  Space,
  Select,
  message,
} from 'antd';
import {
  SearchOutlined,
  FilterOutlined,
  MoreOutlined,
  SendOutlined,
  PictureOutlined,
  SmileOutlined,
  WhatsAppOutlined,
  FacebookOutlined,
  PhoneOutlined,
  MailOutlined,
  GlobalOutlined,
  TagOutlined,
  PlusOutlined,
  CheckCircleOutlined,
  StopOutlined,
  RobotOutlined,
  MobileOutlined,
} from '@ant-design/icons';
import { useChatStore } from '../store/chatStore';
import type { ChatSession, ChatMessage, WhatsAppDevice } from '../types';
import wsService from '../services/websocket';
import api from '../services/api';
import './ChatHistory.css';

const { Sider, Content } = Layout;
const { Text, Title } = Typography;
const { TextArea } = Input;

const viewFilters = [
  { key: 'all', label: 'All', icon: '📋' },
  { key: 'unread', label: 'Unread', icon: '📬' },
  { key: 'unreplied', label: 'Unreplied', icon: '↩️' },
  { key: 'single', label: 'Single Chat', icon: '👤' },
  { key: 'group', label: 'Group', icon: '👥' },
  { key: 'dnd', label: 'Do Not Disturb', icon: '🔕' },
];

const platformIcons: Record<string, React.ReactNode> = {
  whatsapp: <WhatsAppOutlined style={{ color: '#25D366' }} />,
  facebook: <FacebookOutlined style={{ color: '#1877F2' }} />,
};

export default function ChatHistory() {
  const {
    sessions,
    currentSession,
    messages,
    isLoading,
    filter,
    fetchSessions,
    fetchSession,
    fetchMessages,
    sendMessage,
    markAsRead,
    setFilter,
    setCurrentSession,
  } = useChatStore();

  const [messageInput, setMessageInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [sessionTags, setSessionTags] = useState<{ id: number; name: string; color: string }[]>([]);
  const [allTags, setAllTags] = useState<{ id: number; name: string; color: string }[]>([]);
  const [newTagName, setNewTagName] = useState('');
  const [devices, setDevices] = useState<WhatsAppDevice[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState<number | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchSessions();
    loadAllTags();
    loadDevices();
  }, []);

  const loadDevices = async () => {
    try {
      const { devices: devs } = await api.getWhatsAppDevices();
      setDevices((devs || []).filter((d: WhatsAppDevice) => d.status === 'connected'));
    } catch { /* ignore */ }
  };

  const loadAllTags = async () => {
    try {
      const tags = await api.listTags();
      setAllTags(tags);
    } catch { /* ignore */ }
  };

  const loadSessionTags = async (sessionId: number) => {
    try {
      const tags = await api.getSessionTags(sessionId);
      setSessionTags(tags);
    } catch { setSessionTags([]); }
  };

  const handleAddTag = async (tagId: number) => {
    if (!currentSession) return;
    try {
      await api.addTagToSession(currentSession.id, tagId);
      loadSessionTags(currentSession.id);
    } catch { message.error('Failed to add tag'); }
  };

  const handleRemoveTag = async (tagId: number) => {
    if (!currentSession) return;
    try {
      await api.removeTagFromSession(currentSession.id, tagId);
      loadSessionTags(currentSession.id);
    } catch { message.error('Failed to remove tag'); }
  };

  const handleCreateTag = async () => {
    if (!newTagName.trim()) return;
    try {
      const colors = ['blue', 'green', 'red', 'orange', 'purple', 'cyan', 'magenta', 'gold'];
      const color = colors[allTags.length % colors.length];
      await api.createTag(newTagName.trim(), color);
      setNewTagName('');
      loadAllTags();
    } catch { message.error('Failed to create tag'); }
  };

  useEffect(() => {
    if (currentSession) {
      wsService.subscribe(currentSession.id);

      return () => {
        wsService.unsubscribe(currentSession.id);
      };
    }
  }, [currentSession?.id]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSelectChat = (session: ChatSession) => {
    setCurrentSession(session);
    fetchSession(session.id);
    fetchMessages(session.id);
    markAsRead(session.id);
    loadSessionTags(session.id);
  };

  const handleSendMessage = async () => {
    if (!messageInput.trim() || !currentSession) return;

    await sendMessage(currentSession.id, messageInput);
    setMessageInput('');
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  const filteredSessions = sessions.filter((session) => {
    const matchesSearch = session.customer_name?.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesDevice = selectedDeviceId === null || session.device_id === selectedDeviceId;
    return matchesSearch && matchesDevice;
  });

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) {
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } else if (days === 1) {
      return 'Yesterday';
    } else if (days < 7) {
      return date.toLocaleDateString([], { weekday: 'short' });
    }
    return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
  };

  return (
    <Layout className="chat-history-layout">
      {/* Left Panel - View Filters */}
      <Sider width={200} className="chat-sidebar-left">
        <div className="view-header">
          <Text strong>View</Text>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[filter]}
          onClick={({ key }) => setFilter(key)}
          items={viewFilters.map((item) => ({
            key: item.key,
            icon: <span>{item.icon}</span>,
            label: item.label,
          }))}
          className="view-menu"
        />
      </Sider>

      {/* Center-Left Panel - Chat List */}
      <Sider width={320} className="chat-list-panel">
        <div className="chat-list-header">
          {devices.length > 1 && (
            <div className="chat-device-filter">
              <MobileOutlined style={{ color: '#9ca3af', fontSize: 13 }} />
              <Select
                value={selectedDeviceId}
                onChange={setSelectedDeviceId}
                variant="borderless"
                size="small"
                style={{ flex: 1, fontSize: 13 }}
                popupMatchSelectWidth={false}
                options={[
                  { value: null, label: 'All Devices' },
                  ...devices.map((d) => ({
                    value: d.id,
                    label: `${d.push_name || 'Device'} (+${d.phone})`,
                  })),
                ]}
              />
            </div>
          )}
          <div className="chat-list-search">
            <Input
              placeholder="Search customers"
              prefix={<SearchOutlined />}
              suffix={<FilterOutlined />}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
        </div>

        <div className="chat-list-content">
          {isLoading ? (
            <div className="loading-center">
              <Spin />
            </div>
          ) : (
            <List
              dataSource={filteredSessions}
              renderItem={(session) => (
                <div
                  className={`chat-list-item ${currentSession?.id === session.id ? 'active' : ''}`}
                  onClick={() => handleSelectChat(session)}
                >
                  <Avatar
                    size={44}
                    style={{ backgroundColor: '#3b82f6', flexShrink: 0 }}
                  >
                    {session.customer_name?.[0] || '?'}
                  </Avatar>
                  <div className="chat-item-content">
                    <div className="chat-item-header">
                      <span className="customer-name">
                        {platformIcons[session.platform]}
                        {session.customer_name}
                      </span>
                      <span className="chat-time">
                        {session.last_message_at && formatTime(session.last_message_at)}
                      </span>
                    </div>
                    <div className="chat-item-preview">
                      <Text type="secondary" ellipsis>
                        {session.last_sender_type === 'bot' && (
                          <RobotOutlined className="bot-preview-icon" />
                        )}
                        {session.last_message || 'No messages'}
                      </Text>
                      {session.unread_count > 0 && (
                        <Badge count={session.unread_count} className="unread-badge" />
                      )}
                    </div>
                  </div>
                </div>
              )}
              locale={{ emptyText: 'No conversations found' }}
            />
          )}
        </div>
      </Sider>

      {/* Center Panel - Chat Messages */}
      <Content className="chat-messages-panel">
        {currentSession ? (
          <>
            <div className="messages-header">
              <div className="header-left">
                <Avatar size={40} style={{ backgroundColor: '#3b82f6' }}>
                  {currentSession.customer_name?.[0]}
                </Avatar>
                <div className="header-info">
                  <Text strong>{currentSession.customer_name}</Text>
                  <Text type="secondary">
                    {currentSession.platform}
                    {currentSession.device_id && devices.length > 1 && (() => {
                      const dev = devices.find((d) => d.id === currentSession.device_id);
                      return dev ? ` · ${dev.push_name || 'Device'} (+${dev.phone})` : '';
                    })()}
                  </Text>
                </div>
              </div>
              <div className="header-actions">
                <Button type="text" icon={<MoreOutlined />} />
                <Button type="primary" danger>End Conversation</Button>
              </div>
            </div>

            <div className="messages-container">
              {messages.map((msg: ChatMessage) => {
                const isBot = msg.sender_type === 'bot';
                const isOutgoing = msg.sender_type === 'agent' || isBot;
                return (
                  <div
                    key={msg.id}
                    className={`message-item ${isOutgoing ? 'outgoing' : 'incoming'} ${isBot ? 'bot' : ''}`}
                  >
                    {isBot && (
                      <div className="bot-label">
                        <RobotOutlined /> AI Bot
                      </div>
                    )}
                    <div className="message-bubble">
                      {msg.message_type === 'image' && msg.media_url ? (
                        <img src={msg.media_url} alt="attachment" className="message-image" />
                      ) : (
                        <Text>{msg.content}</Text>
                      )}
                    </div>
                    <div className="message-meta">
                      <Text type="secondary">
                        {formatTime(msg.created_at)}
                        {msg.is_read && ' • Read'}
                      </Text>
                    </div>
                  </div>
                );
              })}
              <div ref={messagesEndRef} />
            </div>

            <div className="message-input-container">
              <Popover
                trigger="click"
                content={
                  <div className="emoji-grid">
                    {['😀','😂','😊','😍','🥰','😎','🤔','👍','👋','🙏',
                      '❤️','🔥','✅','⭐','🎉','💯','👏','🤝','💪','😢',
                      '😮','😡','🤣','😴','🤗','😇','🥺','😏','🙄','😬'].map((emoji) => (
                      <span
                        key={emoji}
                        className="emoji-item"
                        onClick={() => setMessageInput((prev) => prev + emoji)}
                      >
                        {emoji}
                      </span>
                    ))}
                  </div>
                }
              >
                <Button type="text" icon={<SmileOutlined />} />
              </Popover>
              <input
                type="file"
                ref={fileInputRef}
                style={{ display: 'none' }}
                accept="image/*"
                onChange={async (e) => {
                  const file = e.target.files?.[0];
                  if (!file || !currentSession) return;
                  setUploading(true);
                  try {
                    await api.sendImageMessage(currentSession.id, file);
                    message.success('Image sent');
                  } catch {
                    message.error('Failed to send image');
                  } finally {
                    setUploading(false);
                    if (fileInputRef.current) fileInputRef.current.value = '';
                  }
                }}
              />
              <Button
                type="text"
                icon={<PictureOutlined />}
                loading={uploading}
                onClick={() => fileInputRef.current?.click()}
              />
              <TextArea
                placeholder="Type a message..."
                autoSize={{ minRows: 1, maxRows: 4 }}
                value={messageInput}
                onChange={(e) => setMessageInput(e.target.value)}
                onKeyDown={handleKeyPress}
                className="message-textarea"
              />
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={handleSendMessage}
                disabled={!messageInput.trim()}
              >
                Send
              </Button>
            </div>
          </>
        ) : (
          <div className="no-chat-selected">
            <MessagePlaceholder />
          </div>
        )}
      </Content>

      {/* Right Panel - Customer Profile */}
      {currentSession && (
        <Sider width={300} className="customer-profile-panel">
          <div className="profile-header">
            <Text strong>Customer Profile</Text>
            <Button type="text" icon={<MoreOutlined />} />
          </div>

          <div className="profile-content">
            <div className="profile-avatar-section">
              <Avatar size={64} style={{ backgroundColor: '#3b82f6' }}>
                {currentSession.customer_name?.[0]}
              </Avatar>
              <Title level={5}>{currentSession.customer_name}</Title>
            </div>

            <Card size="small" className="profile-card">
              <div className="profile-item">
                <MailOutlined />
                <Text type="secondary">--</Text>
              </div>
              <div className="profile-item">
                <PhoneOutlined />
                <Text>{currentSession.customer_phone || '--'}</Text>
              </div>
            </Card>

            <Divider />

            <div className="profile-section">
              <Text strong>Social Accounts</Text>
              <div className="social-accounts">
                <div className="social-item">
                  {platformIcons[currentSession.platform] || <GlobalOutlined />}
                  <Text>{currentSession.platform}</Text>
                  <Text type="secondary">{currentSession.customer_phone}</Text>
                </div>
              </div>
            </div>

            <Divider />

            <div className="profile-section">
              <div className="section-header">
                <Text strong>Tags</Text>
                <Popover
                  trigger="click"
                  title="Manage Tags"
                  content={
                    <div style={{ width: 200 }}>
                      <Space.Compact style={{ width: '100%', marginBottom: 8 }}>
                        <Input
                          size="small"
                          placeholder="New tag name"
                          value={newTagName}
                          onChange={(e) => setNewTagName(e.target.value)}
                          onPressEnter={handleCreateTag}
                        />
                        <Button size="small" type="primary" onClick={handleCreateTag}>Add</Button>
                      </Space.Compact>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                        {allTags
                          .filter((t) => !sessionTags.some((st) => st.id === t.id))
                          .map((tag) => (
                            <Tag
                              key={tag.id}
                              color={tag.color}
                              style={{ cursor: 'pointer' }}
                              onClick={() => handleAddTag(tag.id)}
                            >
                              + {tag.name}
                            </Tag>
                          ))}
                        {allTags.filter((t) => !sessionTags.some((st) => st.id === t.id)).length === 0 && (
                          <Text type="secondary" style={{ fontSize: 12 }}>No more tags to add</Text>
                        )}
                      </div>
                    </div>
                  }
                >
                  <Button type="link" size="small" icon={<PlusOutlined />}>Add</Button>
                </Popover>
              </div>
              <div className="tags-container">
                {sessionTags.map((tag) => (
                  <Tag
                    key={tag.id}
                    color={tag.color}
                    closable
                    onClose={() => handleRemoveTag(tag.id)}
                    icon={<TagOutlined />}
                  >
                    {tag.name}
                  </Tag>
                ))}
                {sessionTags.length === 0 && (
                  <Text type="secondary" style={{ fontSize: 12 }}>No tags</Text>
                )}
              </div>
            </div>

            <Divider />

            <div className="profile-actions">
              {currentSession.assigned_to_id ? (
                <Button type="primary" icon={<CheckCircleOutlined />} disabled>
                  Accepted
                </Button>
              ) : (
                <Button
                  type="primary"
                  icon={<CheckCircleOutlined />}
                  onClick={async () => {
                    try {
                      await api.acceptConversation(currentSession.id);
                      message.success('Conversation accepted');
                      fetchSession(currentSession.id);
                      fetchSessions();
                    } catch { message.error('Failed to accept'); }
                  }}
                >
                  Accept Conversation
                </Button>
              )}
              <Popconfirm
                title="Blacklist this customer?"
                description="This will close the conversation and block future messages."
                onConfirm={async () => {
                  try {
                    await api.blacklistCustomer(currentSession.id);
                    message.success('Customer blacklisted');
                    setCurrentSession(null);
                    fetchSessions();
                  } catch { message.error('Failed to blacklist'); }
                }}
                okText="Yes, Blacklist"
                okButtonProps={{ danger: true }}
              >
                <Button danger icon={<StopOutlined />}>Add to Blacklist</Button>
              </Popconfirm>
            </div>
          </div>
        </Sider>
      )}
    </Layout>
  );
}

function MessagePlaceholder() {
  return (
    <div className="placeholder-content">
      <div className="placeholder-icon">💬</div>
      <Title level={4}>Select a conversation</Title>
      <Text type="secondary">
        Choose a chat from the list to start messaging
      </Text>
    </div>
  );
}

