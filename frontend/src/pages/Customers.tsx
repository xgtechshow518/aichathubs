import { useEffect, useState } from 'react';
import {
  Card, Table, Input, Typography, Avatar, Tag, Button, Drawer, List,
  Tabs, Popconfirm, message,
} from 'antd';
import {
  SearchOutlined, WhatsAppOutlined, MessageOutlined, ClockCircleOutlined,
  StopOutlined, CheckCircleOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { ChatSession } from '../types';
import api from '../services/api';
import './Customers.css';

const { Title, Text } = Typography;

export default function Customers() {
  const navigate = useNavigate();
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [blacklisted, setBlacklisted] = useState<ChatSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [blacklistLoading, setBlacklistLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [activeTab, setActiveTab] = useState('customers');
  const [selectedCustomer, setSelectedCustomer] = useState<ChatSession | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  useEffect(() => {
    loadCustomers();
  }, []);

  useEffect(() => {
    if (activeTab === 'blacklist') {
      loadBlacklisted();
    }
  }, [activeTab]);

  const loadCustomers = async () => {
    setLoading(true);
    try {
      const data = await api.getChats({ page_size: 100 });
      setSessions(data.chats || []);
    } catch {
      // Error loading
    } finally {
      setLoading(false);
    }
  };

  const loadBlacklisted = async () => {
    setBlacklistLoading(true);
    try {
      const data = await api.getBlacklisted();
      setBlacklisted(data.sessions || []);
    } catch {
      // Error loading
    } finally {
      setBlacklistLoading(false);
    }
  };

  const handleUnblacklist = async (id: number) => {
    try {
      await api.unblacklistCustomer(id);
      message.success('Customer removed from blacklist');
      loadBlacklisted();
      loadCustomers();
    } catch {
      message.error('Failed to remove from blacklist');
    }
  };

  const filtered = sessions.filter((s) =>
    s.customer_name?.toLowerCase().includes(search.toLowerCase()) ||
    s.customer_phone?.includes(search)
  );

  const openProfile = (customer: ChatSession) => {
    setSelectedCustomer(customer);
    setDrawerOpen(true);
  };

  const formatDate = (dateStr: string) => {
    if (!dateStr) return '--';
    const d = new Date(dateStr);
    return d.toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });
  };

  const formatTime = (dateStr: string) => {
    if (!dateStr) return '--';
    const d = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - d.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    if (days === 0) return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    if (days === 1) return 'Yesterday';
    if (days < 7) return d.toLocaleDateString([], { weekday: 'short' });
    return formatDate(dateStr);
  };

  const columns = [
    {
      title: 'Customer',
      key: 'customer',
      render: (_: unknown, record: ChatSession) => (
        <div className="customer-cell" onClick={() => openProfile(record)} style={{ cursor: 'pointer' }}>
          <Avatar style={{ backgroundColor: '#3b82f6', flexShrink: 0 }}>
            {record.customer_name?.[0] || '?'}
          </Avatar>
          <div>
            <Text strong>{record.customer_name || 'Unknown'}</Text>
            <div><Text type="secondary" style={{ fontSize: 12 }}>{record.customer_phone}</Text></div>
          </div>
        </div>
      ),
    },
    {
      title: 'Platform',
      dataIndex: 'platform',
      key: 'platform',
      width: 120,
      render: (platform: string) => (
        <Tag icon={<WhatsAppOutlined />} color="green" style={{ textTransform: 'capitalize' }}>
          {platform}
        </Tag>
      ),
    },
    {
      title: 'Last Message',
      key: 'last_message',
      ellipsis: true,
      render: (_: unknown, record: ChatSession) => (
        <Text type="secondary" ellipsis style={{ maxWidth: 250 }}>
          {record.last_message || 'No messages'}
        </Text>
      ),
    },
    {
      title: 'Last Active',
      key: 'last_active',
      width: 140,
      render: (_: unknown, record: ChatSession) => (
        <Text type="secondary">{formatTime(record.last_message_at)}</Text>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'default'} style={{ textTransform: 'capitalize' }}>
          {status}
        </Tag>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 80,
      render: () => (
        <Button
          type="link"
          icon={<MessageOutlined />}
          onClick={() => navigate('/chats')}
        >
          Chat
        </Button>
      ),
    },
  ];

  const filteredBlacklisted = blacklisted.filter((s) =>
    s.customer_name?.toLowerCase().includes(search.toLowerCase()) ||
    s.customer_phone?.includes(search)
  );

  const blacklistColumns = [
    {
      title: 'Customer',
      key: 'customer',
      render: (_: unknown, record: ChatSession) => (
        <div className="customer-cell">
          <Avatar style={{ backgroundColor: '#ef4444', flexShrink: 0 }}>
            {record.customer_name?.[0] || '?'}
          </Avatar>
          <div>
            <Text strong>{record.customer_name || 'Unknown'}</Text>
            <div><Text type="secondary" style={{ fontSize: 12 }}>{record.customer_phone}</Text></div>
          </div>
        </div>
      ),
    },
    {
      title: 'Platform',
      dataIndex: 'platform',
      key: 'platform',
      width: 120,
      render: (platform: string) => (
        <Tag icon={<WhatsAppOutlined />} color="green" style={{ textTransform: 'capitalize' }}>
          {platform}
        </Tag>
      ),
    },
    {
      title: 'Last Message',
      key: 'last_message',
      ellipsis: true,
      render: (_: unknown, record: ChatSession) => (
        <Text type="secondary" ellipsis style={{ maxWidth: 250 }}>
          {record.last_message || 'No messages'}
        </Text>
      ),
    },
    {
      title: 'Blocked On',
      key: 'updated_at',
      width: 150,
      render: (_: unknown, record: ChatSession) => (
        <Text type="secondary">{formatDate(record.updated_at)}</Text>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 140,
      render: (_: unknown, record: ChatSession) => (
        <Popconfirm
          title="Remove from blacklist?"
          description="This customer will be able to message you again."
          onConfirm={() => handleUnblacklist(record.id)}
          okText="Yes, Remove"
        >
          <Button type="link" icon={<CheckCircleOutlined />}>
            Unblock
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div className="customers-page">
      <div className="customers-header">
        <div>
          <Title level={2}>Customers</Title>
          <Text type="secondary">
            {activeTab === 'customers'
              ? `${filtered.length} contact${filtered.length !== 1 ? 's' : ''}`
              : `${filteredBlacklisted.length} blocked`}
          </Text>
        </div>
      </div>

      <Card className="customers-card">
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'customers',
              label: `All Customers (${sessions.length})`,
            },
            {
              key: 'blacklist',
              label: (
                <span>
                  <StopOutlined style={{ marginRight: 4 }} />
                  Blacklist ({blacklisted.length})
                </span>
              ),
            },
          ]}
          style={{ marginBottom: 0 }}
        />

        <div className="customers-toolbar" style={{ marginTop: 16 }}>
          <Input
            placeholder="Search by name or phone..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ maxWidth: 320 }}
            allowClear
          />
        </div>

        {activeTab === 'customers' ? (
          <Table
            dataSource={filtered}
            columns={columns}
            rowKey="id"
            loading={loading}
            pagination={{ pageSize: 15, showSizeChanger: false }}
            locale={{ emptyText: 'No customers yet. They will appear here when someone messages your WhatsApp.' }}
          />
        ) : (
          <Table
            dataSource={filteredBlacklisted}
            columns={blacklistColumns}
            rowKey="id"
            loading={blacklistLoading}
            pagination={{ pageSize: 15, showSizeChanger: false }}
            locale={{ emptyText: 'No blacklisted customers. Customers you block will appear here.' }}
          />
        )}
      </Card>

      <Drawer
        title="Customer Profile"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={400}
      >
        {selectedCustomer && (
          <div className="customer-profile">
            <div className="profile-top">
              <Avatar size={64} style={{ backgroundColor: '#3b82f6' }}>
                {selectedCustomer.customer_name?.[0] || '?'}
              </Avatar>
              <Title level={4} style={{ marginTop: 12, marginBottom: 4 }}>
                {selectedCustomer.customer_name || 'Unknown'}
              </Title>
              <Tag icon={<WhatsAppOutlined />} color="green">{selectedCustomer.platform}</Tag>
            </div>

            <List
              className="profile-details"
              itemLayout="horizontal"
              dataSource={[
                { label: 'Phone', value: selectedCustomer.customer_phone || '--' },
                { label: 'Status', value: selectedCustomer.status },
                { label: 'First Contact', value: formatDate(selectedCustomer.created_at) },
                { label: 'Last Active', value: formatTime(selectedCustomer.last_message_at) },
                { label: 'Unread Messages', value: String(selectedCustomer.unread_count) },
              ]}
              renderItem={(item) => (
                <List.Item>
                  <Text type="secondary">{item.label}</Text>
                  <Text strong>{item.value}</Text>
                </List.Item>
              )}
            />

            {selectedCustomer.last_message && (
              <Card size="small" className="last-message-card">
                <Text type="secondary" style={{ fontSize: 12 }}>
                  <ClockCircleOutlined /> Last message
                </Text>
                <div style={{ marginTop: 4 }}>{selectedCustomer.last_message}</div>
              </Card>
            )}

            <Button
              type="primary"
              icon={<MessageOutlined />}
              block
              style={{ marginTop: 16 }}
              onClick={() => { setDrawerOpen(false); navigate('/chats'); }}
            >
              Open Conversation
            </Button>
          </div>
        )}
      </Drawer>
    </div>
  );
}
