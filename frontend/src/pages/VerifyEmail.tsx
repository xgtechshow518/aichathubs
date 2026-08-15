import { useState, useEffect, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Button, Card, message, Typography } from 'antd';
import { useAuthStore } from '../store/authStore';
import api from '../services/api';
import './Login.css';
import './VerifyEmail.css';

const { Title, Text } = Typography;

export default function VerifyEmail() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const email = searchParams.get('email') || '';
  const { login, isAuthenticated } = useAuthStore();
  const [code, setCode] = useState(['', '', '', '', '', '']);
  const [loading, setLoading] = useState(false);
  const [resending, setResending] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

  useEffect(() => {
    if (isAuthenticated) navigate('/dashboard');
  }, [isAuthenticated, navigate]);

  useEffect(() => {
    if (!email) navigate('/register');
  }, [email, navigate]);

  useEffect(() => {
    if (countdown > 0) {
      const timer = setTimeout(() => setCountdown(countdown - 1), 1000);
      return () => clearTimeout(timer);
    }
  }, [countdown]);

  useEffect(() => {
    inputRefs.current[0]?.focus();
  }, []);

  const handleChange = (index: number, value: string) => {
    if (!/^\d*$/.test(value)) return;
    const newCode = [...code];
    newCode[index] = value.slice(-1);
    setCode(newCode);

    if (value && index < 5) {
      inputRefs.current[index + 1]?.focus();
    }

    const fullCode = newCode.join('');
    if (fullCode.length === 6) {
      handleVerify(fullCode);
    }
  };

  const handleKeyDown = (index: number, e: React.KeyboardEvent) => {
    if (e.key === 'Backspace' && !code[index] && index > 0) {
      inputRefs.current[index - 1]?.focus();
    }
  };

  const handlePaste = (e: React.ClipboardEvent) => {
    e.preventDefault();
    const pasted = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, 6);
    if (pasted.length === 6) {
      const newCode = pasted.split('');
      setCode(newCode);
      handleVerify(pasted);
    }
  };

  const handleVerify = async (fullCode: string) => {
    setLoading(true);
    try {
      const response = await api.verifyEmail(email, fullCode);
      login(response.token, response.user);
      message.success('Email verified! Welcome aboard!');
      navigate('/dashboard');
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } } };
      message.error(error.response?.data?.message || 'Invalid verification code');
      setCode(['', '', '', '', '', '']);
      inputRefs.current[0]?.focus();
    } finally {
      setLoading(false);
    }
  };

  const handleResend = async () => {
    setResending(true);
    try {
      await api.resendCode(email);
      message.success('New verification code sent!');
      setCountdown(60);
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } } };
      message.error(error.response?.data?.message || 'Failed to resend code');
    } finally {
      setResending(false);
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
          <h2 className="branding-tagline">Almost there!</h2>
        </div>
      </div>

      <div className="login-form-container">
        <Card className="login-card" bordered={false}>
          <div className="verify-content">
            <div className="verify-icon">✉️</div>
            <Title level={3}>Check your email</Title>
            <Text type="secondary">
              We sent a 6-digit code to<br />
              <Text strong>{email}</Text>
            </Text>

            <div className="code-inputs" onPaste={handlePaste}>
              {code.map((digit, i) => (
                <input
                  key={i}
                  ref={(el) => { inputRefs.current[i] = el; }}
                  type="text"
                  inputMode="numeric"
                  maxLength={1}
                  value={digit}
                  onChange={(e) => handleChange(i, e.target.value)}
                  onKeyDown={(e) => handleKeyDown(i, e)}
                  className="code-input"
                  disabled={loading}
                />
              ))}
            </div>

            <Button
              type="link"
              onClick={handleResend}
              loading={resending}
              disabled={countdown > 0}
              style={{ marginTop: 16 }}
            >
              {countdown > 0 ? `Resend code in ${countdown}s` : "Didn't receive the code? Resend"}
            </Button>
          </div>
        </Card>
      </div>
    </div>
  );
}
