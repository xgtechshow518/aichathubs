import StaticPageLayout from '../components/StaticPageLayout';

export default function Privacy() {
  return (
    <StaticPageLayout>
      <h1>Privacy Policy</h1>
      <p className="last-updated">Last updated: April 17, 2026</p>

      <p>
        At AIChatsHub, your privacy is important to us. This Privacy Policy explains
        what information we collect, how we use it, and what choices you have.
      </p>

      <h2>1. Information We Collect</h2>
      <h3>Account Information</h3>
      <p>
        When you create an account, we collect your name, email address, and password.
        If you sign in via Google, we receive your name, email, and profile picture
        from Google.
      </p>
      <h3>Usage Data</h3>
      <p>
        We collect information about how you use the Service, including pages visited,
        features used, and interaction patterns. This helps us improve the product.
      </p>
      <h3>WhatsApp Messages</h3>
      <p>
        When you connect your WhatsApp account, incoming customer messages are
        processed by our AI to generate responses. Messages are retained only as
        long as necessary to provide the Service and are not shared with third parties.
      </p>
      <h3>Payment Information</h3>
      <p>
        Payment processing is handled by Stripe. We do not store your credit card
        numbers. Stripe's privacy policy governs payment data.
      </p>

      <h2>2. How We Use Your Information</h2>
      <ul>
        <li>Provide, operate, and maintain the Service.</li>
        <li>Process transactions and send related notifications.</li>
        <li>Respond to your comments, questions, and support requests.</li>
        <li>Send you service-related announcements and updates.</li>
        <li>Monitor and analyze trends, usage, and activities.</li>
        <li>Detect, prevent, and address fraud or technical issues.</li>
      </ul>

      <h2>3. Data Sharing</h2>
      <p>
        We do not sell your personal information. We may share data with:
      </p>
      <ul>
        <li><strong>Service providers</strong> who assist us in operating the platform (e.g., hosting, payment processing, email delivery).</li>
        <li><strong>Law enforcement</strong> if required by law or to protect our rights.</li>
        <li><strong>Business transfers</strong> in connection with a merger, acquisition, or sale of assets.</li>
      </ul>

      <h2>4. Data Retention</h2>
      <p>
        We retain your account data for as long as your account is active. Chat
        messages and usage data are retained for up to 12 months after account
        deletion, after which they are permanently removed.
      </p>

      <h2>5. Data Security</h2>
      <p>
        We implement industry-standard security measures to protect your data,
        including encryption in transit (TLS) and at rest. However, no method of
        transmission or storage is 100% secure.
      </p>

      <h2>6. Your Rights</h2>
      <p>Depending on your jurisdiction, you may have the right to:</p>
      <ul>
        <li>Access the personal data we hold about you.</li>
        <li>Request correction of inaccurate data.</li>
        <li>Request deletion of your data.</li>
        <li>Object to or restrict certain processing.</li>
        <li>Data portability (receive your data in a structured format).</li>
      </ul>
      <p>
        To exercise these rights, contact us at{' '}
        <a href="mailto:hello@aichatshub.com">hello@aichatshub.com</a>.
      </p>

      <h2>7. Cookies</h2>
      <p>
        We use essential cookies for authentication and session management. We do not
        use tracking or advertising cookies.
      </p>

      <h2>8. Children's Privacy</h2>
      <p>
        The Service is not intended for users under 16 years of age. We do not
        knowingly collect personal information from children.
      </p>

      <h2>9. Changes to This Policy</h2>
      <p>
        We may update this Privacy Policy from time to time. We will notify you of
        material changes by posting the updated policy on our website and updating the
        "Last updated" date.
      </p>

      <h2>10. Contact</h2>
      <p>
        If you have questions about this Privacy Policy, please contact us at{' '}
        <a href="mailto:hello@aichatshub.com">hello@aichatshub.com</a>.
      </p>
    </StaticPageLayout>
  );
}
