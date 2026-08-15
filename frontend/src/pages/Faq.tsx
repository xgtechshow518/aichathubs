import { useState } from 'react';
import StaticPageLayout from '../components/StaticPageLayout';

const faqs = [
  {
    q: 'How does AIChatsHub work?',
    a: 'AIChatsHub connects to your WhatsApp account via QR code (just like WhatsApp Web). You then upload your FAQ or add Q&A pairs to build a knowledge base. When a customer messages you, our AI reads the question, finds the best matching answer from your knowledge base, and replies automatically.',
  },
  {
    q: 'Do I need a WhatsApp Business API account?',
    a: 'No! AIChatsHub works with any regular WhatsApp account. You simply scan a QR code to connect — no API setup, no business verification, no approval process needed.',
  },
  {
    q: 'How fast are AI responses?',
    a: 'Responses are typically delivered in under 2 seconds. Your customers get instant answers 24/7, even outside business hours.',
  },
  {
    q: 'What happens if the AI can\'t answer a question?',
    a: 'If the AI doesn\'t find a matching answer in your knowledge base, it will politely let the customer know and suggest connecting with a human agent. You\'ll see the unanswered question in your dashboard so you can follow up.',
  },
  {
    q: 'Can the bot reply in different languages?',
    a: 'Yes! The AI automatically detects the language your customer writes in and responds in the same language. This works for all major languages.',
  },
  {
    q: 'How do I set up my knowledge base?',
    a: 'You can either upload documents (PDF, Excel, etc.) containing your FAQs, or manually add question-and-answer pairs through the dashboard. The AI learns from these to answer customer queries.',
  },
  {
    q: 'Is there a free trial?',
    a: 'Yes! Every new account gets a 7-day free trial with 1 WhatsApp device included. No credit card required to start.',
  },
  {
    q: 'How much does it cost after the trial?',
    a: 'AIChatsHub costs $9.99 per WhatsApp device per month. Everything is unlimited — messages, Q&A items, and all features are included with every device.',
  },
  {
    q: 'Can I connect multiple WhatsApp numbers?',
    a: 'Absolutely. Each device slot lets you connect one WhatsApp number. You can add as many devices as you need — just adjust your subscription.',
  },
  {
    q: 'Is my data secure?',
    a: 'Yes. We take security seriously. All connections are encrypted, and we never store your WhatsApp messages on our servers beyond what\'s needed for the AI to respond. See our Privacy Policy for full details.',
  },
];

export default function Faq() {
  const [openIndex, setOpenIndex] = useState<number | null>(null);

  const toggle = (i: number) => setOpenIndex(openIndex === i ? null : i);

  return (
    <StaticPageLayout>
      <h1>Frequently Asked Questions</h1>
      <p className="page-subtitle">Everything you need to know about AIChatsHub.</p>

      {faqs.map((faq, i) => (
        <div key={i} className="faq-item">
          <button className="faq-question" onClick={() => toggle(i)}>
            {faq.q}
            <span className={`faq-toggle ${openIndex === i ? 'open' : ''}`}>+</span>
          </button>
          {openIndex === i && <div className="faq-answer">{faq.a}</div>}
        </div>
      ))}
    </StaticPageLayout>
  );
}
