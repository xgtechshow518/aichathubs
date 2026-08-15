import { useEffect, useState, useCallback } from 'react';
import { Table, Select, Card, Typography, Tag, Space, message, Button } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import api from '../../services/api';

interface SubscriptionRow {
  id: number;
  user_id: number;
  stripe_subscription_id: string;
  stripe_price_id: string;
  plan: string;
  status: string;
  current_period_start?: string;
  current_period_end?: string;
  cancel_at_period_end: boolean;
  created_at: string;
  user_email: string;
  user_name: string;
}

export default function AdminPayments() {
  const [subscriptions, setSubscriptions] = useState<SubscriptionRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState('');
  const [planFilter, setPlanFilter] = useState('');

  const loadSubscriptions = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getAdminSubscriptions({
        page, page_size: pageSize, status: statusFilter, plan: planFilter,
      });
      setSubscriptions((data.subscriptions as SubscriptionRow[]) || []);
      setTotal(data.total as number);
    } catch {
      message.error('Failed to load subscriptions');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, statusFilter, planFilter]);

  useEffect(() => { loadSubscriptions(); }, [loadSubscriptions]);

  const statusColors: Record<string, string> = {
    active: 'green', trialing: 'blue', cancelled: 'red', past_due: 'orange', incomplete: 'volcano',
  };

  const columns = [
    { title: 'User', dataIndex: 'user_email', key: 'user_email' },
    { title: 'Name', dataIndex: 'user_name', key: 'user_name' },
    {
      title: 'Plan', dataIndex: 'plan', key: 'plan',
      render: (p: string) => <Tag color={p === 'active' ? 'green' : 'default'}>{p}</Tag>,
    },
    {
      title: 'Status', dataIndex: 'status', key: 'status',
      render: (s: string) => <Tag color={statusColors[s] || 'default'}>{s}</Tag>,
    },
    {
      title: 'Stripe ID', dataIndex: 'stripe_subscription_id', key: 'stripe_id',
      ellipsis: true,
      render: (id: string) => (
        <Typography.Text copyable={{ text: id }} style={{ fontSize: 12 }}>
          {id ? `...${id.slice(-12)}` : '-'}
        </Typography.Text>
      ),
    },
    {
      title: 'Period Start', dataIndex: 'current_period_start', key: 'period_start',
      render: (d: string) => d ? new Date(d).toLocaleDateString() : '-',
    },
    {
      title: 'Period End', dataIndex: 'current_period_end', key: 'period_end',
      render: (d: string) => d ? new Date(d).toLocaleDateString() : '-',
    },
    {
      title: 'Auto-Renew', dataIndex: 'cancel_at_period_end', key: 'cancel',
      render: (cancel: boolean) => (
        <Tag color={cancel ? 'red' : 'green'}>{cancel ? 'Cancelling' : 'Yes'}</Tag>
      ),
    },
    {
      title: 'Created', dataIndex: 'created_at', key: 'created_at',
      render: (d: string) => new Date(d).toLocaleDateString(),
    },
  ];

  return (
    <div>
      <Typography.Title level={4}>Payments & Subscriptions</Typography.Title>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Select
            placeholder="Status"
            value={statusFilter || undefined}
            onChange={(v) => { setStatusFilter(v || ''); setPage(1); }}
            style={{ width: 160 }}
            allowClear
            options={[
              { label: 'Active', value: 'active' },
              { label: 'Trialing', value: 'trialing' },
              { label: 'Cancelled', value: 'cancelled' },
              { label: 'Past Due', value: 'past_due' },
            ]}
          />
          <Select
            placeholder="Plan"
            value={planFilter || undefined}
            onChange={(v) => { setPlanFilter(v || ''); setPage(1); }}
            style={{ width: 160 }}
            allowClear
            options={[
              { label: 'Active', value: 'active' },
              { label: 'Trial', value: 'trial' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={loadSubscriptions}>Refresh</Button>
        </Space>
      </Card>

      <Table
        columns={columns}
        dataSource={subscriptions}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p) => setPage(p),
          showSizeChanger: false,
          showTotal: (t) => `${t} subscriptions`,
        }}
        size="middle"
        scroll={{ x: 1100 }}
      />
    </div>
  );
}
