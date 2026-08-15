import { useEffect, useRef, useState } from 'react';
import {
  Card, Button, Table, Modal, Form, Input, Switch, Typography, Space,
  Upload, message, Tag, Popconfirm, Divider, Spin, Progress,
} from 'antd';
import {
  PlusOutlined, UploadOutlined, DeleteOutlined, DownloadOutlined, SearchOutlined,
  EditOutlined, SendOutlined, RobotOutlined, CloudUploadOutlined, CheckCircleOutlined,
  UserOutlined, ReloadOutlined,
} from '@ant-design/icons';
import type { QAItem, KnowledgeBase } from '../types';
import api from '../services/api';
import './Bots.css';

const { Title, Text } = Typography;

type TestMessage = { role: 'user' | 'bot'; content: string };

export default function Bots() {
  const [qaItems, setQaItems] = useState<QAItem[]>([]);
  const [kb, setKb] = useState<KnowledgeBase | null>(null);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<QAItem | null>(null);
  const [testModalOpen, setTestModalOpen] = useState(false);
  const [testInput, setTestInput] = useState('');
  const [testMessages, setTestMessages] = useState<TestMessage[]>([]);
  const [testing, setTesting] = useState(false);
  const testScrollRef = useRef<HTMLDivElement | null>(null);
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [uploadStep, setUploadStep] = useState<'idle' | 'uploading' | 'analyzing' | 'done' | 'error'>('idle');
  const [uploadFileName, setUploadFileName] = useState('');
  const [uploadProgress, setUploadProgress] = useState(0);
  const [searchText, setSearchText] = useState('');
  const [form] = Form.useForm();

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [kbData, qaData] = await Promise.all([
        api.getKnowledgeBase(),
        api.getQAItems(),
      ]);
      setKb(kbData);
      setQaItems(qaData.items || []);
    } catch {
      // KB might not exist yet
    } finally {
      setLoading(false);
    }
  };

  const handleAutoReplyToggle = async (checked: boolean) => {
    try {
      const updated = await api.updateKnowledgeBase({ auto_reply_enabled: checked });
      setKb(updated);
      message.success(checked ? 'Auto-reply enabled' : 'Auto-reply disabled');
    } catch {
      message.error('Failed to update setting');
    }
  };

  const handleAddEdit = () => {
    form.validateFields().then(async (values) => {
      try {
        if (editingItem) {
          await api.updateQAItem(editingItem.id, values);
          message.success('Q&A item updated');
        } else {
          await api.createQAItem(values);
          message.success('Q&A item created');
        }
        setModalOpen(false);
        setEditingItem(null);
        form.resetFields();
        loadData();
      } catch {
        message.error('Failed to save Q&A item');
      }
    });
  };

  const handleDelete = async (id: number) => {
    try {
      await api.deleteQAItem(id);
      message.success('Q&A item deleted');
      loadData();
    } catch {
      message.error('Failed to delete');
    }
  };

  const handleDeleteAll = async () => {
    try {
      const result = await api.deleteAllQAItems();
      message.success(`Deleted ${result.deleted} Q&A items`);
      loadData();
    } catch {
      message.error('Failed to delete all items');
    }
  };

  const handleDownloadTemplate = async () => {
    try {
      const response = await api.downloadTemplate();
      const url = window.URL.createObjectURL(new Blob([response]));
      const link = document.createElement('a');
      link.href = url;
      link.download = 'qa_template.xlsx';
      link.click();
      window.URL.revokeObjectURL(url);
    } catch {
      message.error('Failed to download template');
    }
  };

  const handleFileUpload = async (file: File) => {
    setUploadFileName(file.name);
    setUploadStep('uploading');
    setUploadProgress(0);
    setUploadModalOpen(true);

    // Simulate upload progress
    const progressInterval = setInterval(() => {
      setUploadProgress((prev) => {
        if (prev >= 90) { clearInterval(progressInterval); return 90; }
        return prev + Math.random() * 15;
      });
    }, 300);

    try {
      // Upload phase
      setUploadStep('uploading');
      await new Promise((r) => setTimeout(r, 500));

      setUploadStep('analyzing');
      setUploadProgress(95);
      clearInterval(progressInterval);

      const result = await api.uploadKnowledgeFile(file);

      setUploadProgress(100);
      setUploadStep('done');
      loadData();

      if (result.count) {
        message.success(`Imported ${result.count} Q&A items from ${file.name}`);
      }
    } catch (err: unknown) {
      clearInterval(progressInterval);
      setUploadStep('error');
      const error = err as { response?: { data?: { message?: string } } };
      message.error(error.response?.data?.message || 'Upload failed');
    }
    return false;
  };

  const closeUploadModal = () => {
    setUploadModalOpen(false);
    setUploadStep('idle');
    setUploadProgress(0);
    setUploadFileName('');
  };

  useEffect(() => {
    // Keep the test conversation scrolled to the latest message.
    const el = testScrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [testMessages, testing]);

  const handleSendTestMessage = async () => {
    const text = testInput.trim();
    if (!text || testing) return;

    // The conversation so far (before this turn) is the context we replay to
    // the backend so the bot answers with full multi-turn memory.
    const history = testMessages.map((m) => ({ role: m.role, content: m.content }));

    setTestMessages((prev) => [...prev, { role: 'user', content: text }]);
    setTestInput('');
    setTesting(true);
    try {
      const result = await api.testChat(text, history);
      setTestMessages((prev) => [...prev, { role: 'bot', content: result.reply }]);
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } } };
      setTestMessages((prev) => [
        ...prev,
        { role: 'bot', content: 'Error: ' + (error.response?.data?.message || 'Failed to get a reply') },
      ]);
    } finally {
      setTesting(false);
    }
  };

  const openTestChat = () => {
    setTestModalOpen(true);
  };

  const resetTestChat = () => {
    setTestMessages([]);
    setTestInput('');
  };

  const closeTestChat = () => {
    setTestModalOpen(false);
  };

  const openEdit = (item: QAItem) => {
    setEditingItem(item);
    form.setFieldsValue(item);
    setModalOpen(true);
  };

  const openAdd = () => {
    setEditingItem(null);
    form.resetFields();
    setModalOpen(true);
  };

  const filteredItems = searchText
    ? qaItems.filter((item) => {
        const term = searchText.toLowerCase();
        return (
          item.question.toLowerCase().includes(term) ||
          item.answer.toLowerCase().includes(term) ||
          (item.category || '').toLowerCase().includes(term)
        );
      })
    : qaItems;

  const columns = [
    {
      title: 'Category',
      dataIndex: 'category',
      key: 'category',
      width: 110,
      ellipsis: true,
      render: (cat: string) => cat ? <Tag color="blue">{cat}</Tag> : <Tag>General</Tag>,
    },
    {
      title: 'Question',
      dataIndex: 'question',
      key: 'question',
      width: '35%',
      ellipsis: true,
    },
    {
      title: 'Answer',
      dataIndex: 'answer',
      key: 'answer',
      width: '35%',
      ellipsis: true,
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 90,
      render: (_: unknown, record: QAItem) => (
        <Space>
          <Button type="text" icon={<EditOutlined />} onClick={() => openEdit(record)} />
          <Popconfirm title="Delete this Q&A?" onConfirm={() => handleDelete(record.id)}>
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  if (loading) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="bots-page">
      <div className="bots-header">
        <div>
          <Title level={2}>AI Auto-Reply Bot</Title>
          <Text type="secondary">
            Manage your Q&A knowledge base. When auto-reply is enabled, incoming WhatsApp messages will be automatically answered using AI.
          </Text>
        </div>
      </div>

      {/* Settings Card */}
      <Card className="settings-card" title="Bot Settings">
        <div className="setting-row">
          <div className="setting-info">
            <Text strong>Auto-Reply</Text>
            <Text type="secondary">Automatically reply to incoming WhatsApp messages using your knowledge base</Text>
          </div>
          <Switch
            checked={kb?.auto_reply_enabled || false}
            onChange={handleAutoReplyToggle}
            checkedChildren="ON"
            unCheckedChildren="OFF"
          />
        </div>

        {kb?.last_synced_at && (
          <>
            <Divider />
            <Text type="secondary">Last synced: {new Date(kb.last_synced_at).toLocaleString()}</Text>
          </>
        )}
      </Card>

      {/* Q&A Items */}
      <Card
        className="qa-card"
        title={`Q&A Items (${searchText ? `${filteredItems.length} / ` : ''}${qaItems.length})`}
        extra={
          <Space>
            {qaItems.length > 0 && (
              <Popconfirm
                title="Delete all Q&A items?"
                description={`This will permanently delete all ${qaItems.length} items.`}
                onConfirm={handleDeleteAll}
                okText="Delete All"
                okButtonProps={{ danger: true }}
              >
                <Button danger icon={<DeleteOutlined />}>Delete All</Button>
              </Popconfirm>
            )}
            <Button icon={<DownloadOutlined />} onClick={handleDownloadTemplate}>
              Download Template
            </Button>
            <Upload
              beforeUpload={handleFileUpload}
              showUploadList={false}
              accept=".xls,.xlsx"
            >
              <Button icon={<UploadOutlined />}>Upload File</Button>
            </Upload>
            <Button
              icon={<RobotOutlined />}
              onClick={openTestChat}
            >
              Test Bot
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openAdd}>
              Add Q&A
            </Button>
          </Space>
        }
      >
        <Input
          placeholder="Search by category, question, or answer..."
          prefix={<SearchOutlined />}
          allowClear
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          style={{ marginBottom: 16 }}
        />
        <Table
          dataSource={filteredItems}
          columns={columns}
          rowKey="id"
          tableLayout="fixed"
          pagination={{ defaultPageSize: 10, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'] }}
          locale={{ emptyText: searchText ? 'No matching Q&A items found.' : 'No Q&A items yet. Add some questions and answers to get started!' }}
        />
      </Card>

      {/* Add/Edit Modal */}
      <Modal
        title={editingItem ? 'Edit Q&A Item' : 'Add Q&A Item'}
        open={modalOpen}
        onOk={handleAddEdit}
        onCancel={() => { setModalOpen(false); setEditingItem(null); form.resetFields(); }}
        okText={editingItem ? 'Update' : 'Add'}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="category" label="Category">
            <Input placeholder="e.g., Pricing, Returns, Shipping..." />
          </Form.Item>
          <Form.Item name="question" label="Question" rules={[{ required: true, message: 'Required' }]}>
            <Input.TextArea rows={2} placeholder="Customer's question..." />
          </Form.Item>
          <Form.Item name="answer" label="Answer" rules={[{ required: true, message: 'Required' }]}>
            <Input.TextArea rows={4} placeholder="Your answer..." />
          </Form.Item>
        </Form>
      </Modal>

      {/* Test Chat Modal — simulates a real customer conversation */}
      <Modal
        title={
          <Space>
            <RobotOutlined />
            <span>Test Bot — Customer Chat</span>
          </Space>
        }
        open={testModalOpen}
        onCancel={closeTestChat}
        footer={null}
        width={600}
      >
        <div className="test-chat">
          <div className="test-chat-toolbar">
            <Text type="secondary">
              Chat exactly like one of your customers would. The bot remembers the whole
              conversation, so you can ask follow-up questions.
            </Text>
            <Button
              size="small"
              icon={<ReloadOutlined />}
              onClick={resetTestChat}
              disabled={testing || testMessages.length === 0}
            >
              New chat
            </Button>
          </div>

          <div className="test-chat-window" ref={testScrollRef}>
            {testMessages.length === 0 && !testing && (
              <div className="test-chat-empty">
                <RobotOutlined className="test-chat-empty-icon" />
                <Text type="secondary">
                  Send a message to start the conversation. Try something a real customer
                  might ask, like “Hi, what products do you have?”
                </Text>
              </div>
            )}

            {testMessages.map((msg, idx) => (
              <div
                key={idx}
                className={`test-chat-row ${msg.role === 'user' ? 'is-user' : 'is-bot'}`}
              >
                <div className="test-chat-avatar">
                  {msg.role === 'user' ? <UserOutlined /> : <RobotOutlined />}
                </div>
                <div className="test-chat-bubble">{msg.content}</div>
              </div>
            ))}

            {testing && (
              <div className="test-chat-row is-bot">
                <div className="test-chat-avatar"><RobotOutlined /></div>
                <div className="test-chat-bubble test-chat-typing">
                  <span className="dot" />
                  <span className="dot" />
                  <span className="dot" />
                </div>
              </div>
            )}
          </div>

          <Space.Compact style={{ width: '100%', marginTop: 16 }}>
            <Input
              placeholder="Type a message as a customer..."
              value={testInput}
              onChange={(e) => setTestInput(e.target.value)}
              onPressEnter={handleSendTestMessage}
              disabled={testing}
            />
            <Button
              type="primary"
              icon={<SendOutlined />}
              onClick={handleSendTestMessage}
              loading={testing}
            >
              Send
            </Button>
          </Space.Compact>
        </div>
      </Modal>

      {/* Upload Progress Modal */}
      <Modal
        title={null}
        open={uploadModalOpen}
        footer={uploadStep === 'done' || uploadStep === 'error' ? (
          <Button type="primary" block onClick={closeUploadModal}>
            {uploadStep === 'done' ? 'Done' : 'Close'}
          </Button>
        ) : null}
        closable={false}
        centered
        width={420}
      >
        <div className="upload-progress-content">
          {uploadStep === 'uploading' && (
            <>
              <div className="upload-progress-icon uploading">
                <CloudUploadOutlined />
              </div>
              <Title level={4}>Uploading file...</Title>
              <Text type="secondary">{uploadFileName}</Text>
              <Progress
                percent={Math.min(Math.round(uploadProgress), 90)}
                status="active"
                strokeColor="#3b82f6"
                style={{ marginTop: 16 }}
              />
              <Text type="secondary" style={{ fontSize: 12, marginTop: 8 }}>Uploading file</Text>
            </>
          )}

          {uploadStep === 'analyzing' && (
            <>
              <div className="upload-progress-icon analyzing">
                <RobotOutlined />
              </div>
              <Title level={4}>Analyzing content...</Title>
              <Text type="secondary">{uploadFileName}</Text>
              <Progress
                percent={Math.round(uploadProgress)}
                status="active"
                strokeColor={{ '0%': '#3b82f6', '100%': '#22c55e' }}
                style={{ marginTop: 16 }}
              />
              <Text type="secondary" style={{ fontSize: 12, marginTop: 8 }}>
                AI is indexing your file for search
              </Text>
            </>
          )}

          {uploadStep === 'done' && (
            <>
              <div className="upload-progress-icon done">
                <CheckCircleOutlined />
              </div>
              <Title level={4}>Upload complete!</Title>
              <Text type="secondary">
                <Text strong>{uploadFileName}</Text> has been indexed.
                Your AI bot can now answer questions from this file.
              </Text>
            </>
          )}

          {uploadStep === 'error' && (
            <>
              <div className="upload-progress-icon error">
                <CloudUploadOutlined />
              </div>
              <Title level={4}>Upload failed</Title>
              <Text type="secondary">
                Failed to upload <Text strong>{uploadFileName}</Text>. Please try again.
              </Text>
            </>
          )}
        </div>
      </Modal>
    </div>
  );
}
