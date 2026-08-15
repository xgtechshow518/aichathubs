import { useEffect, useState, useCallback } from 'react';
import {
  Table, Input, Select, Card, Typography, Tag, Button, Modal, Form, InputNumber,
  DatePicker, Switch, Drawer, Tabs, message, Space, Spin, Descriptions, Alert,
  Popconfirm, Divider,
} from 'antd';
import {
  SearchOutlined, DeleteOutlined, EyeOutlined, ReloadOutlined, LockOutlined,
  StopOutlined, CheckCircleOutlined, MessageOutlined, EditOutlined,
  RobotOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import api from '../../services/api';

interface UserRow {
  id: number;
  email: string;
  name: string;
  avatar_url: string;
  provider: string;
  subscription_plan: string;
  subscription_status: string;
  trial_ends_at?: string;
  max_devices: number;
  email_verified: boolean;
  suspended?: boolean;
  suspended_at?: string;
  suspended_reason?: string;
  created_at: string;
  device_count: number;
}

interface DeviceInfo {
  id: number;
  phone: string;
  jid: string;
  status: string;
  connected_at?: string;
  push_name?: string;
}

interface SubscriptionInfo {
  id: number;
  plan: string;
  status: string;
  stripe_subscription_id: string;
  current_period_start?: string;
  current_period_end?: string;
}

interface UserDetail {
  user: UserRow;
  devices: DeviceInfo[];
  subscriptions: SubscriptionInfo[];
  chat_count: number;
  message_count: number;
  has_kb: boolean;
  qa_count: number;
}

interface ChatSessionRow {
  id: number;
  customer_name: string;
  customer_phone: string;
  platform: string;
  status: string;
  unread_count: number;
  last_message: string;
  last_message_at?: string;
  created_at: string;
}

interface ChatMessageRow {
  id: number;
  sender_type: string;
  content: string;
  message_type: string;
  media_url?: string;
  is_read: boolean;
  created_at: string;
}

interface KBInfo {
  id?: number;
  user_id: number;
  auto_reply_enabled: boolean;
  system_prompt: string;
  last_synced_at?: string;
  qa_count?: number;
}

const planColors: Record<string, string> = {
  trial: 'orange', active: 'green', cancelled: 'red',
};
const statusColors: Record<string, string> = {
  trialing: 'blue', active: 'green', cancelled: 'red', past_due: 'orange',
};

export default function AdminUsers() {
  const [users, setUsers] = useState<UserRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [search, setSearch] = useState('');
  const [planFilter, setPlanFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const [detailUser, setDetailUser] = useState<UserDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('overview');

  // Device → chats drilldown state
  const [deviceChats, setDeviceChats] = useState<{ device?: DeviceInfo; sessions: ChatSessionRow[] }>({ sessions: [] });
  const [deviceChatsLoading, setDeviceChatsLoading] = useState(false);
  const [selectedDeviceId, setSelectedDeviceId] = useState<number | null>(null);

  // Chat messages modal
  const [messagesOpen, setMessagesOpen] = useState(false);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [currentSession, setCurrentSession] = useState<ChatSessionRow | null>(null);
  const [messages, setMessages] = useState<ChatMessageRow[]>([]);

  // Knowledge base state
  const [kb, setKb] = useState<KBInfo | null>(null);
  const [kbLoading, setKbLoading] = useState(false);
  const [kbSaving, setKbSaving] = useState(false);

  // Edit form
  const [editForm] = Form.useForm();
  const [editSaving, setEditSaving] = useState(false);

  const loadUsers = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getAdminUsers({
        page, page_size: pageSize, search, plan: planFilter, status: statusFilter,
      });
      setUsers((data.users as UserRow[]) || []);
      setTotal(data.total as number);
    } catch {
      message.error('Failed to load users');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search, planFilter, statusFilter]);

  useEffect(() => { loadUsers(); }, [loadUsers]);

  const openDrawer = async (id: number) => {
    setDrawerOpen(true);
    setActiveTab('overview');
    setDetailLoading(true);
    setDetailUser(null);
    setDeviceChats({ sessions: [] });
    setSelectedDeviceId(null);
    setKb(null);
    try {
      const data = await api.getAdminUser(id) as unknown as UserDetail;
      setDetailUser(data);
      editForm.setFieldsValue({
        name: data.user.name,
        subscription_plan: data.user.subscription_plan,
        subscription_status: data.user.subscription_status,
        max_devices: data.user.max_devices,
        trial_ends_at: data.user.trial_ends_at ? dayjs(data.user.trial_ends_at) : null,
        email_verified: data.user.email_verified,
      });
    } catch {
      message.error('Failed to load user details');
    } finally {
      setDetailLoading(false);
    }
  };

  const deleteUser = (id: number, email: string) => {
    Modal.confirm({
      title: 'Delete User',
      content: `Are you sure you want to delete ${email}? This action cannot be undone.`,
      okText: 'Delete',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.deleteAdminUser(id);
          message.success('User deleted');
          loadUsers();
        } catch {
          message.error('Failed to delete user');
        }
      },
    });
  };

  const handleSaveProfile = async () => {
    if (!detailUser) return;
    const values = await editForm.validateFields();
    setEditSaving(true);
    try {
      await api.updateAdminUser(detailUser.user.id, {
        name: values.name,
        subscription_plan: values.subscription_plan,
        subscription_status: values.subscription_status,
        max_devices: values.max_devices,
        trial_ends_at: values.trial_ends_at
          ? (values.trial_ends_at as Dayjs).toISOString()
          : '',
        email_verified: values.email_verified,
      });
      message.success('User settings updated');
      await openDrawer(detailUser.user.id);
      loadUsers();
    } catch {
      message.error('Failed to update user');
    } finally {
      setEditSaving(false);
    }
  };

  const handleResetPassword = async () => {
    if (!detailUser) return;
    Modal.confirm({
      title: 'Reset Password',
      content: 'A new random password will be generated and shown once. Continue?',
      okText: 'Reset',
      onOk: async () => {
        try {
          const res = await api.resetAdminUserPassword(detailUser.user.id);
          if (res.generated && res.password) {
            Modal.success({
              title: 'New Password Generated',
              content: (
                <div>
                  <p>Share this password with the user securely. It will not be shown again.</p>
                  <Input.Password value={res.password} readOnly />
                </div>
              ),
              width: 500,
            });
          } else {
            message.success('Password reset');
          }
        } catch {
          message.error('Failed to reset password');
        }
      },
    });
  };

  const handleToggleSuspend = async () => {
    if (!detailUser) return;
    const suspended = detailUser.user.suspended;
    if (suspended) {
      try {
        await api.unsuspendAdminUser(detailUser.user.id);
        message.success('User unsuspended');
        await openDrawer(detailUser.user.id);
        loadUsers();
      } catch {
        message.error('Failed to unsuspend');
      }
      return;
    }
    let reason = '';
    Modal.confirm({
      title: 'Suspend User',
      content: (
        <div>
          <p>The user will be blocked from logging in and any active session will be invalidated.</p>
          <Input.TextArea
            placeholder="Reason (optional)"
            rows={2}
            onChange={(e) => { reason = e.target.value; }}
          />
        </div>
      ),
      okText: 'Suspend',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.suspendAdminUser(detailUser.user.id, reason);
          message.success('User suspended');
          await openDrawer(detailUser.user.id);
          loadUsers();
        } catch {
          message.error('Failed to suspend');
        }
      },
    });
  };

  const loadDeviceChats = async (deviceId: number) => {
    setSelectedDeviceId(deviceId);
    setDeviceChatsLoading(true);
    try {
      const data = await api.getAdminDeviceChats(deviceId, { page: 1, page_size: 50 });
      setDeviceChats({
        device: data.device as DeviceInfo,
        sessions: (data.sessions as ChatSessionRow[]) || [],
      });
    } catch {
      message.error('Failed to load chats for this device');
    } finally {
      setDeviceChatsLoading(false);
    }
  };

  const openMessages = async (session: ChatSessionRow) => {
    setCurrentSession(session);
    setMessagesOpen(true);
    setMessagesLoading(true);
    setMessages([]);
    try {
      const data = await api.getAdminChatMessages(session.id, { page: 1, page_size: 200 });
      setMessages((data.messages as ChatMessageRow[]) || []);
    } catch {
      message.error('Failed to load messages');
    } finally {
      setMessagesLoading(false);
    }
  };

  const loadKB = useCallback(async () => {
    if (!detailUser) return;
    setKbLoading(true);
    try {
      const data = await api.getAdminUserKnowledge(detailUser.user.id);
      setKb(data);
    } catch {
      message.error('Failed to load knowledge base');
    } finally {
      setKbLoading(false);
    }
  }, [detailUser]);

  useEffect(() => {
    if (activeTab === 'knowledge' && detailUser && !kb) {
      loadKB();
    }
  }, [activeTab, detailUser, kb, loadKB]);

  const handleSaveKB = async () => {
    if (!detailUser || !kb) return;
    setKbSaving(true);
    try {
      await api.updateAdminUserKnowledge(detailUser.user.id, {
        auto_reply_enabled: kb.auto_reply_enabled,
        system_prompt: kb.system_prompt,
      });
      message.success('Knowledge base updated');
    } catch {
      message.error('Failed to update knowledge base');
    } finally {
      setKbSaving(false);
    }
  };

  const columns = [
    {
      title: 'Name', dataIndex: 'name', key: 'name',
      render: (name: string, record: UserRow) => (
        <Space>
          {record.avatar_url ? (
            <img src={record.avatar_url} alt="" style={{ width: 28, height: 28, borderRadius: '50%' }} />
          ) : (
            <div style={{ width: 28, height: 28, borderRadius: '50%', background: '#1890ff', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12 }}>
              {(name || '?')[0]?.toUpperCase()}
            </div>
          )}
          <span>{name || '-'}</span>
          {record.suspended && <Tag color="red">Suspended</Tag>}
        </Space>
      ),
    },
    { title: 'Email', dataIndex: 'email', key: 'email' },
    {
      title: 'Provider', dataIndex: 'provider', key: 'provider',
      render: (p: string) => <Tag>{p || 'email'}</Tag>,
    },
    {
      title: 'Plan', dataIndex: 'subscription_plan', key: 'plan',
      render: (p: string) => <Tag color={planColors[p] || 'default'}>{p}</Tag>,
    },
    {
      title: 'Status', dataIndex: 'subscription_status', key: 'status',
      render: (s: string) => <Tag color={statusColors[s] || 'default'}>{s}</Tag>,
    },
    { title: 'Devices', dataIndex: 'device_count', key: 'devices', width: 80 },
    {
      title: 'Created', dataIndex: 'created_at', key: 'created_at',
      render: (d: string) => new Date(d).toLocaleDateString(),
    },
    {
      title: 'Actions', key: 'actions', width: 120,
      render: (_: unknown, record: UserRow) => (
        <Space>
          <Button size="small" icon={<EyeOutlined />} onClick={() => openDrawer(record.id)} />
          <Button size="small" danger icon={<DeleteOutlined />} onClick={() => deleteUser(record.id, record.email)} />
        </Space>
      ),
    },
  ];

  const deviceColumns = [
    { title: 'Phone', dataIndex: 'phone', key: 'phone' },
    {
      title: 'Status', dataIndex: 'status', key: 'status',
      render: (s: string) => <Tag color={s === 'connected' ? 'green' : 'red'}>{s}</Tag>,
    },
    { title: 'JID', dataIndex: 'jid', key: 'jid', ellipsis: true },
    {
      title: 'Connected At', dataIndex: 'connected_at', key: 'connected_at',
      render: (d?: string) => d ? new Date(d).toLocaleString() : '-',
    },
    {
      title: 'Actions', key: 'actions', width: 140,
      render: (_: unknown, record: DeviceInfo) => (
        <Button
          size="small"
          type={selectedDeviceId === record.id ? 'primary' : 'default'}
          icon={<MessageOutlined />}
          onClick={() => loadDeviceChats(record.id)}
        >
          View Chats
        </Button>
      ),
    },
  ];

  const chatColumns = [
    { title: 'Customer', dataIndex: 'customer_name', key: 'customer_name',
      render: (v: string, r: ChatSessionRow) => v || r.customer_phone || '-' },
    { title: 'Phone', dataIndex: 'customer_phone', key: 'customer_phone', width: 140 },
    { title: 'Platform', dataIndex: 'platform', key: 'platform', width: 100,
      render: (p: string) => <Tag>{p || '-'}</Tag> },
    {
      title: 'Status', dataIndex: 'status', key: 'status', width: 90,
      render: (s: string) => <Tag color={s === 'active' ? 'green' : 'default'}>{s}</Tag>,
    },
    { title: 'Last Message', dataIndex: 'last_message', key: 'last_message', ellipsis: true },
    {
      title: 'Last At', dataIndex: 'last_message_at', key: 'last_message_at', width: 160,
      render: (d?: string) => d ? new Date(d).toLocaleString() : '-',
    },
    {
      title: '', key: 'actions', width: 90,
      render: (_: unknown, r: ChatSessionRow) => (
        <Button size="small" icon={<EyeOutlined />} onClick={() => openMessages(r)}>Open</Button>
      ),
    },
  ];

  return (
    <div>
      <Typography.Title level={4}>Users Management</Typography.Title>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Input
            placeholder="Search users..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            style={{ width: 250 }}
            allowClear
          />
          <Select
            placeholder="Plan"
            value={planFilter || undefined}
            onChange={(v) => { setPlanFilter(v || ''); setPage(1); }}
            style={{ width: 140 }}
            allowClear
            options={[
              { label: 'Trial', value: 'trial' },
              { label: 'Active', value: 'active' },
              { label: 'Cancelled', value: 'cancelled' },
            ]}
          />
          <Select
            placeholder="Status"
            value={statusFilter || undefined}
            onChange={(v) => { setStatusFilter(v || ''); setPage(1); }}
            style={{ width: 140 }}
            allowClear
            options={[
              { label: 'Trialing', value: 'trialing' },
              { label: 'Active', value: 'active' },
              { label: 'Cancelled', value: 'cancelled' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={loadUsers}>Refresh</Button>
        </Space>
      </Card>

      <Table
        columns={columns}
        dataSource={users}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p) => setPage(p),
          showSizeChanger: false,
          showTotal: (t) => `${t} users`,
        }}
        size="middle"
        scroll={{ x: 900 }}
      />

      <Drawer
        title={detailUser ? `User: ${detailUser.user.email}` : 'User Details'}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={960}
        destroyOnClose
      >
        {detailLoading || !detailUser ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : (
          <>
            {detailUser.user.suspended && (
              <Alert
                type="error"
                showIcon
                style={{ marginBottom: 16 }}
                message="This user is suspended"
                description={detailUser.user.suspended_reason || 'No reason provided.'}
              />
            )}
            <Space style={{ marginBottom: 16 }} wrap>
              <Button
                icon={<LockOutlined />}
                onClick={handleResetPassword}
              >
                Reset Password
              </Button>
              <Button
                danger={!detailUser.user.suspended}
                type={detailUser.user.suspended ? 'primary' : 'default'}
                icon={detailUser.user.suspended ? <CheckCircleOutlined /> : <StopOutlined />}
                onClick={handleToggleSuspend}
              >
                {detailUser.user.suspended ? 'Unsuspend' : 'Suspend'}
              </Button>
              <Popconfirm
                title="Delete this user?"
                description="This action cannot be undone."
                okType="danger"
                onConfirm={() => {
                  deleteUser(detailUser.user.id, detailUser.user.email);
                  setDrawerOpen(false);
                }}
              >
                <Button danger icon={<DeleteOutlined />}>Delete</Button>
              </Popconfirm>
            </Space>

            <Tabs
              activeKey={activeTab}
              onChange={setActiveTab}
              items={[
                {
                  key: 'overview',
                  label: 'Overview',
                  children: (
                    <div>
                      <Descriptions bordered column={2} size="small">
                        <Descriptions.Item label="Name">{detailUser.user.name || '-'}</Descriptions.Item>
                        <Descriptions.Item label="Email">{detailUser.user.email}</Descriptions.Item>
                        <Descriptions.Item label="Provider">{detailUser.user.provider || 'email'}</Descriptions.Item>
                        <Descriptions.Item label="Verified">
                          <Tag color={detailUser.user.email_verified ? 'green' : 'red'}>
                            {detailUser.user.email_verified ? 'Yes' : 'No'}
                          </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="Plan">
                          <Tag color={planColors[detailUser.user.subscription_plan] || 'default'}>
                            {detailUser.user.subscription_plan}
                          </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="Status">
                          <Tag color={statusColors[detailUser.user.subscription_status] || 'default'}>
                            {detailUser.user.subscription_status}
                          </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="Max Devices">{detailUser.user.max_devices}</Descriptions.Item>
                        <Descriptions.Item label="Trial Ends">
                          {detailUser.user.trial_ends_at ? new Date(detailUser.user.trial_ends_at).toLocaleString() : '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label="Chats">{detailUser.chat_count}</Descriptions.Item>
                        <Descriptions.Item label="Messages Sent">{detailUser.message_count}</Descriptions.Item>
                        <Descriptions.Item label="Knowledge Base">
                          {detailUser.has_kb ? `Yes (${detailUser.qa_count} Q&A)` : 'No'}
                        </Descriptions.Item>
                        <Descriptions.Item label="Created">
                          {new Date(detailUser.user.created_at).toLocaleString()}
                        </Descriptions.Item>
                      </Descriptions>

                      {detailUser.subscriptions.length > 0 && (
                        <>
                          <Typography.Title level={5} style={{ marginTop: 24 }}>Stripe Subscriptions</Typography.Title>
                          <Table
                            dataSource={detailUser.subscriptions}
                            rowKey="id"
                            size="small"
                            pagination={false}
                            columns={[
                              { title: 'Plan', dataIndex: 'plan', key: 'plan' },
                              {
                                title: 'Status', dataIndex: 'status', key: 'status',
                                render: (s: string) => <Tag color={statusColors[s] || 'default'}>{s}</Tag>,
                              },
                              { title: 'Stripe ID', dataIndex: 'stripe_subscription_id', key: 'stripe_id', ellipsis: true },
                              {
                                title: 'Period', key: 'period',
                                render: (_: unknown, r: SubscriptionInfo) => (
                                  <span>
                                    {r.current_period_start ? new Date(r.current_period_start).toLocaleDateString() : '-'}
                                    {' ~ '}
                                    {r.current_period_end ? new Date(r.current_period_end).toLocaleDateString() : '-'}
                                  </span>
                                ),
                              },
                            ]}
                          />
                        </>
                      )}
                    </div>
                  ),
                },
                {
                  key: 'devices',
                  label: <span><MessageOutlined /> WhatsApp & Chats</span>,
                  children: (
                    <div>
                      <Typography.Paragraph type="secondary">
                        Click <b>View Chats</b> on any connected device to see all customer conversations
                        routed through that WhatsApp account.
                      </Typography.Paragraph>
                      <Table
                        dataSource={detailUser.devices}
                        rowKey="id"
                        columns={deviceColumns}
                        size="small"
                        pagination={false}
                        locale={{ emptyText: 'No WhatsApp devices connected' }}
                      />

                      {selectedDeviceId && (
                        <>
                          <Divider />
                          <Typography.Title level={5}>
                            Chat sessions for {deviceChats.device?.phone || `device #${selectedDeviceId}`}
                          </Typography.Title>
                          <Table
                            dataSource={deviceChats.sessions}
                            rowKey="id"
                            columns={chatColumns}
                            size="small"
                            loading={deviceChatsLoading}
                            pagination={{ pageSize: 10 }}
                            locale={{ emptyText: 'No chat sessions on this device' }}
                          />
                        </>
                      )}
                    </div>
                  ),
                },
                {
                  key: 'settings',
                  label: <span><EditOutlined /> Edit Settings</span>,
                  children: (
                    <Form
                      form={editForm}
                      layout="vertical"
                      style={{ maxWidth: 600 }}
                    >
                      <Form.Item name="name" label="Name">
                        <Input />
                      </Form.Item>
                      <Form.Item name="email_verified" label="Email Verified" valuePropName="checked">
                        <Switch />
                      </Form.Item>
                      <Form.Item name="subscription_plan" label="Subscription Plan">
                        <Select
                          options={[
                            { label: 'Trial', value: 'trial' },
                            { label: 'Active', value: 'active' },
                            { label: 'Cancelled', value: 'cancelled' },
                          ]}
                        />
                      </Form.Item>
                      <Form.Item name="subscription_status" label="Subscription Status">
                        <Select
                          options={[
                            { label: 'Trialing', value: 'trialing' },
                            { label: 'Active', value: 'active' },
                            { label: 'Cancelled', value: 'cancelled' },
                            { label: 'Past Due', value: 'past_due' },
                          ]}
                        />
                      </Form.Item>
                      <Form.Item name="max_devices" label="Max WhatsApp Devices">
                        <InputNumber min={0} max={1000} style={{ width: '100%' }} />
                      </Form.Item>
                      <Form.Item
                        name="trial_ends_at"
                        label="Trial Ends At"
                        help="Leave empty to clear the trial expiry."
                      >
                        <DatePicker showTime style={{ width: '100%' }} />
                      </Form.Item>
                      <Form.Item>
                        <Button type="primary" loading={editSaving} onClick={handleSaveProfile}>
                          Save Changes
                        </Button>
                      </Form.Item>
                      <Alert
                        type="info"
                        showIcon
                        message="Note"
                        description="These fields override what Stripe reports. Use this to grant access, adjust device limits, or fix sync issues. The next Stripe sync may overwrite plan/status/max_devices."
                      />
                    </Form>
                  ),
                },
                {
                  key: 'knowledge',
                  label: <span><RobotOutlined /> Knowledge Base</span>,
                  children: kbLoading ? (
                    <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
                  ) : kb ? (
                    <div style={{ maxWidth: 700 }}>
                      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Space>
                          <Typography.Text strong>Auto-Reply</Typography.Text>
                          <Switch
                            checked={kb.auto_reply_enabled}
                            onChange={(v) => setKb({ ...kb, auto_reply_enabled: v })}
                          />
                        </Space>
                        {typeof kb.qa_count === 'number' && (
                          <Typography.Text type="secondary">{kb.qa_count} Q&A items</Typography.Text>
                        )}
                      </div>

                      <Form layout="vertical">
                        <Form.Item label="System Prompt" help="Instructions that shape how the bot replies for this user.">
                          <Input.TextArea
                            rows={6}
                            value={kb.system_prompt}
                            onChange={(e) => setKb({ ...kb, system_prompt: e.target.value })}
                            placeholder="e.g. You are a friendly customer support agent for AcmeCorp..."
                          />
                        </Form.Item>
                        <Form.Item>
                          <Button type="primary" loading={kbSaving} onClick={handleSaveKB}>
                            Save Knowledge Base
                          </Button>
                        </Form.Item>
                      </Form>
                    </div>
                  ) : null,
                },
              ]}
            />
          </>
        )}
      </Drawer>

      <Modal
        open={messagesOpen}
        onCancel={() => setMessagesOpen(false)}
        footer={null}
        title={currentSession ? `Chat with ${currentSession.customer_name || currentSession.customer_phone || 'customer'}` : 'Messages'}
        width={720}
      >
        {messagesLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : (
          <div style={{ maxHeight: 500, overflowY: 'auto', padding: 8, background: '#fafafa', borderRadius: 6 }}>
            {messages.length === 0 && <Typography.Text type="secondary">No messages.</Typography.Text>}
            {messages.map((m) => {
              const isAgent = m.sender_type === 'agent';
              return (
                <div
                  key={m.id}
                  style={{
                    display: 'flex',
                    justifyContent: isAgent ? 'flex-end' : 'flex-start',
                    marginBottom: 10,
                  }}
                >
                  <div
                    style={{
                      maxWidth: '75%',
                      background: isAgent ? '#1890ff' : '#fff',
                      color: isAgent ? '#fff' : '#000',
                      padding: '8px 12px',
                      borderRadius: 8,
                      border: isAgent ? 'none' : '1px solid #eee',
                      wordBreak: 'break-word',
                    }}
                  >
                    {m.message_type === 'image' && m.media_url ? (
                      <img src={m.media_url} alt="" style={{ maxWidth: 240, display: 'block', marginBottom: 4 }} />
                    ) : null}
                    {m.content && <div>{m.content}</div>}
                    <div style={{ fontSize: 11, opacity: 0.7, marginTop: 4, textAlign: 'right' }}>
                      {new Date(m.created_at).toLocaleString()}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </Modal>
    </div>
  );
}
