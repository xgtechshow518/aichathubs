import { useEffect, useState } from 'react';
import { Card, Col, Row, Typography, Spin, Statistic, Tag, List } from 'antd';
import { MessageOutlined, CommentOutlined } from '@ant-design/icons';
import api from '../../services/api';

interface AnalyticsData {
  messages_per_day: { date: string; count: number }[];
  sessions_per_day: { date: string; count: number }[];
  platform_breakdown: { platform: string; count: number }[];
  status_breakdown: { status: string; count: number }[];
  message_types: { status: string; count: number }[];
  total_sessions: number;
  total_messages: number;
}

export default function AdminAnalytics() {
  const [data, setData] = useState<AnalyticsData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadAnalytics();
  }, []);

  const loadAnalytics = async () => {
    try {
      const result = await api.getAdminChatAnalytics();
      setData(result as unknown as AnalyticsData);
    } catch {
      // handle error
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 100 }}><Spin size="large" /></div>;
  }

  if (!data) return null;

  const platformColors: Record<string, string> = {
    whatsapp: 'green', facebook: 'blue', telegram: 'cyan',
    instagram: 'magenta', line: 'lime', email: 'gold', unknown: 'default',
  };

  const renderBarChart = (chartData: { date: string; count: number }[], color: string) => {
    if (!chartData || chartData.length === 0) {
      return <Typography.Text type="secondary">No data available</Typography.Text>;
    }
    const max = Math.max(...chartData.map(i => i.count), 1);
    return (
      <div>
        <div style={{ display: 'flex', alignItems: 'flex-end', gap: 2, height: 140, padding: '0 4px' }}>
          {chartData.map((item, index) => {
            const height = (item.count / max) * 120;
            return (
              <div
                key={index}
                title={`${item.date}: ${item.count}`}
                style={{
                  flex: 1, height: Math.max(height, 3),
                  background: color, borderRadius: '2px 2px 0 0',
                  minWidth: 3, cursor: 'pointer', transition: 'opacity 0.2s',
                }}
                onMouseEnter={(e) => (e.currentTarget.style.opacity = '0.7')}
                onMouseLeave={(e) => (e.currentTarget.style.opacity = '1')}
              />
            );
          })}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 8, padding: '0 4px' }}>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>{chartData[0]?.date}</Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>{chartData[chartData.length - 1]?.date}</Typography.Text>
        </div>
      </div>
    );
  };

  return (
    <div>
      <Typography.Title level={4}>Chat Analytics</Typography.Title>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12}>
          <Card hoverable>
            <Statistic title="Total Chat Sessions" value={data.total_sessions} prefix={<CommentOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12}>
          <Card hoverable>
            <Statistic title="Total Messages" value={data.total_messages} prefix={<MessageOutlined />} />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card title="Messages per Day (Last 30 Days)" size="small">
            {renderBarChart(data.messages_per_day, '#1890ff')}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="New Sessions per Day (Last 30 Days)" size="small">
            {renderBarChart(data.sessions_per_day, '#52c41a')}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} sm={12} lg={8}>
          <Card title="Platform Breakdown" size="small">
            <List
              dataSource={data.platform_breakdown || []}
              renderItem={(item) => (
                <List.Item>
                  <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                    <Tag color={platformColors[item.platform] || 'default'}>{item.platform}</Tag>
                    <Typography.Text strong>{item.count}</Typography.Text>
                  </div>
                </List.Item>
              )}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card title="Session Status" size="small">
            <List
              dataSource={data.status_breakdown || []}
              renderItem={(item) => (
                <List.Item>
                  <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                    <Tag color={item.status === 'active' ? 'green' : 'default'}>{item.status}</Tag>
                    <Typography.Text strong>{item.count}</Typography.Text>
                  </div>
                </List.Item>
              )}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card title="Message Types" size="small">
            <List
              dataSource={data.message_types || []}
              renderItem={(item) => (
                <List.Item>
                  <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                    <Tag color={item.status === 'agent' ? 'blue' : item.status === 'customer' ? 'green' : 'orange'}>
                      {item.status}
                    </Tag>
                    <Typography.Text strong>{item.count}</Typography.Text>
                  </div>
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
