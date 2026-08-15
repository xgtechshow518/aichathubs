import { useEffect, useState } from 'react';
import {
  Card, Button, Typography, Space, Spin, Alert, InputNumber, Tag, Popconfirm, message, Row, Col, Statistic,
} from 'antd';
import {
  SaveOutlined, UndoOutlined, ReloadOutlined, ClockCircleOutlined, HourglassOutlined,
} from '@ant-design/icons';
import api from '../../services/api';

const { Title, Text, Paragraph } = Typography;

interface DelayData {
  min_seconds: number;
  max_seconds: number;
  default_min_seconds: number;
  default_max_seconds: number;
  is_custom: boolean;
  updated_at?: string;
  updated_by?: string;
}

export default function AdminAIReplyDelay() {
  const [data, setData] = useState<DelayData | null>(null);
  const [minSeconds, setMinSeconds] = useState<number>(2);
  const [maxSeconds, setMaxSeconds] = useState<number>(30);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const applyData = (d: DelayData) => {
    setData(d);
    setMinSeconds(d.min_seconds);
    setMaxSeconds(d.max_seconds);
  };

  const load = async () => {
    setLoading(true);
    try {
      const result = await api.getAdminAIReplyDelay();
      applyData(result);
    } catch {
      message.error('Failed to load AI reply delay settings');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const handleSave = async () => {
    if (minSeconds < 0 || maxSeconds < 0) {
      message.error('Delay values must be non-negative');
      return;
    }
    if (minSeconds > maxSeconds) {
      message.error('Minimum delay must be less than or equal to maximum delay');
      return;
    }
    setSaving(true);
    try {
      const result = await api.updateAdminAIReplyDelay(minSeconds, maxSeconds);
      applyData({ ...result, updated_at: new Date().toISOString() });
      message.success('AI reply delay updated. Changes apply to new replies within ~30 seconds.');
    } catch {
      message.error('Failed to save AI reply delay');
    } finally {
      setSaving(false);
    }
  };

  const handleReset = async () => {
    setSaving(true);
    try {
      const result = await api.resetAdminAIReplyDelay();
      applyData(result);
      message.success('Reverted to default delay range');
    } catch {
      message.error('Failed to reset AI reply delay');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 100 }}><Spin size="large" /></div>;
  }
  if (!data) return null;

  const isDirty = minSeconds !== data.min_seconds || maxSeconds !== data.max_seconds;
  const invalidRange = minSeconds > maxSeconds;

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>
            <HourglassOutlined /> AI Reply Delay
          </Title>
          <Text type="secondary">
            Add a randomized "typing" delay before the AI sends its reply, so automated responses feel more human.
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
              Before sending each AI auto-reply, the server sleeps a random amount between <Text strong>min</Text> and <Text strong>max</Text> seconds.
            </Text>
            <Text type="secondary">
              Applies platform-wide to every user's WhatsApp auto-replies. Changes take effect within ~30 seconds.
            </Text>
          </Space>
        }
      />

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8}>
          <Card size="small">
            <Statistic
              title="Current minimum"
              value={data.min_seconds}
              suffix="sec"
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card size="small">
            <Statistic
              title="Current maximum"
              value={data.max_seconds}
              suffix="sec"
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card size="small">
            <Statistic
              title="Average delay"
              value={((data.min_seconds + data.max_seconds) / 2).toFixed(1)}
              suffix="sec"
            />
          </Card>
        </Col>
      </Row>

      <Card
        title={
          <Space>
            <span>Edit delay range</span>
            {data.is_custom
              ? <Tag color="blue">Custom (admin-managed)</Tag>
              : <Tag>Default ({data.default_min_seconds}–{data.default_max_seconds}s)</Tag>}
          </Space>
        }
        extra={
          data.updated_at && data.is_custom ? (
            <Text type="secondary" style={{ fontSize: 12 }}>
              Last updated {new Date(data.updated_at).toLocaleString()}
              {data.updated_by ? ` by ${data.updated_by}` : ''}
            </Text>
          ) : null
        }
      >
        <Paragraph type="secondary" style={{ marginBottom: 16 }}>
          Recommended range: <Text code>2–30 seconds</Text>. Setting both values to 0 disables the delay entirely. Maximum allowed is 600 seconds.
        </Paragraph>

        <Row gutter={16}>
          <Col xs={24} sm={12}>
            <div style={{ marginBottom: 8 }}>
              <Text strong>Minimum delay (seconds)</Text>
            </div>
            <InputNumber
              min={0}
              max={600}
              value={minSeconds}
              onChange={(v) => setMinSeconds(Number(v ?? 0))}
              style={{ width: '100%' }}
              addonAfter="sec"
            />
          </Col>
          <Col xs={24} sm={12}>
            <div style={{ marginBottom: 8 }}>
              <Text strong>Maximum delay (seconds)</Text>
            </div>
            <InputNumber
              min={0}
              max={600}
              value={maxSeconds}
              onChange={(v) => setMaxSeconds(Number(v ?? 0))}
              style={{ width: '100%' }}
              addonAfter="sec"
            />
          </Col>
        </Row>

        {invalidRange && (
          <Alert
            type="error"
            showIcon
            style={{ marginTop: 16 }}
            message="Minimum delay cannot be greater than maximum delay"
          />
        )}

        <Space style={{ marginTop: 20, width: '100%', justifyContent: 'space-between' }}>
          <Space>
            {data.is_custom && (
              <Popconfirm
                title="Reset to default delay range?"
                description={`This clears the admin override and reverts to ${data.default_min_seconds}–${data.default_max_seconds} seconds.`}
                onConfirm={handleReset}
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
              onClick={() => {
                setMinSeconds(data.min_seconds);
                setMaxSeconds(data.max_seconds);
              }}
            >
              Discard changes
            </Button>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={saving}
              disabled={!isDirty || invalidRange}
              onClick={handleSave}
            >
              Save
            </Button>
          </Space>
        </Space>
      </Card>
    </div>
  );
}
