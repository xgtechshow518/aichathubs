import { useEffect, useState } from 'react';
import { Table, Typography, Button, Tag, message } from 'antd';
import { ReloadOutlined, UserOutlined } from '@ant-design/icons';
import api from '../services/api';
import type { LeadRow } from '../types';

const { Title, Text } = Typography;

export default function Leads() {
  const [leads, setLeads] = useState<LeadRow[]>([]);
  const [loading, setLoading] = useState(true);

  const load = async () => {
    setLoading(true);
    try {
      const { leads: list } = await api.listLeads();
      setLeads(list || []);
    } catch {
      message.error('Failed to load leads');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const columns = [
    {
      title: 'Customer',
      key: 'customer',
      render: (_: unknown, r: LeadRow) => (
        <span>
          <UserOutlined /> {r.customer_name || r.customer_phone || 'Unknown'}
        </span>
      ),
    },
    { title: 'SKU', dataIndex: 'sku', key: 'sku' },
    { title: 'Qty', dataIndex: 'quantity', key: 'quantity', width: 70 },
    { title: 'Notes', dataIndex: 'notes', key: 'notes', ellipsis: true },
    {
      title: 'Captured',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: 'Session',
      dataIndex: 'session_id',
      key: 'session_id',
      width: 90,
      render: (id: number) => <Tag>#{id}</Tag>,
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>Sales Leads</Title>
          <Text type="secondary">Leads captured by the AI bot via capture_lead during WhatsApp conversations.</Text>
        </div>
        <Button icon={<ReloadOutlined />} onClick={load}>Refresh</Button>
      </div>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={leads}
        loading={loading}
        pagination={{ pageSize: 20 }}
        locale={{ emptyText: 'No leads yet — the bot will record them when customers show purchase intent.' }}
      />
    </div>
  );
}
