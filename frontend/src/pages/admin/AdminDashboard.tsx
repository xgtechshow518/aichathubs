import { useEffect, useState } from 'react';
import { Card, Col, Row, Statistic, Typography, Spin, Tag, List } from 'antd';
import {
  UserOutlined,
  MobileOutlined,
  MessageOutlined,
  CreditCardOutlined,
  RiseOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import api from '../../services/api';

interface DashboardData {
  total_users: number;
  verified_users: number;
  active_subscriptions: number;
  trialing_users: number;
  total_devices: number;
  connected_devices: number;
  total_chats: number;
  active_chats: number;
  total_messages: number;
  messages_today: number;
  total_subscriptions: number;
  new_users_today: number;
  new_users_this_week: number;
  user_trend: { date: string; count: number }[];
  plan_distribution: { plan: string; count: number }[];
  provider_distribution: { plan: string; count: number }[];
}

export default function AdminDashboard() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadDashboard();
  }, []);

  const loadDashboard = async () => {
    try {
      const result = await api.getAdminDashboard();
      setData(result as unknown as DashboardData);
    } catch {
      // handle error silently
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 100 }}><Spin size="large" /></div>;
  }

  if (!data) return null;

  const planColors: Record<string, string> = {
    trial: 'orange',
    active: 'green',
    cancelled: 'red',
    trialing: 'blue',
  };

  const providerColors: Record<string, string> = {
    email: '#1890ff',
    google: '#ea4335',
    facebook: '#4267B2',
  };

  return (
    <div>
      <Typography.Title level={4} style={{ marginBottom: 24 }}>Dashboard Overview</Typography.Title>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="Total Users"
              value={data.total_users}
              prefix={<TeamOutlined style={{ color: '#1890ff' }} />}
            />
            <div style={{ marginTop: 8 }}>
              <Typography.Text type="secondary">
                <RiseOutlined /> {data.new_users_today} today, {data.new_users_this_week} this week
              </Typography.Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="Active Subscriptions"
              value={data.active_subscriptions}
              prefix={<CreditCardOutlined style={{ color: '#52c41a' }} />}
            />
            <div style={{ marginTop: 8 }}>
              <Typography.Text type="secondary">
                <ClockCircleOutlined /> {data.trialing_users} on trial
              </Typography.Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="WhatsApp Devices"
              value={data.total_devices}
              prefix={<MobileOutlined style={{ color: '#722ed1' }} />}
            />
            <div style={{ marginTop: 8 }}>
              <Typography.Text type="secondary">
                <CheckCircleOutlined /> {data.connected_devices} connected
              </Typography.Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="Total Messages"
              value={data.total_messages}
              prefix={<MessageOutlined style={{ color: '#fa8c16' }} />}
            />
            <div style={{ marginTop: 8 }}>
              <Typography.Text type="secondary">
                <RiseOutlined /> {data.messages_today} today
              </Typography.Text>
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="Verified Users"
              value={data.verified_users}
              suffix={`/ ${data.total_users}`}
              prefix={<UserOutlined style={{ color: '#13c2c2' }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="Total Chats"
              value={data.total_chats}
              prefix={<MessageOutlined style={{ color: '#2f54eb' }} />}
            />
            <div style={{ marginTop: 8 }}>
              <Typography.Text type="secondary">
                {data.active_chats} active
              </Typography.Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="Total Subscriptions"
              value={data.total_subscriptions}
              prefix={<CreditCardOutlined style={{ color: '#eb2f96' }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="New Users Today"
              value={data.new_users_today}
              prefix={<RiseOutlined style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card title="Subscription Plans" size="small">
            <List
              dataSource={data.plan_distribution || []}
              renderItem={(item) => (
                <List.Item>
                  <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                    <Tag color={planColors[item.plan] || 'default'}>{item.plan || 'none'}</Tag>
                    <Typography.Text strong>{item.count} users</Typography.Text>
                  </div>
                </List.Item>
              )}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="Auth Providers" size="small">
            <List
              dataSource={data.provider_distribution || []}
              renderItem={(item) => (
                <List.Item>
                  <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <div style={{
                        width: 12, height: 12, borderRadius: '50%',
                        background: providerColors[item.plan] || '#999',
                      }} />
                      <Typography.Text>{item.plan || 'email'}</Typography.Text>
                    </div>
                    <Typography.Text strong>{item.count} users</Typography.Text>
                  </div>
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>

      {data.user_trend && data.user_trend.length > 0 && (
        <Card title="User Registrations (Last 30 Days)" size="small" style={{ marginTop: 16 }}>
          <div style={{ display: 'flex', alignItems: 'flex-end', gap: 2, height: 120, padding: '0 8px' }}>
            {data.user_trend.map((item, index) => {
              const max = Math.max(...data.user_trend.map(i => i.count), 1);
              const height = (item.count / max) * 100;
              return (
                <div
                  key={index}
                  title={`${item.date}: ${item.count} users`}
                  style={{
                    flex: 1,
                    height: Math.max(height, 4),
                    background: '#1890ff',
                    borderRadius: '2px 2px 0 0',
                    minWidth: 4,
                    cursor: 'pointer',
                    transition: 'opacity 0.2s',
                  }}
                  onMouseEnter={(e) => (e.currentTarget.style.opacity = '0.7')}
                  onMouseLeave={(e) => (e.currentTarget.style.opacity = '1')}
                />
              );
            })}
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 8, padding: '0 8px' }}>
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              {data.user_trend[0]?.date}
            </Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              {data.user_trend[data.user_trend.length - 1]?.date}
            </Typography.Text>
          </div>
        </Card>
      )}
    </div>
  );
}
