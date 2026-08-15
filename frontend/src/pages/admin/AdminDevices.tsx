import { useEffect, useState, useCallback } from 'react';
import { Table, Input, Select, Card, Typography, Tag, Space, message, Button } from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import api from '../../services/api';

interface DeviceRow {
  id: number;
  user_id: number;
  jid: string;
  push_name: string;
  phone: string;
  status: string;
  connected_at?: string;
  created_at: string;
  user_email: string;
  user_name: string;
}

export default function AdminDevices() {
  const [devices, setDevices] = useState<DeviceRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const loadDevices = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getAdminDevices({
        page, page_size: pageSize, search, status: statusFilter,
      });
      setDevices((data.devices as DeviceRow[]) || []);
      setTotal(data.total as number);
    } catch {
      message.error('Failed to load devices');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search, statusFilter]);

  useEffect(() => { loadDevices(); }, [loadDevices]);

  const columns = [
    { title: 'Phone', dataIndex: 'phone', key: 'phone' },
    { title: 'Push Name', dataIndex: 'push_name', key: 'push_name' },
    { title: 'JID', dataIndex: 'jid', key: 'jid', ellipsis: true },
    {
      title: 'Status', dataIndex: 'status', key: 'status',
      render: (s: string) => (
        <Tag color={s === 'connected' ? 'green' : s === 'scanning' ? 'blue' : 'red'}>
          {s}
        </Tag>
      ),
    },
    { title: 'User', dataIndex: 'user_email', key: 'user_email' },
    { title: 'User Name', dataIndex: 'user_name', key: 'user_name' },
    {
      title: 'Connected At', dataIndex: 'connected_at', key: 'connected_at',
      render: (d: string) => d ? new Date(d).toLocaleString() : '-',
    },
    {
      title: 'Created', dataIndex: 'created_at', key: 'created_at',
      render: (d: string) => new Date(d).toLocaleDateString(),
    },
  ];

  return (
    <div>
      <Typography.Title level={4}>WhatsApp Devices</Typography.Title>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Input
            placeholder="Search by phone, JID, or user email..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            style={{ width: 300 }}
            allowClear
          />
          <Select
            placeholder="Status"
            value={statusFilter || undefined}
            onChange={(v) => { setStatusFilter(v || ''); setPage(1); }}
            style={{ width: 160 }}
            allowClear
            options={[
              { label: 'Connected', value: 'connected' },
              { label: 'Disconnected', value: 'disconnected' },
              { label: 'Scanning', value: 'scanning' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={loadDevices}>Refresh</Button>
        </Space>
      </Card>

      <Table
        columns={columns}
        dataSource={devices}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p) => setPage(p),
          showSizeChanger: false,
          showTotal: (t) => `${t} devices`,
        }}
        size="middle"
        scroll={{ x: 1000 }}
      />
    </div>
  );
}
