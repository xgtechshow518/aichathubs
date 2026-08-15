import StaticPageLayout from '../components/StaticPageLayout';

export default function Terms() {
  return (
    <StaticPageLayout>
      <h1>Terms of Service</h1>
      <p className="last-updated">Last updated: April 17, 2026</p>

      <p>
        Welcome to AIChatsHub. By accessing or using our website and services, you
        agree to be bound by these Terms of Service. If you do not agree, please do
        not use our services.
      </p>

      <h2>1. Definitions</h2>
      <p>
        "Service" refers to the AIChatsHub platform, including the website, dashboard,
        API, and any related tools. "User," "you," and "your" refer to anyone who
        accesses or uses the Service. "We," "us," and "our" refer to AIChatsHub.
      </p>

      <h2>2. Account Registration</h2>
      <p>
        To use certain features, you must create an account. You agree to provide
        accurate information and keep your account credentials secure. You are
        responsible for all activity under your account.
      </p>

      <h2>3. Acceptable Use</h2>
      <p>You agree not to:</p>
      <ul>
        <li>Use the Service for any unlawful purpose or to violate any laws.</li>
        <li>Send spam, unsolicited messages, or bulk automated messages through the Service.</li>
        <li>Attempt to gain unauthorized access to the Service or its related systems.</li>
        <li>Reverse engineer, decompile, or disassemble any part of the Service.</li>
        <li>Use the Service in a way that could harm, disable, or impair it.</li>
      </ul>

      <h2>4. Subscriptions & Payments</h2>
      <p>
        AIChatsHub offers paid subscription plans billed monthly. By subscribing, you
        authorize us to charge your payment method on a recurring basis. You may cancel
        your subscription at any time; cancellation takes effect at the end of the
        current billing period.
      </p>

      <h2>5. Free Trial</h2>
      <p>
        New users may be eligible for a free trial. If you do not cancel before the
        trial ends, your subscription will begin automatically and you will be charged
        according to the plan selected.
      </p>

      <h2>6. Intellectual Property</h2>
      <p>
        All content, features, and functionality of the Service are owned by AIChatsHub
        and are protected by intellectual property laws. You may not copy, modify, or
        distribute any part of the Service without our prior written consent.
      </p>

      <h2>7. Data & Privacy</h2>
      <p>
        Your use of the Service is also governed by our{' '}
        <a href="/privacy">Privacy Policy</a>. By using the Service, you consent to
        the collection and use of data as described therein.
      </p>

      <h2>8. Limitation of Liability</h2>
      <p>
        To the maximum extent permitted by law, AIChatsHub shall not be liable for any
        indirect, incidental, special, consequential, or punitive damages arising from
        your use of the Service. Our total liability shall not exceed the amount you
        paid us in the 12 months preceding the claim.
      </p>

      <h2>9. Disclaimer of Warranties</h2>
      <p>
        The Service is provided "as is" and "as available" without warranties of any
        kind, either express or implied. We do not guarantee that the Service will be
        uninterrupted, error-free, or completely secure.
      </p>

      <h2>10. Termination</h2>
      <p>
        We reserve the right to suspend or terminate your account at any time if you
        violate these Terms or engage in conduct that we determine is harmful to the
        Service or other users.
      </p>

      <h2>11. Changes to Terms</h2>
      <p>
        We may update these Terms from time to time. We will notify you of material
        changes by posting the updated Terms on our website. Your continued use of the
        Service after changes constitutes acceptance.
      </p>

      <h2>12. Contact</h2>
      <p>
        If you have questions about these Terms, please contact us at{' '}
        <a href="mailto:hello@aichatshub.com">hello@aichatshub.com</a>.
      </p>
    </StaticPageLayout>
  );
}
