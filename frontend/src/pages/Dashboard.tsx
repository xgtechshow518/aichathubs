import { useEffect } from 'react';
import { Card, Col, Row, Statistic, Typography, List, Avatar, Tag, message } from 'antd';
import {
  MessageOutlined,
  TeamOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  WhatsAppOutlined,
} from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import { useChatStore } from '../store/chatStore';
import { useAuthStore } from '../store/authStore';
import api from '../services/api';
import './Dashboard.css';

const { Title, Text } = Typography;

const platformIcons: Record<string, React.ReactNode> = {
  whatsapp: <WhatsAppOutlined style={{ color: '#25D366' }} />,
};

export default function Dashboard() {
  const { stats, sessions, fetchStats, fetchSessions } = useChatStore();
  const { fetchUser } = useAuthStore();
  const [searchParams, setSearchParams] = useSearchParams();

  useEffect(() => {
    fetchStats();
    fetchSessions();

    const paymentStatus = searchParams.get('payment');
    if (paymentStatus === 'success') {
      searchParams.delete('payment');
      setSearchParams(searchParams, { replace: true });

      const syncPayment = async () => {
        try {
          const result = await api.syncSubscription();
          if (result.synced) {
            message.success(`Payment successful! You now have ${result.max_devices} device${result.max_devices > 1 ? 's' : ''}.`);
          }
          await fetchUser();
        } catch {
          await fetchUser();
        }
      };
      syncPayment();
    }
  }, []);

  const recentChats = sessions.slice(0, 5);

  return (
    <div className="dashboard-page">
      <div className="dashboard-header">
        <Title level={2}>Dashboard</Title>
        <Text type="secondary">Welcome back! Here's your chat overview.</Text>
      </div>

      {/* Stats Cards */}
      <Row gutter={[24, 24]} className="stats-row">
        <Col xs={24} sm={12} lg={6}>
          <Card className="stat-card">
            <Statistic
              title="Total Chats"
              value={stats?.total_chats || 0}
              prefix={<MessageOutlined />}
              valueStyle={{ color: '#3b82f6' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="stat-card">
            <Statistic
              title="Unread Messages"
              value={stats?.unread_chats || 0}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#f59e0b' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="stat-card">
            <Statistic
              title="Active Chats"
              value={stats?.active_chats || 0}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#22c55e' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="stat-card">
            <Statistic
              title="Customers"
              value={sessions.length}
              prefix={<TeamOutlined />}
              valueStyle={{ color: '#8b5cf6' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Recent Chats */}
      <Row gutter={[24, 24]}>
        <Col xs={24} lg={16}>
          <Card title="Recent Conversations" className="recent-chats-card">
            <List
              itemLayout="horizontal"
              dataSource={recentChats}
              renderItem={(chat) => (
                <List.Item
                  actions={[
                    chat.unread_count > 0 && (
                      <Tag color="red">{chat.unread_count} new</Tag>
                    ),
                  ].filter(Boolean)}
                >
                  <List.Item.Meta
                    avatar={
                      <Avatar style={{ backgroundColor: '#3b82f6' }}>
                        {chat.customer_name?.[0] || '?'}
                      </Avatar>
                    }
                    title={
                      <span className="chat-title">
                        {platformIcons[chat.platform]}
                        <span className="customer-name">{chat.customer_name}</span>
                      </span>
                    }
                    description={
                      <Text type="secondary" ellipsis>
                        {chat.last_message || 'No messages yet'}
                      </Text>
                    }
                  />
                </List.Item>
              )}
              locale={{ emptyText: 'No recent conversations' }}
            />
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card title="Quick Actions" className="quick-actions-card">
            <div className="quick-action-list">
              <div className="quick-action-item">
                <div className="action-icon whatsapp">
                  <WhatsAppOutlined />
                </div>
                <div className="action-info">
                  <Text strong>WhatsApp</Text>
                  <Text type="secondary">Connected</Text>
                </div>
                <Tag color="success">Active</Tag>
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}

