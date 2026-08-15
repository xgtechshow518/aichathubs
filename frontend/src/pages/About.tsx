import StaticPageLayout from '../components/StaticPageLayout';

const values = [
  { icon: '🤖', title: 'AI-First', desc: 'We leverage cutting-edge AI to deliver instant, accurate customer responses around the clock.' },
  { icon: '⚡', title: 'Simplicity', desc: 'No coding, no API keys, no business verification. Set up in under 5 minutes.' },
  { icon: '🔒', title: 'Privacy & Security', desc: 'Your data stays yours. End-to-end encryption and strict data protection policies.' },
  { icon: '🌍', title: 'Global Reach', desc: 'Multi-language support so you can serve customers anywhere in the world.' },
];

export default function About() {
  return (
    <StaticPageLayout>
      <h1>About Us</h1>
      <p className="page-subtitle">Empowering businesses with AI-driven customer conversations.</p>

      <p>
        AIChatsHub was founded with a simple mission: make world-class customer service
        accessible to every business, regardless of size or technical expertise. We believe
        that no customer message should go unanswered and that every business deserves
        an always-on, intelligent support agent.
      </p>

      <p>
        Our platform connects directly to WhatsApp — the world's most popular messaging
        app — and uses advanced AI to understand and respond to customer queries instantly.
        Whether it's answering frequently asked questions, providing product information,
        or routing complex issues to human agents, AIChatsHub handles it all.
      </p>

      <h2>Our Values</h2>
      <div className="about-values">
        {values.map((v) => (
          <div key={v.title} className="about-value-card">
            <span className="value-icon">{v.icon}</span>
            <h3>{v.title}</h3>
            <p>{v.desc}</p>
          </div>
        ))}
      </div>

      <h2>Our Story</h2>
      <p>
        We started as a small team frustrated by the gap between enterprise-grade
        customer service tools and what small businesses could afford. We saw that
        millions of businesses rely on WhatsApp to talk to their customers, yet
        most lacked any automation or AI assistance.
      </p>
      <p>
        That's why we built AIChatsHub — a platform that anyone can set up in minutes,
        with no technical skills required. Just scan a QR code, upload your knowledge
        base, and let the AI do the rest.
      </p>
      <p>
        Today, businesses worldwide use AIChatsHub to automate their customer
        conversations, reduce response times from hours to seconds, and keep their
        customers happy 24/7.
      </p>
    </StaticPageLayout>
  );
}
