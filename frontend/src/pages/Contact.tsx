import StaticPageLayout from '../components/StaticPageLayout';

const contacts = [
  { icon: '✉️', title: 'Email Support', desc: 'Get a response within 24 hours.', detail: 'hello@aichatshub.com' },
  { icon: '🐛', title: 'Bug Reports', desc: 'Found an issue? Let us know.', detail: 'hello@aichatshub.com' },
];

export default function Contact() {
  return (
    <StaticPageLayout>
      <h1>Contact Us</h1>
      <p className="page-subtitle">Have a question or need help? We'd love to hear from you.</p>

      <p>
        Whether you need help setting up your account, have a question about billing,
        or just want to say hello, our team is here for you. Choose the best way to
        reach us below.
      </p>

      <div className="contact-grid">
        {contacts.map((c) => (
          <div key={c.title} className="contact-card">
            <span className="contact-icon">{c.icon}</span>
            <h3>{c.title}</h3>
            <p>{c.desc}</p>
            <p><strong>{c.detail}</strong></p>
          </div>
        ))}
      </div>

      <h2>General Inquiries</h2>
      <p>
        For partnerships, press inquiries, or anything else, email us at{' '}
        <a href="mailto:hello@aichatshub.com">hello@aichatshub.com</a>.
        We typically respond within one business day.
      </p>
    </StaticPageLayout>
  );
}
