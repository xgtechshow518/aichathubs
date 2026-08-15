import { useState, useEffect, useRef } from 'react';
import { useNavigate, useSearchParams, Link as RouterLink } from 'react-router-dom';
import { App, Button, Card, Divider, Form, Input, Spin, Typography } from 'antd';
import { GoogleOutlined, MailOutlined, LockOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/authStore';
import api from '../services/api';
import './Login.css';

const { Title, Text, Link } = Typography;

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID || '';

export default function Login() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { login, isAuthenticated, isLoading } = useAuthStore();
  const [emailLoading, setEmailLoading] = useState(false);
  // Only show the Google button when the server has OAuth configured AND the
  // frontend has a client ID. Hidden by default to avoid a broken button flash.
  const [showGoogle, setShowGoogle] = useState(false);
  const { message } = App.useApp();

  useEffect(() => {
    api.getPublicConfig()
      .then((cfg) => setShowGoogle(cfg.googleAuth && !!GOOGLE_CLIENT_ID))
      // Fail open: if the config call can't be reached, fall back to the
      // build-time client ID so a working Google login isn't hidden by a
      // transient fetch error (the backend still 503s if truly unconfigured).
      .catch(() => setShowGoogle(!!GOOGLE_CLIENT_ID));
  }, []);

  // StrictMode (dev) mounts effects twice. OAuth codes are single-use on
  // Google's side, but they have a short grace window where both calls can
  // succeed — causing "Login successful!" to appear twice. Guard with a ref
  // so the callback is handled exactly once per code.
  const oauthHandledRef = useRef(false);

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard');
    }
  }, [isAuthenticated, navigate]);

  useEffect(() => {
    const code = searchParams.get('code');
    const state = searchParams.get('state');
    const error = searchParams.get('error');

    if (error) {
      message.error('Login failed. Please try again.');
      return;
    }

    if (code && !oauthHandledRef.current) {
      oauthHandledRef.current = true;
      handleOAuthCallback(code, state);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const handleOAuthCallback = async (code: string, state: string | null) => {
    // Strip OAuth params from the URL immediately so a back-nav / refresh
    // can't re-trigger the exchange (also avoids leaking the code in history).
    setSearchParams({}, { replace: true });

    try {
      const redirectUri = `${window.location.origin}/login`;
      let response;

      if (state === 'google') {
        response = await api.googleCallback(code, redirectUri);
      } else if (state === 'facebook') {
        response = await api.facebookCallback(code, redirectUri);
      } else {
        response = await api.googleCallback(code, redirectUri);
      }

      login(response.token, response.user);
      message.success('Login successful!');
      navigate('/dashboard');
    } catch (error) {
      console.error('OAuth callback error:', error);
      message.error('Login failed. Please try again.');
      navigate('/login', { replace: true });
    }
  };

  const handleGoogleLogin = () => {
    const redirectUri = encodeURIComponent(`${window.location.origin}/login`);
    const scope = encodeURIComponent('email profile');
    const url = `https://accounts.google.com/o/oauth2/v2/auth?client_id=${GOOGLE_CLIENT_ID}&redirect_uri=${redirectUri}&response_type=code&scope=${scope}&state=google&access_type=offline&prompt=consent`;
    window.location.href = url;
  };

  const handleEmailLogin = async (values: { email: string; password: string }) => {
    setEmailLoading(true);
    try {
      const response = await api.emailLogin(values.email, values.password);
      login(response.token, response.user);
      message.success('Login successful!');
      navigate('/dashboard');
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } } };
      const msg = error.response?.data?.message || 'Login failed';
      if (msg.includes('verify your email')) {
        message.warning(msg);
        navigate(`/verify-email?email=${encodeURIComponent(values.email)}`);
      } else {
        message.error(msg);
      }
    } finally {
      setEmailLoading(false);
    }
  };

  if (isLoading) {
    return (
      <div className="login-container">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="login-container">
      <div className="login-branding">
        <div className="branding-content">
          <div className="branding-logo">
            <span className="logo-icon">💬</span>
            <span className="logo-text">AIChatsHub</span>
          </div>
          <div className="branding-illustration">
            <div className="chat-preview">
              <div className="chat-window">
                <div className="chat-header">Chats</div>
                <div className="chat-list">
                  {['Chris', 'Benjamin', 'James', 'Ava', 'Olivia'].map((name, i) => (
                    <div key={name} className="chat-item">
                      <div className="avatar">{name[0]}</div>
                      <div className="chat-info">
                        <div className="chat-name">{name}</div>
                        <div className="chat-preview-text">Hello, how can I help you?</div>
                      </div>
                      {i < 2 && <div className="unread-badge">{i + 1}</div>}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
          <h2 className="branding-tagline">
            Multi-platform Customer Engagement
          </h2>
          <div className="platform-icons">
            <span className="platform-icon" title="WhatsApp">
              <svg viewBox="0 0 24 24" width="28" height="28" fill="currentColor">
                <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
              </svg>
            </span>
            <span className="platform-icon" title="Telegram">
              <svg viewBox="0 0 24 24" width="28" height="28" fill="currentColor">
                <path d="M11.944 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 0 1 .171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.479.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z"/>
              </svg>
            </span>
            <span className="platform-icon" title="Instagram">
              <svg viewBox="0 0 24 24" width="28" height="28" fill="currentColor">
                <path d="M12 0C8.74 0 8.333.015 7.053.072 5.775.132 4.905.333 4.14.63c-.789.306-1.459.717-2.126 1.384S.935 3.35.63 4.14C.333 4.905.131 5.775.072 7.053.012 8.333 0 8.74 0 12s.015 3.667.072 4.947c.06 1.277.261 2.148.558 2.913.306.788.717 1.459 1.384 2.126.667.666 1.336 1.079 2.126 1.384.766.296 1.636.499 2.913.558C8.333 23.988 8.74 24 12 24s3.667-.015 4.947-.072c1.277-.06 2.148-.262 2.913-.558.788-.306 1.459-.718 2.126-1.384.666-.667 1.079-1.335 1.384-2.126.296-.765.499-1.636.558-2.913.06-1.28.072-1.687.072-4.947s-.015-3.667-.072-4.947c-.06-1.277-.262-2.149-.558-2.913-.306-.789-.718-1.459-1.384-2.126C21.319 1.347 20.651.935 19.86.63c-.765-.297-1.636-.499-2.913-.558C15.667.012 15.26 0 12 0zm0 2.16c3.203 0 3.585.016 4.85.071 1.17.055 1.805.249 2.227.415.562.217.96.477 1.382.896.419.42.679.819.896 1.381.164.422.36 1.057.413 2.227.057 1.266.07 1.646.07 4.85s-.015 3.585-.074 4.85c-.061 1.17-.256 1.805-.421 2.227-.224.562-.479.96-.899 1.382-.419.419-.824.679-1.38.896-.42.164-1.065.36-2.235.413-1.274.057-1.649.07-4.859.07-3.211 0-3.586-.015-4.859-.074-1.171-.061-1.816-.256-2.236-.421-.569-.224-.96-.479-1.379-.899-.421-.419-.69-.824-.9-1.38-.165-.42-.359-1.065-.42-2.235-.045-1.26-.061-1.649-.061-4.844 0-3.196.016-3.586.061-4.861.061-1.17.255-1.814.42-2.234.21-.57.479-.96.9-1.381.419-.419.81-.689 1.379-.898.42-.166 1.051-.361 2.221-.421 1.275-.045 1.65-.06 4.859-.06l.045.03zm0 3.678a6.162 6.162 0 1 0 0 12.324 6.162 6.162 0 1 0 0-12.324zM12 16c-2.21 0-4-1.79-4-4s1.79-4 4-4 4 1.79 4 4-1.79 4-4 4zm7.846-10.405a1.441 1.441 0 1 1-2.882 0 1.441 1.441 0 0 1 2.882 0z"/>
              </svg>
            </span>
            <span className="platform-icon" title="Messenger">
              <svg viewBox="0 0 24 24" width="28" height="28" fill="currentColor">
                <path d="M.001 11.639C.001 4.949 5.241 0 12.001 0S24 4.95 24 11.639c0 6.689-5.24 11.638-12 11.638-1.21 0-2.38-.16-3.47-.46a.96.96 0 00-.64.05l-2.39 1.05a.96.96 0 01-1.35-.85l-.07-2.14a.97.97 0 00-.32-.68A11.39 11.389 0 01.002 11.64zm8.32-2.19l-3.52 5.6c-.35.53.32 1.139.82.75l3.79-2.87c.26-.2.6-.2.87-.01l2.78 2.09c.8.6 1.96.4 2.49-.45l3.52-5.6c.35-.53-.32-1.13-.82-.75l-3.79 2.87c-.25.2-.6.2-.86.01l-2.79-2.09c-.8-.6-1.96-.4-2.48.45z"/>
              </svg>
            </span>
          </div>
        </div>
      </div>

      <div className="login-form-container">
        <Card className="login-card" bordered={false}>
          <div className="login-header">
            <Title level={2} className="login-title">Welcome Back</Title>
            <Text type="secondary">
              Don't have an account?{' '}
              <RouterLink to="/register"><Link>Sign up now</Link></RouterLink>
            </Text>
          </div>

          <div className="login-form">
            <Form layout="vertical" onFinish={handleEmailLogin} size="large" requiredMark={false}>
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
                rules={[{ required: true, message: 'Please enter your password' }]}
              >
                <Input.Password prefix={<LockOutlined />} placeholder="Password" />
              </Form.Item>

              <Form.Item>
                <Button type="primary" htmlType="submit" loading={emailLoading} block className="social-button">
                  Sign In
                </Button>
              </Form.Item>
            </Form>

            {showGoogle && (
              <>
                <Divider plain>
                  <Text type="secondary">or continue with</Text>
                </Divider>

                <div className="social-buttons">
                  <Button
                    size="large"
                    icon={<GoogleOutlined />}
                    onClick={handleGoogleLogin}
                    className="social-button google-button"
                    block
                  >
                    Continue with Google
                  </Button>
                </div>
              </>
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}

