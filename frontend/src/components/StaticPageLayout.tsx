import { Link, useNavigate } from 'react-router-dom';
import { Button } from 'antd';
import '../pages/StaticPage.css';

const footerLinks = {
  product: [
    { label: 'Features', href: '/#features' },
    { label: 'Pricing', href: '/#pricing' },
    { label: 'FAQ', href: '/faq' },
  ],
  company: [
    { label: 'About Us', href: '/about' },
    { label: 'Contact', href: '/contact' },
  ],
  legal: [
    { label: 'Terms of Service', href: '/terms' },
    { label: 'Privacy Policy', href: '/privacy' },
  ],
};

export default function StaticPageLayout({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate();

  return (
    <div className="static-page">
      <nav className="static-nav">
        <div className="nav-inner">
          <Link to="/" className="nav-logo">
            <span className="nav-logo-icon">💬</span>
            <span className="nav-logo-text">AIChatsHub</span>
          </Link>
          <div className="nav-links">
            <Link to="/#features">Features</Link>
            <Link to="/#pricing">Pricing</Link>
            <Link to="/#testimonials">Reviews</Link>
            <Button onClick={() => navigate('/login')}>Sign In</Button>
            <Button type="primary" onClick={() => navigate('/register')}>Get Started Free</Button>
          </div>
        </div>
      </nav>

      <main className="static-content">
        {children}
      </main>

      <footer className="static-footer">
        <div className="footer-inner">
          <div className="footer-columns">
            <div className="footer-brand">
              <div className="brand-logo">
                <span className="brand-logo-icon">💬</span>
                <span className="brand-logo-text">AIChatsHub</span>
              </div>
              <p>AI-powered WhatsApp automation for smarter customer engagement.</p>
            </div>
            <div className="footer-column">
              <h4>Product</h4>
              <ul>
                {footerLinks.product.map((link) => (
                  <li key={link.href}>
                    <Link to={link.href}>{link.label}</Link>
                  </li>
                ))}
              </ul>
            </div>
            <div className="footer-column">
              <h4>Company</h4>
              <ul>
                {footerLinks.company.map((link) => (
                  <li key={link.href}>
                    <Link to={link.href}>{link.label}</Link>
                  </li>
                ))}
              </ul>
            </div>
            <div className="footer-column">
              <h4>Legal</h4>
              <ul>
                {footerLinks.legal.map((link) => (
                  <li key={link.href}>
                    <Link to={link.href}>{link.label}</Link>
                  </li>
                ))}
              </ul>
            </div>
          </div>
          <div className="footer-bottom">
            &copy; {new Date().getFullYear()} AIChatsHub. All rights reserved.
          </div>
        </div>
      </footer>
    </div>
  );
}
