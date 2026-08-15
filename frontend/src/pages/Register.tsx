import { useState, useEffect } from 'react';
import { useNavigate, Link as RouterLink } from 'react-router-dom';
import { Button, Card, Form, Input, message, Typography } from 'antd';
import { MailOutlined, LockOutlined, UserOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/authStore';
import api from '../services/api';
import './Login.css';

const { Title, Text, Link } = Typography;

export default function Register() {
  const navigate = useNavigate();
  const { isAuthenticated, login } = useAuthStore();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard');
    }
  }, [isAuthenticated, navigate]);

  const onFinish = async (values: { name: string; email: string; password: string }) => {
    setLoading(true);
    try {
      const res = await api.register(values.email, values.password, values.name);
      // No SMTP on the server → account is auto-verified and we're logged in.
      if (res.token && res.user) {
        login(res.token, res.user);
        message.success('Account created!');
        navigate('/dashboard');
        return;
      }
      message.success('Verification code sent to your email!');
      navigate(`/verify-email?email=${encodeURIComponent(values.email)}`);
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } } };
      message.error(error.response?.data?.message || 'Registration failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <div className="login-branding">
        <div className="branding-content">
          <div className="branding-logo">
            <span className="logo-icon">💬</span>
            <span className="logo-text">AIChatsHub</span>
          </div>
          <h2 className="branding-tagline">
            Create your account and start<br />engaging with customers
          </h2>
        </div>
      </div>

      <div className="login-form-container">
        <Card className="login-card" bordered={false}>
          <div className="login-header">
            <Title level={2} className="login-title">Create Account</Title>
            <Text type="secondary">
              Already have an account?{' '}
              <RouterLink to="/login"><Link>Sign in</Link></RouterLink>
            </Text>
          </div>

          <Form layout="vertical" onFinish={onFinish} size="large" requiredMark={false}>
            <Form.Item name="name" rules={[{ required: true, message: 'Please enter your name' }]}>
              <Input prefix={<UserOutlined />} placeholder="Full name" />
            </Form.Item>

            <Form.Item
              name="email"
              rules={[
                { required: true, message: 'Please enter your email' },
                { type: 'email', message: 'Please enter a valid email' },
              ]}
            >
              <Input prefix={<MailOutlined />} placeholder="Email address" />
            </Form.Item>

            <Form.Item
              name="password"
              rules={[
                { required: true, message: 'Please enter a password' },
                { min: 8, message: 'Password must be at least 8 characters' },
              ]}
            >
              <Input.Password prefix={<LockOutlined />} placeholder="Password (min 8 characters)" />
            </Form.Item>

            <Form.Item
              name="confirmPassword"
              dependencies={['password']}
              rules={[
                { required: true, message: 'Please confirm your password' },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue('password') === value) {
                      return Promise.resolve();
                    }
                    return Promise.reject(new Error('Passwords do not match'));
                  },
                }),
              ]}
            >
              <Input.Password prefix={<LockOutlined />} placeholder="Confirm password" />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading} block className="social-button">
                Sign Up
              </Button>
            </Form.Item>
          </Form>
        </Card>
      </div>
    </div>
  );
}
