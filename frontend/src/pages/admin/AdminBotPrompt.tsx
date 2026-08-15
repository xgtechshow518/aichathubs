import { useEffect, useState } from 'react';
import {
  Card, Button, Input, Typography, Space, Spin, Alert, message, Tag, Popconfirm,
} from 'antd';
import {
  SaveOutlined, UndoOutlined, ReloadOutlined, RobotOutlined, ThunderboltOutlined,
} from '@ant-design/icons';
import api from '../../services/api';

const { Title, Text, Paragraph } = Typography;

interface PromptData {
  value: string;
  default: string;
  is_custom: boolean;
  updated_at?: string;
  updated_by?: string;
}

const SALES_TEMPLATE =
  `You are Maya, a friendly sales associate chatting on WhatsApp.

# Voice
- Warm, casual, concise. 1–3 short sentences per message.
- Match the customer's language.
- Never sound like a brochure.
- Emojis only if the customer uses them first.

# Sales playbook
1. GREET briefly the first time.
2. DISCOVER with one clarifying question if needed.
3. RECOMMEND via search_products. Lead with the BEST match and one-line reason.
4. SHOW via send_product_image when the customer shows interest.
5. HANDLE OBJECTIONS honestly. Offer a cheaper alternative if appropriate.
6. CLOSE via capture_lead when the customer signals intent.

# Hard rules
- NEVER invent SKUs, prices, stock, or discounts. Always call a tool first.
- NEVER promise delivery dates or refunds you can't verify.
- Outside scope or upset customer → say a teammate will follow up. Stop.`;

export default function AdminBotPrompt() {
  const [data, setData] = useState<PromptData | null>(null);
  const [value, setValue] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const result = await api.getAdminBotPrompt();
      setData(result);
      setValue(result.is_custom ? result.value : '');
    } catch {
      message.error('Failed to load system prompt');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      const result = await api.updateAdminBotPrompt(value);
      setData(result);
      message.success(
        value.trim()
          ? 'Prompt saved. New prompt will apply to all users on their next message.'
          : 'Custom prompt cleared. Reverted to the default prompt.'
      );
    } catch {
      message.error('Failed to save system prompt');
    } finally {
      setSaving(false);
    }
  };

  const handleResetToDefault = async () => {
    setSaving(true);
    try {
      const result = await api.updateAdminBotPrompt('');
      setData(result);
      setValue('');
      message.success('Reverted to default prompt');
    } catch {
      message.error('Failed to reset prompt');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 100 }}><Spin size="large" /></div>;
  }
  if (!data) return null;

  const currentEffective = data.is_custom ? data.value : data.default;
  const isDirty = value.trim() !== (data.is_custom ? data.value : '').trim();

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>
            <RobotOutlined /> AI Bot System Prompt
          </Title>
          <Text type="secondary">
            This prompt controls how the AI sales bot talks to every customer across all users on the platform.
          </Text>
        </div>
        <Button icon={<ReloadOutlined />} onClick={load}>Reload</Button>
      </div>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="Global setting"
        description={
          <Space direction="vertical" size={4}>
            <Text>
              Saving applies to <Text strong>every user</Text> on the platform. Individual users cannot override this.
            </Text>
            <Text type="secondary">
              Changes take effect within ~30 seconds for new conversations. All cached Gemini contexts are invalidated on save so your new prompt is used immediately.
            </Text>
          </Space>
        }
      />

      <Card
        title={
          <Space>
            <span>Current Prompt</span>
            {data.is_custom
              ? <Tag color="blue">Custom (admin-managed)</Tag>
              : <Tag>Default (from env)</Tag>}
          </Space>
        }
        extra={
          <Space>
            {data.updated_at && data.is_custom && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                Last updated {new Date(data.updated_at).toLocaleString()}
                {data.updated_by ? ` by ${data.updated_by}` : ''}
              </Text>
            )}
          </Space>
        }
      >
        <Paragraph type="secondary" style={{ marginBottom: 12 }}>
          Leave empty and save to fall back to the default prompt from the server's <Text code>GEMINI_SYSTEM_PROMPT</Text> env var.
        </Paragraph>

        <Input.TextArea
          rows={14}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={data.default}
          style={{ fontFamily: 'monospace', fontSize: 13 }}
        />

        <Space style={{ marginTop: 16, width: '100%', justifyContent: 'space-between' }}>
          <Space>
            <Button
              icon={<ThunderboltOutlined />}
              onClick={() => setValue(SALES_TEMPLATE)}
              disabled={saving}
            >
              Load sales persona template
            </Button>
            {data.is_custom && (
              <Popconfirm
                title="Reset to default prompt?"
                description="This clears the admin-managed prompt and reverts to the env default for every user."
                onConfirm={handleResetToDefault}
                okText="Reset"
                okButtonProps={{ danger: true }}
              >
                <Button danger icon={<UndoOutlined />} disabled={saving}>
                  Reset to default
                </Button>
              </Popconfirm>
            )}
          </Space>

          <Space>
            <Button
              disabled={!isDirty || saving}
              onClick={() => setValue(data.is_custom ? data.value : '')}
            >
              Discard changes
            </Button>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={saving}
              disabled={!isDirty}
              onClick={handleSave}
            >
              Save for all users
            </Button>
          </Space>
        </Space>
      </Card>

      <Card title="Effective Prompt (currently used by the bot)" style={{ marginTop: 16 }} size="small">
        <pre style={{
          margin: 0,
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          fontSize: 12,
          fontFamily: 'monospace',
          background: '#fafafa',
          padding: 12,
          borderRadius: 4,
          maxHeight: 300,
          overflow: 'auto',
        }}>
          {currentEffective}
        </pre>
      </Card>
    </div>
  );
}
