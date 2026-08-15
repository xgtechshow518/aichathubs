import { useEffect, useState, useCallback } from 'react';
import { Card, Button, Tag, Typography, Modal, Spin, message, Empty } from 'antd';
import {
  WhatsAppOutlined,
  PlusOutlined,
  DisconnectOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { WhatsAppDevice } from '../types';
import { useAuthStore } from '../store/authStore';
import api from '../services/api';
import wsService from '../services/websocket';
import './Integrations.css';

const { Title, Text } = Typography;

export default function Integrations() {
  const { user } = useAuthStore();
  const [devices, setDevices] = useState<WhatsAppDevice[]>([]);
  const [loading, setLoading] = useState(true);
  const [qrImage, setQrImage] = useState<string>('');
  const [qrModalOpen, setQrModalOpen] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [disconnectingId, setDisconnectingId] = useState<number | null>(null);

  const loadDevices = useCallback(async () => {
    try {
      const { devices: devs } = await api.getWhatsAppDevices();
      setDevices(devs || []);
    } catch { /* ignore */ } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadDevices();

    const unsubQR = wsService.onWhatsAppQR((img) => {
      setQrImage(img);
      setConnecting(false);
    });

    const unsubStatus = wsService.onWhatsAppStatus((payload) => {
      if (payload.status === 'connected') {
        setQrModalOpen(false);
        setQrImage('');
        message.success('WhatsApp device connected!');
        loadDevices();
      } else if (payload.status === 'error') {
        setQrModalOpen(false);
        setQrImage('');
        setConnecting(false);
        message.error(payload.message || 'Connection failed');
        loadDevices();
      } else if (payload.status === 'timeout') {
        setQrImage('');
        message.warning('QR code expired. Please try again.');
      } else if (payload.status === 'disconnected') {
        loadDevices();
      }
    });

    return () => {
      unsubQR();
      unsubStatus();
    };
  }, [loadDevices]);

  const handleConnect = async () => {
    setConnecting(true);
    setQrImage('');
    setQrModalOpen(true);
    try {
      await api.connectWhatsApp();
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } } };
      message.error(error.response?.data?.message || 'Failed to connect');
      setConnecting(false);
      setQrModalOpen(false);
    }
  };

  const handleDisconnect = async (deviceId: number) => {
    setDisconnectingId(deviceId);
    try {
      await api.disconnectWhatsAppDevice(deviceId);
      message.success('Device disconnected');
      loadDevices();
    } catch {
      message.error('Failed to disconnect');
    } finally {
      setDisconnectingId(null);
    }
  };

  const connectedCount = devices.filter((d) => d.status === 'connected').length;
  const maxDevices = user?.max_devices ?? 1;
  const canAddMore = connectedCount < maxDevices;

  return (
    <div className="integrations-page">
      <div className="integrations-header">
        <Title level={2}>Integrations</Title>
        <Text type="secondary">Connect your messaging platforms to start receiving messages.</Text>
      </div>

      <Card
        className="integration-card"
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <div className="integration-icon whatsapp-icon">
              <WhatsAppOutlined />
            </div>
            <div>
              <Title level={4} style={{ margin: 0 }}>WhatsApp Devices</Title>
              <Text type="secondary">{connectedCount} / {maxDevices} connected</Text>
            </div>
          </div>
        }
        extra={
          <div style={{ display: 'flex', gap: 8 }}>
            <Button icon={<ReloadOutlined />} onClick={loadDevices}>Refresh</Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={handleConnect}
              disabled={!canAddMore}
              loading={connecting}
              style={canAddMore ? { background: '#25D366', borderColor: '#25D366' } : undefined}
            >
              Connect Device
            </Button>
          </div>
        }
      >
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : devices.length === 0 ? (
          <Empty
            description="No WhatsApp devices connected yet"
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          >
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={handleConnect}
              style={{ background: '#25D366', borderColor: '#25D366' }}
            >
              Connect Your First Device
            </Button>
          </Empty>
        ) : (
          <div className="devices-list">
            {devices.map((device) => (
              <div key={device.id} className="device-row">
                <div className="device-info">
                  <div className="device-icon">
                    <WhatsAppOutlined style={{ color: '#25D366', fontSize: 20 }} />
                  </div>
                  <div>
                    <Text strong>{device.push_name || 'WhatsApp Device'}</Text>
                    <div>
                      <Text type="secondary" style={{ fontSize: 13 }}>
                        {device.phone ? `+${device.phone}` : 'No phone'}
                      </Text>
                    </div>
                  </div>
                </div>
                <div className="device-actions">
                  <Tag color={device.status === 'connected' ? 'success' : 'default'}>
                    {device.status === 'connected' ? 'Connected' : 'Disconnected'}
                  </Tag>
                  <Button
                    danger
                    size="small"
                    icon={<DisconnectOutlined />}
                    loading={disconnectingId === device.id}
                    onClick={() => handleDisconnect(device.id)}
                  >
                    Remove
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}

        {!canAddMore && devices.length > 0 && (
          <div style={{ marginTop: 16, textAlign: 'center' }}>
            <Text type="secondary">
              You've reached your device limit ({maxDevices}). Upgrade your plan to connect more devices.
            </Text>
          </div>
        )}
      </Card>

      <Modal
        title="Scan QR Code with WhatsApp"
        open={qrModalOpen}
        onCancel={() => {
          setQrModalOpen(false);
          setQrImage('');
        }}
        footer={null}
        centered
        width={440}
      >
        <div className="qr-modal-content">
          <div className="qr-instructions">
            <Text>1. Open WhatsApp on your phone</Text>
            <Text>2. Tap <Text strong>Menu</Text> or <Text strong>Settings</Text> and select <Text strong>Linked Devices</Text></Text>
            <Text>3. Tap on <Text strong>Link a device</Text></Text>
            <Text>4. Point your phone at this screen to scan the QR code</Text>
          </div>

          <div className="qr-code-container">
            {qrImage ? (
              <img src={qrImage} alt="WhatsApp QR Code" className="qr-code-image" />
            ) : (
              <div className="qr-loading">
                <Spin size="large" />
                <Text type="secondary">Generating QR code...</Text>
              </div>
            )}
          </div>
        </div>
      </Modal>
    </div>
  );
}
