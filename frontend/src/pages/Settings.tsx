import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { Card, Button, Typography, Tag, Divider, Spin, message, InputNumber, Alert } from 'antd';
import {
  CreditCardOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
  MinusOutlined,
  PlusOutlined,
  WhatsAppOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../store/authStore';
import api from '../services/api';
import './Settings.css';

const { Title, Text } = Typography;

const UNIT_PRICE = 9.99;

export default function Settings() {
  const { user } = useAuthStore();
  const [searchParams, setSearchParams] = useSearchParams();
  const [subscription, setSubscription] = useState<Record<string, unknown> | null>(null);
  const [loading, setLoading] = useState(true);
  const [quantity, setQuantity] = useState(1);
  const [submitting, setSubmitting] = useState(false);
  const isExpired = searchParams.get('expired') === 'true';

  useEffect(() => {
    loadSubscription();
  }, []);

  useEffect(() => {
    if (isExpired && subscription) {
      const status = (subscription.status as string) || '';
      if (status === 'active' || status === 'trialing') {
        const next = new URLSearchParams(searchParams);
        next.delete('expired');
        setSearchParams(next, { replace: true });
      }
    }
  }, [isExpired, subscription, searchParams, setSearchParams]);

  const loadSubscription = async () => {
    try {
      const data = await api.getSubscription();
      setSubscription(data);
      setQuantity((data.max_devices as number) || 1);
    } catch {
      // No subscription
    } finally {
      setLoading(false);
    }
  };

  const handleManageBilling = async () => {
    try {
      const { portal_url } = await api.createPortal();
      window.location.href = portal_url;
    } catch {
      message.error('Failed to open billing portal');
    }
  };

  const handleCheckout = async () => {
    setSubmitting(true);
    try {
      const { checkout_url } = await api.createCheckout(quantity);
      window.location.href = checkout_url;
    } catch {
      message.error('Failed to start checkout');
    } finally {
      setSubmitting(false);
    }
  };

  const handleUpdateQuantity = async () => {
    const currentDevices = (subscription?.max_devices as number) || 1;
    if (quantity === currentDevices) {
      message.info('No changes to apply');
      return;
    }

    setSubmitting(true);
    try {
      const result = await api.updateQuantity(quantity);
      message.success(`Updated to ${result.max_devices} device${result.max_devices > 1 ? 's' : ''}. Proration applied.`);
      loadSubscription();
    } catch {
      message.error('Failed to update quantity');
    } finally {
      setSubmitting(false);
    }
  };

  const statusIcon = () => {
    switch (subscription?.status as string) {
      case 'active':
        return <CheckCircleOutlined style={{ color: '#22c55e' }} />;
      case 'trialing':
        return <ClockCircleOutlined style={{ color: '#3b82f6' }} />;
      default:
        return <ExclamationCircleOutlined style={{ color: '#f59e0b' }} />;
    }
  };

  const statusTag = () => {
    switch (subscription?.status as string) {
      case 'active':
        return <Tag color="success">Active</Tag>;
      case 'trialing':
        return <Tag color="processing">Trial</Tag>;
      case 'cancelled':
        return <Tag color="error">Cancelled</Tag>;
      default:
        return <Tag>No Plan</Tag>;
    }
  };

  if (loading) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  const plan = (subscription?.plan as string) || 'trial';
  const status = (subscription?.status as string) || '';
  const trialActive = subscription?.trial_active as boolean;
  const trialDaysLeft = (subscription?.trial_days_left as number) || 0;
  const maxDevices = (subscription?.max_devices as number) || 1;
  const hasActiveSub = status === 'active' || status === 'trialing';
  const needsNewSub = plan === 'trial' || plan === 'cancelled' || !hasActiveSub;

  return (
    <div className="settings-page">
      <div className="settings-header">
        <Title level={2}>Settings</Title>
      </div>

      {isExpired && !hasActiveSub && (
        <Alert
          message="Trial Expired"
          description="Your 7-day free trial has ended. Please subscribe below to continue using AIChatsHub."
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      {/* Account Info */}
      <Card title="Account" className="settings-card">
        <div className="setting-item">
          <Text type="secondary">Name</Text>
          <Text strong>{user?.name}</Text>
        </div>
        <div className="setting-item">
          <Text type="secondary">Email</Text>
          <Text strong>{user?.email}</Text>
        </div>
      </Card>

      {/* Subscription */}
      <Card
        title={<span>{statusIcon()} Subscription</span>}
        className="settings-card"
        extra={statusTag()}
      >
        {trialActive && (
          <div className="trial-banner">
            <ClockCircleOutlined />
            <div>
              <Text strong>Free Trial</Text>
              <Text type="secondary"> - {trialDaysLeft} days remaining</Text>
            </div>
          </div>
        )}

        <div className="setting-item">
          <Text type="secondary">Current Plan</Text>
          <Text strong style={{ textTransform: 'capitalize' }}>{plan}</Text>
        </div>
        <div className="setting-item">
          <Text type="secondary">WhatsApp Devices</Text>
          <Text strong>{maxDevices}</Text>
        </div>
        <div className="setting-item">
          <Text type="secondary">Monthly Cost</Text>
          <Text strong>${(maxDevices * UNIT_PRICE).toFixed(2)}/mo</Text>
        </div>

        <Divider />

        {needsNewSub ? (
          <div className="purchase-section">
            <div className="purchase-header">
              <WhatsAppOutlined style={{ fontSize: 20, color: '#25D366' }} />
              <Text strong style={{ fontSize: 16 }}>Subscribe to AIChatsHub</Text>
            </div>

            <div className="quantity-selector">
              <Text type="secondary">WhatsApp Devices</Text>
              <div className="quantity-controls">
                <Button
                  icon={<MinusOutlined />}
                  shape="circle"
                  size="small"
                  onClick={() => setQuantity(Math.max(1, quantity - 1))}
                  disabled={quantity <= 1}
                />
                <InputNumber
                  min={1}
                  max={100}
                  value={quantity}
                  onChange={(v) => setQuantity(v || 1)}
                  className="quantity-input"
                  controls={false}
                />
                <Button
                  icon={<PlusOutlined />}
                  shape="circle"
                  size="small"
                  onClick={() => setQuantity(Math.min(100, quantity + 1))}
                />
              </div>
            </div>

            <div className="price-summary">
              <div className="price-row">
                <Text type="secondary">{quantity} device{quantity > 1 ? 's' : ''} x ${UNIT_PRICE}/mo</Text>
                <Text strong style={{ fontSize: 20 }}>
                  ${(quantity * UNIT_PRICE).toFixed(2)}
                  <span style={{ fontSize: 14, fontWeight: 400, color: '#9ca3af' }}>/mo</span>
                </Text>
              </div>
            </div>

            <Button
              type="primary"
              size="large"
              block
              loading={submitting}
              onClick={handleCheckout}
            >
              Subscribe — ${(quantity * UNIT_PRICE).toFixed(2)}/month
            </Button>
            <div style={{ textAlign: 'center', marginTop: 8 }}>
              <Text type="secondary" style={{ fontSize: 12 }}>7-day free trial included. Cancel anytime.</Text>
            </div>
          </div>
        ) : (
          <div className="purchase-section">
            <div className="purchase-header">
              <WhatsAppOutlined style={{ fontSize: 20, color: '#25D366' }} />
              <Text strong style={{ fontSize: 16 }}>Change Device Quantity</Text>
            </div>

            <div className="quantity-selector">
              <Text type="secondary">WhatsApp Devices</Text>
              <div className="quantity-controls">
                <Button
                  icon={<MinusOutlined />}
                  shape="circle"
                  size="small"
                  onClick={() => setQuantity(Math.max(1, quantity - 1))}
                  disabled={quantity <= 1}
                />
                <InputNumber
                  min={1}
                  max={100}
                  value={quantity}
                  onChange={(v) => setQuantity(v || 1)}
                  className="quantity-input"
                  controls={false}
                />
                <Button
                  icon={<PlusOutlined />}
                  shape="circle"
                  size="small"
                  onClick={() => setQuantity(Math.min(100, quantity + 1))}
                />
              </div>
            </div>

            <div className="price-summary">
              <div className="price-row">
                <Text type="secondary">{quantity} device{quantity > 1 ? 's' : ''} x ${UNIT_PRICE}/mo</Text>
                <Text strong style={{ fontSize: 20 }}>
                  ${(quantity * UNIT_PRICE).toFixed(2)}
                  <span style={{ fontSize: 14, fontWeight: 400, color: '#9ca3af' }}>/mo</span>
                </Text>
              </div>
              {quantity !== maxDevices && (
                <div className="price-change">
                  {quantity > maxDevices ? (
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      Adding {quantity - maxDevices} device{quantity - maxDevices > 1 ? 's' : ''}. Prorated charge will apply for the remaining billing period.
                    </Text>
                  ) : (
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      Removing {maxDevices - quantity} device{maxDevices - quantity > 1 ? 's' : ''}. Credit will apply to your next invoice.
                    </Text>
                  )}
                </div>
              )}
            </div>

            <div className="action-buttons">
              <Button
                type="primary"
                size="large"
                loading={submitting}
                onClick={handleUpdateQuantity}
                disabled={quantity === maxDevices}
              >
                {quantity === maxDevices ? 'No Changes' : `Update to ${quantity} Device${quantity > 1 ? 's' : ''}`}
              </Button>
              <Button icon={<CreditCardOutlined />} onClick={handleManageBilling}>
                Manage Billing
              </Button>
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
