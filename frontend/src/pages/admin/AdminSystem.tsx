import { useEffect, useState } from 'react';
import { Card, Col, Row, Typography, Spin, Statistic, Tag, Table, Button } from 'antd';
import {
  DatabaseOutlined, ClockCircleOutlined, CheckCircleOutlined,
  CloseCircleOutlined, ReloadOutlined,
} from '@ant-design/icons';
import api from '../../services/api';

interface SystemData {
  db_status: string;
  uptime: string;
  uptime_seconds: number;
  tables: Record<string, number>;
}

export default function AdminSystem() {
  const [data, setData] = useState<SystemData | null>(null);
  const [loading, setLoading] = useState(true);

  const loadSystemInfo = async () => {
    setLoading(true);
    try {
      const result = await api.getAdminSystemInfo();
      setData(result as unknown as SystemData);
    } catch {
      // handle error
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadSystemInfo(); }, []);

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 100 }}><Spin size="large" /></div>;
  }

  if (!data) return null;

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const parts: string[] = [];
    if (days > 0) parts.push(`${days}d`);
    if (hours > 0) parts.push(`${hours}h`);
    parts.push(`${mins}m`);
    return parts.join(' ');
  };

  const tableData = Object.entries(data.tables).map(([name, count]) => ({
    key: name,
    name,
    count,
  }));

  const totalRecords = Object.values(data.tables).reduce((a, b) => a + b, 0);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Typography.Title level={4} style={{ margin: 0 }}>System Information</Typography.Title>
        <Button icon={<ReloadOutlined />} onClick={loadSystemInfo}>Refresh</Button>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card hoverable>
            <Statistic
              title="Database Status"
              value={data.db_status}
              prefix={data.db_status === 'connected'
                ? <CheckCircleOutlined style={{ color: '#52c41a' }} />
                : <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
              }
              valueStyle={{ color: data.db_status === 'connected' ? '#52c41a' : '#ff4d4f' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card hoverable>
            <Statistic
              title="Server Uptime"
              value={formatUptime(data.uptime_seconds)}
              prefix={<ClockCircleOutlined style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card hoverable>
            <Statistic
              title="Total Records"
              value={totalRecords}
              prefix={<DatabaseOutlined style={{ color: '#722ed1' }} />}
            />
          </Card>
        </Col>
      </Row>

      <Card title="Database Tables" size="small" style={{ marginTop: 16 }}>
        <Table
          dataSource={tableData}
          pagination={false}
          size="small"
          columns={[
            {
              title: 'Table', dataIndex: 'name', key: 'name',
              render: (name: string) => (
                <Typography.Text code>{name}</Typography.Text>
              ),
            },
            {
              title: 'Records', dataIndex: 'count', key: 'count',
              render: (count: number) => (
                <Tag color={count > 0 ? 'blue' : 'default'}>{count.toLocaleString()}</Tag>
              ),
              sorter: (a: { count: number }, b: { count: number }) => a.count - b.count,
            },
          ]}
        />
      </Card>
    </div>
  );
}
