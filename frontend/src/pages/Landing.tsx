import { useNavigate, Link } from 'react-router-dom';
import { Button } from 'antd';
import {
  RobotOutlined,
  WhatsAppOutlined,
  ThunderboltOutlined,
  TeamOutlined,
  SafetyOutlined,
  GlobalOutlined,
  CheckOutlined,
  StarFilled,
  ArrowRightOutlined,
} from '@ant-design/icons';
import './Landing.css';

const features = [
  {
    icon: <RobotOutlined />,
    title: 'AI Auto-Reply',
    desc: 'Your AI bot answers customer questions instantly using your knowledge base.',
  },
  {
    icon: <WhatsAppOutlined />,
    title: 'WhatsApp Integration',
    desc: 'Connect your WhatsApp account in seconds via QR code. No API setup or business verification needed.',
  },
  {
    icon: <ThunderboltOutlined />,
    title: 'Instant Responses',
    desc: 'Customers get answers in under 2 seconds, 24/7. No more waiting for business hours.',
  },
  {
    icon: <TeamOutlined />,
    title: 'Human Handoff',
    desc: 'When the bot can\'t answer, it seamlessly suggests connecting with a human agent.',
  },
  {
    icon: <SafetyOutlined />,
    title: 'Knowledge Base',
    desc: 'Upload your FAQ docs or add Q&A pairs. The AI learns your business and answers accurately.',
  },
  {
    icon: <GlobalOutlined />,
    title: 'Multi-Language',
    desc: 'The bot automatically replies in the same language your customer writes in.',
  },
];

const allFeatures = [
  'AI Auto-Reply',
  'Unlimited messages',
  'Unlimited Q&A items',
  'Knowledge Base & File Upload',
  'Multi-language support',
  'Real-time chat dashboard',
  '7-day free trial (1 device)',
  'Priority support',
];

const testimonials = [
  {
    name: 'Sarah Chen',
    role: 'E-Commerce Owner',
    avatar: 'S',
    text: 'AIChatsHub cut our response time from 2 hours to 2 seconds. Our customer satisfaction jumped 40% in the first month!',
    rating: 5,
  },
  {
    name: 'Marcus Rodriguez',
    role: 'Real Estate Agent',
    avatar: 'M',
    text: 'I connected my WhatsApp in 30 seconds. Now my clients get instant property info even when I\'m showing houses. Game changer.',
    rating: 5,
  },
  {
    name: 'Aisha Patel',
    role: 'Clinic Manager',
    avatar: 'A',
    text: 'Patients ask about appointment times, pricing, and services at all hours. The AI handles 80% of queries automatically. Love it!',
    rating: 5,
  },
  {
    name: 'James Liu',
    role: 'SaaS Founder',
    avatar: 'J',
    text: 'We uploaded our entire help docs and the bot answers technical questions better than most support agents. The multi-language support is incredible.',
    rating: 5,
  },
];

