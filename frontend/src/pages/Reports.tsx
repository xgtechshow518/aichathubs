import { useEffect, useState } from 'react';
import { Avatar, Card, Col, Row, Statistic, Typography, Table, Tag, Spin, Progress } from 'antd';
import {
  MessageOutlined,
  TeamOutlined,
  RobotOutlined,
  UserOutlined,
  WhatsAppOutlined,
  BarChartOutlined,
} from '@ant-design/icons';
import api from '../services/api';
import './Reports.css';

const { Title, Text } = Typography;

interface DayStat {
  day: string;
  count: number;
}

interface TopCustomer {
  customer_name: string;
  platform: string;
  message_count: number;
}

interface PlatformStat {
  platform: string;
  count: number;
}

export default function Reports() {
  const [data, setData] = useState<Record<string, unknown> | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const result = await api.getReportData();
      setData(result);
    } catch {
      // Error
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  const totalMessages = (data?.total_messages as number) || 0;
  const customerMessages = (data?.customer_messages as number) || 0;
  const agentMessages = (data?.agent_messages as number) || 0;
  const totalCustomers = (data?.total_customers as number) || 0;
  const autoReplyRate = (data?.auto_reply_rate as number) || 0;
  const dailyStats = (data?.daily_stats as DayStat[]) || [];
  const topCustomers = (data?.top_customers as TopCustomer[]) || [];
  const platformStats = (data?.platform_stats as PlatformStat[]) || [];

  const maxDayCount = Math.max(...dailyStats.map(d => d.count), 1);

  return (
    <div className="reports-page">
      <div className="reports-header">
        <Title level={2}>Reports</Title>
        <Text type="secondary">Overview of your messaging activity</Text>
      </div>

      {/* Stats Cards */}
      <Row gutter={[16, 16]} className="stats-row">
        <Col xs={24} sm={12} lg={6}>
          <Card className="stat-card">
            <Statistic
              title="Total Messages"
              value={totalMessages}
              prefix={<MessageOutlined />}
              valueStyle={{ color: '#3b82f6' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="stat-card">
            <Statistic
              title="Customer Messages"
              value={customerMessages}
              prefix={<UserOutlined />}
              valueStyle={{ color: '#8b5cf6' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="stat-card">
            <Statistic
              title="Bot Replies"
              value={agentMessages}
              prefix={<RobotOutlined />}
              valueStyle={{ color: '#22c55e' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="stat-card">
            <Statistic
              title="Total Customers"
              value={totalCustomers}
              prefix={<TeamOutlined />}
              valueStyle={{ color: '#f59e0b' }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        {/* Daily Activity */}
        <Col xs={24} lg={14}>
          <Card title={<span><BarChartOutlined /> Messages (Last 7 Days)</span>} className="report-card">
            {dailyStats.length > 0 ? (
              <div className="daily-chart">
                {dailyStats.map((stat) => (
                  <div key={stat.day} className="chart-bar-row">
                    <Text type="secondary" className="chart-label">
                      {new Date(stat.day).toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })}
                    </Text>
                    <div className="chart-bar-container">
                      <div
                        className="chart-bar"
                        style={{ width: `${(stat.count / maxDayCount) * 100}%` }}
                      />
                      <Text className="chart-value">{stat.count}</Text>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <Text type="secondary">No message data for the last 7 days</Text>
            )}
          </Card>
        </Col>

        {/* Auto-Reply Rate */}
        <Col xs={24} lg={10}>
          <Card title="Auto-Reply Performance" className="report-card">
            <div className="reply-rate">
              <Progress
                type="circle"
                percent={Math.round(autoReplyRate)}
                size={140}
                strokeColor="#22c55e"
                format={(p) => <span style={{ fontSize: 28, fontWeight: 700 }}>{p}%</span>}
              />
              <div className="rate-label">
                <Text type="secondary">of customer messages received a bot reply</Text>
              </div>
            </div>

            <div className="rate-breakdown">
              <div className="rate-item">
                <div className="rate-dot" style={{ background: '#8b5cf6' }} />
                <Text type="secondary">Customer Messages</Text>
                <Text strong>{customerMessages}</Text>
              </div>
              <div className="rate-item">
                <div className="rate-dot" style={{ background: '#22c55e' }} />
                <Text type="secondary">Bot Replies</Text>
                <Text strong>{agentMessages}</Text>
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        {/* Top Customers */}
        <Col xs={24} lg={14}>
          <Card title="Top Customers" className="report-card">
            <Table
              dataSource={topCustomers}
              rowKey="customer_name"
              pagination={false}
              size="small"
              columns={[
                {
                  title: 'Customer',
                  dataIndex: 'customer_name',
                  render: (name: string) => (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Avatar size="small" style={{ backgroundColor: '#3b82f6' }}>{name?.[0]}</Avatar>
                      {name}
                    </div>
                  ),
                },
                {
                  title: 'Platform',
                  dataIndex: 'platform',
                  render: (p: string) => <Tag icon={<WhatsAppOutlined />} color="green">{p}</Tag>,
                },
                {
                  title: 'Messages',
                  dataIndex: 'message_count',
                  align: 'right' as const,
                  render: (count: number) => <Text strong>{count}</Text>,
                },
              ]}
              locale={{ emptyText: 'No customer data yet' }}
            />
          </Card>
        </Col>

        {/* Platform Breakdown */}
        <Col xs={24} lg={10}>
          <Card title="Platform Breakdown" className="report-card">
            {platformStats.length > 0 ? (
              <div className="platform-breakdown">
                {platformStats.map((stat) => (
                  <div key={stat.platform} className="platform-row">
                    <div className="platform-info">
                      <WhatsAppOutlined style={{ color: '#25D366', fontSize: 18 }} />
                      <Text style={{ textTransform: 'capitalize' }}>{stat.platform}</Text>
                    </div>
                    <Text strong>{stat.count} customer{stat.count !== 1 ? 's' : ''}</Text>
                  </div>
                ))}
              </div>
            ) : (
              <Text type="secondary">No platform data yet</Text>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}
