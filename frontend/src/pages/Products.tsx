import { useEffect, useState } from 'react';
import {
  Table, Button, Input, Space, Typography, Modal, Form, InputNumber, Switch,
  message, Upload, Popconfirm, Tag, Image,
} from 'antd';
import {
  PlusOutlined, UploadOutlined, DownloadOutlined, ReloadOutlined, DeleteOutlined,
} from '@ant-design/icons';
import api from '../services/api';
import type { Product } from '../types';

const { Title, Text } = Typography;

export default function Products() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Product | null>(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const { products: list } = await api.listProducts({ search: search || undefined });
      setProducts(list || []);
    } catch {
      message.error('Failed to load products');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ currency: 'USD', active: true, stock: 0 });
    setModalOpen(true);
  };

  const openEdit = (p: Product) => {
    setEditing(p);
    form.setFieldsValue(p);
    setModalOpen(true);
  };

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      if (editing) {
        await api.updateProduct(editing.id, values);
        message.success('Product updated');
      } else {
        await api.createProduct(values);
        message.success('Product created');
      }
      setModalOpen(false);
      load();
    } catch {
      message.error('Failed to save product');
    }
  };

  const handleImport = async (file: File) => {
    try {
      const res = await api.importProducts(file);
      message.success(`Imported ${res.total} products (${res.created} new)`);
      load();
    } catch {
      message.error('Import failed');
    }
    return false;
  };

  const downloadTemplate = async () => {
    try {
      const blob = await api.downloadProductTemplate();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'products_template.xlsx';
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      message.error('Failed to download template');
    }
  };

  const filtered = products.filter((p) =>
    !search || p.name.toLowerCase().includes(search.toLowerCase()) ||
    p.sku.toLowerCase().includes(search.toLowerCase())
  );

  const columns = [
    { title: 'SKU', dataIndex: 'sku', key: 'sku', width: 120 },
    { title: 'Name', dataIndex: 'name', key: 'name' },
    {
      title: 'Price',
      key: 'price',
      render: (_: unknown, r: Product) => `${r.price.toFixed(2)} ${r.currency}`,
    },
    { title: 'Stock', dataIndex: 'stock', key: 'stock', width: 80 },
    { title: 'Category', dataIndex: 'category', key: 'category', width: 120 },
    {
      title: 'Status',
      key: 'active',
      width: 90,
      render: (_: unknown, r: Product) => (
        r.active ? <Tag color="green">Active</Tag> : <Tag>Inactive</Tag>
      ),
    },
    {
      title: 'Image',
      key: 'image',
      width: 80,
      render: (_: unknown, r: Product) => {
        const img = r.images?.find((i) => i.is_primary) || r.images?.[0];
        return img ? <Image src={img.url} width={40} height={40} style={{ objectFit: 'cover' }} /> : '—';
      },
    },
    {
      title: 'Links',
      key: 'links',
      width: 110,
      render: (_: unknown, r: Product) => {
        if (!r.product_url && !r.checkout_url) return '—';
        return (
          <Space size={4}>
            {r.product_url && (
              <a href={r.product_url} target="_blank" rel="noreferrer">Product</a>
            )}
            {r.checkout_url && (
              <a href={r.checkout_url} target="_blank" rel="noreferrer">Checkout</a>
            )}
          </Space>
        );
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 160,
      render: (_: unknown, r: Product) => (
        <Space>
          <Button size="small" onClick={() => openEdit(r)}>Edit</Button>
          <Popconfirm title="Delete this product?" onConfirm={async () => {
            await api.deleteProduct(r.id);
            message.success('Deleted');
            load();
          }}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>Product Catalog</Title>
          <Text type="secondary">Structured products power the AI sales bot via function calling.</Text>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load}>Refresh</Button>
          <Button icon={<DownloadOutlined />} onClick={downloadTemplate}>Template</Button>
          <Upload beforeUpload={(f) => { handleImport(f); return false; }} showUploadList={false} accept=".csv,.xlsx,.xls">
            <Button icon={<UploadOutlined />}>Import</Button>
          </Upload>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Product</Button>
        </Space>
      </div>

      <Input.Search
        placeholder="Search SKU or name"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        style={{ marginBottom: 16, maxWidth: 320 }}
        allowClear
      />

      <Table
        rowKey="id"
        columns={columns}
        dataSource={filtered}
        loading={loading}
        pagination={{ pageSize: 20 }}
      />

      <Modal
        title={editing ? 'Edit Product' : 'New Product'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleSave}
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="sku" label="SKU" rules={[{ required: true }]}>
            <Input disabled={!!editing} />
          </Form.Item>
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Space style={{ width: '100%' }}>
            <Form.Item name="price" label="Price" rules={[{ required: true }]}>
              <InputNumber min={0} precision={2} style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="currency" label="Currency">
              <Input style={{ width: 80 }} />
            </Form.Item>
            <Form.Item name="stock" label="Stock">
              <InputNumber min={0} style={{ width: 100 }} />
            </Form.Item>
          </Space>
          <Form.Item name="category" label="Category">
            <Input />
          </Form.Item>
          <Form.Item name="tags" label="Tags (comma-separated)">
            <Input />
          </Form.Item>
          <Form.Item
            name="product_url"
            label="Product URL"
            tooltip="Product page link. The AI bot can send this when a customer wants more info."
            rules={[{ type: 'url', message: 'Enter a valid URL' }]}
          >
            <Input placeholder="https://shop.example.com/p/sku-001" />
          </Form.Item>
          <Form.Item
            name="checkout_url"
            label="Checkout URL"
            tooltip="Direct buy/checkout link. The AI bot can send this when a customer is ready to purchase."
            rules={[{ type: 'url', message: 'Enter a valid URL' }]}
          >
            <Input placeholder="https://shop.example.com/checkout?sku=SKU-001" />
          </Form.Item>
          <Form.Item name="active" label="Active" valuePropName="checked">
            <Switch />
          </Form.Item>
          {editing && (
            <Form.Item label="Primary image URL">
              <Input.Search
                placeholder="https://..."
                enterButton="Add"
                onSearch={async (url) => {
                  if (!url.trim()) return;
                  await api.addProductImage(editing.id, url.trim(), true);
                  message.success('Image added');
                  load();
                  const { products: list } = await api.listProducts();
                  const updated = list.find((p) => p.id === editing.id);
                  if (updated) setEditing(updated);
                }}
              />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </div>
  );
}