export default function Landing() {
  const navigate = useNavigate();

  return (
    <div className="landing-page">
      {/* Navigation */}
      <nav className="landing-nav">
        <div className="nav-container">
          <div className="nav-logo">
            <span className="nav-logo-icon">💬</span>
            <span className="nav-logo-text">AIChatsHub</span>
          </div>
          <div className="nav-links">
            <a href="#features">Features</a>
            <a href="#pricing">Pricing</a>
            <a href="#testimonials">Reviews</a>
            <Button onClick={() => navigate('/login')}>Sign In</Button>
            <Button type="primary" onClick={() => navigate('/register')}>Get Started Free</Button>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <section className="hero-section">
        <div className="hero-container">
          <div className="hero-badge">AI-Powered Chat Platform</div>
          <h1 className="hero-title">
            Turn Your WhatsApp Into an
            <span className="hero-highlight"> AI-Powered </span>
            Customer Service Agent
          </h1>
          <p className="hero-subtitle">
            Connect your WhatsApp, upload your FAQ, and let AI handle customer queries 24/7.
            Set up in under 5 minutes. No coding required.
          </p>
          <div className="hero-actions">
            <Button
              type="primary"
              size="large"
              icon={<ArrowRightOutlined />}
              onClick={() => navigate('/register')}
              className="hero-cta"
            >
              Start Free Trial
            </Button>
            <Button size="large" onClick={() => navigate('/login')} className="hero-secondary">
              Sign In
            </Button>
          </div>
          <div className="hero-stats">
            <div className="stat">
              <strong>2s</strong>
              <span>Avg Response</span>
            </div>
            <div className="stat-divider" />
            <div className="stat">
              <strong>24/7</strong>
              <span>Always Online</span>
            </div>
            <div className="stat-divider" />
            <div className="stat">
            <strong>99%</strong>
            <span>Auto-Resolved</span>
            </div>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="features-section" id="features">
        <div className="section-container">
          <div className="section-header">
            <h2>Everything You Need to Automate Customer Service</h2>
            <p>From WhatsApp integration to AI-powered responses, we've got you covered.</p>
          </div>
          <div className="features-grid">
            {features.map((f) => (
              <div key={f.title} className="feature-card">
                <div className="feature-icon">{f.icon}</div>
                <h3>{f.title}</h3>
                <p>{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section className="how-section">
        <div className="section-container">
          <div className="section-header">
            <h2>Up and Running in 3 Simple Steps</h2>
          </div>
          <div className="steps-grid">
            <div className="step-card">
              <div className="step-number">1</div>
              <h3>Connect WhatsApp</h3>
              <p>Scan a QR code with your phone — just like WhatsApp Web. Takes 30 seconds.</p>
            </div>
            <div className="step-card">
              <div className="step-number">2</div>
              <h3>Add Your Knowledge</h3>
              <p>Upload your FAQ docs or add Q&A pairs. The AI learns everything about your business.</p>
            </div>
            <div className="step-card">
              <div className="step-number">3</div>
              <h3>Turn On Auto-Reply</h3>
              <p>Flip the switch. Your AI bot starts answering customer messages instantly.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Pricing */}
      <section className="pricing-section" id="pricing">
        <div className="section-container">
          <div className="section-header">
            <h2>Simple, Transparent Pricing</h2>
            <p>One price per WhatsApp device. Everything unlimited. No hidden fees.</p>
          </div>
          <div className="pricing-layout">
            <div className="pricing-card popular">
              <div className="popular-badge">All-Inclusive</div>
              <h3>Per WhatsApp Device</h3>
              <div className="pricing-amount">
                <span className="currency">$</span>
                <span className="price">9.99</span>
                <span className="period">/device/month</span>
              </div>
              <div className="pricing-devices">Select how many WhatsApp devices you need</div>

              <div className="pricing-example">
                <div className="example-row"><span>1 device</span><span className="tier-price">$9.99/mo</span></div>
                <div className="example-row"><span>3 devices</span><span className="tier-price">$29.97/mo</span></div>
                <div className="example-row"><span>5 devices</span><span className="tier-price">$49.95/mo</span></div>
                <div className="example-row"><span>10 devices</span><span className="tier-price">$99.90/mo</span></div>
              </div>

              <Button
                type="primary"
                size="large"
                block
                onClick={() => navigate('/register')}
                style={{ marginTop: 24 }}
              >
                Start 7-Day Free Trial
              </Button>
              <div style={{ textAlign: 'center', marginTop: 8 }}>
                <span style={{ fontSize: 13, color: '#9ca3af' }}>1 free device for 7 days. No credit card required.</span>
              </div>
            </div>

            <div className="features-card">
              <h3>Everything Included</h3>
              <p className="features-subtitle">Every device comes with the full feature set</p>
              <ul className="pricing-features">
                {allFeatures.map((f) => (
                  <li key={f}><CheckOutlined /> {f}</li>
                ))}
              </ul>
              <Button
                size="large"
                block
                onClick={() => navigate('/register')}
              >
                Get Started Free
              </Button>
            </div>
          </div>
        </div>
      </section>

      {/* Testimonials */}
      <section className="testimonials-section" id="testimonials">
        <div className="section-container">
          <div className="section-header">
            <h2>Loved by Businesses Worldwide</h2>
          </div>
          <div className="testimonials-grid">
            {testimonials.map((t) => (
              <div key={t.name} className="testimonial-card">
                <div className="testimonial-stars">
                  {Array.from({ length: t.rating }).map((_, i) => (
                    <StarFilled key={i} />
                  ))}
                </div>
                <p className="testimonial-text">"{t.text}"</p>
                <div className="testimonial-author">
                  <div className="testimonial-avatar">{t.avatar}</div>
                  <div>
                    <strong>{t.name}</strong>
                    <span>{t.role}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="cta-section">
        <div className="section-container">
          <h2>Ready to Transform Your Customer Service?</h2>
          <p>Join thousands of businesses using AI to delight their customers.</p>
          <Button
            type="primary"
            size="large"
            icon={<ArrowRightOutlined />}
            onClick={() => navigate('/register')}
            className="hero-cta"
          >
            Start Your Free Trial
          </Button>
        </div>
      </section>

      {/* Footer */}
      <footer className="landing-footer-full">
        <div className="footer-full-inner">
          <div className="footer-full-columns">
            <div className="footer-full-brand">
              <div className="footer-full-logo">
                <span className="nav-logo-icon">💬</span>
                <span className="nav-logo-text">AIChatsHub</span>
              </div>
              <p className="footer-full-desc">AI-powered WhatsApp automation for smarter customer engagement.</p>
            </div>
            <div className="footer-full-col">
              <h4>Product</h4>
              <ul>
                <li><a href="#features">Features</a></li>
                <li><a href="#pricing">Pricing</a></li>
                <li><Link to="/faq">FAQ</Link></li>
              </ul>
            </div>
            <div className="footer-full-col">
              <h4>Company</h4>
              <ul>
                <li><Link to="/about">About Us</Link></li>
                <li><Link to="/contact">Contact</Link></li>
              </ul>
            </div>
            <div className="footer-full-col">
              <h4>Legal</h4>
              <ul>
                <li><Link to="/terms">Terms of Service</Link></li>
                <li><Link to="/privacy">Privacy Policy</Link></li>
              </ul>
            </div>
          </div>
          <div className="footer-full-bottom">
            &copy; {new Date().getFullYear()} AIChatsHub. All rights reserved.
          </div>
        </div>
      </footer>
    </div>
  );
}
