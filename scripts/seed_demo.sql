-- =====================================================================
-- AIChatsHub — demo / dummy data seed
--
-- Populates the admin panel and dashboards with realistic sample data:
-- operators, subscriptions, WhatsApp devices, chats + messages, a product
-- catalog, leads, customer profiles, tags, and a knowledge base.
--
-- Intended for a FRESH database (right after first boot). It is NOT
-- idempotent — re-running on an already-seeded DB fails on duplicate
-- emails. To re-seed, drop/recreate the DB (or `docker compose down -v`)
-- first, let the app run once to migrate, then apply this script.
--
-- The whole thing runs in one transaction: any error rolls back cleanly,
-- leaving the database untouched.
--
-- Demo operator login (created by this script):
--     email:    demo@aichathubs.local
--     password: demo123456
-- (The bcrypt hash below encodes exactly that password.)
--
-- Usage (Docker):
--     docker cp scripts/seed_demo.sql aichathubs-db:/tmp/seed_demo.sql
--     docker exec aichathubs-db \
--       psql -U postgres -d smart_live_chats -v ON_ERROR_STOP=1 -f /tmp/seed_demo.sql
-- =====================================================================
BEGIN;

-- ---------------------------------------------------------------------
-- 0. Make the existing demo user a usable operator (known password)
--    bcrypt hash below == "demo123456"
-- ---------------------------------------------------------------------
UPDATE users SET
  password = '$2a$10$MUKwDPPb..x6DqzqTFU3quhJ72xot09Z5PT4biiRrSVtyG1fA3jfy',
  provider = 'email',
  email_verified = true,
  subscription_plan = 'pro',
  subscription_status = 'active',
  max_devices = 5
WHERE email = 'demo@aichathubs.local';

-- ---------------------------------------------------------------------
-- 1. Operator users (for admin breadth: providers / plans / statuses /
--    verified mix / suspended / 30-day signup trend)
-- ---------------------------------------------------------------------
INSERT INTO users (email, name, provider, email_verified, subscription_plan, subscription_status, max_devices, avatar_url, suspended, created_at, updated_at) VALUES
 ('sarah.chen@brightmart.com',   'Sarah Chen',    'google',   true,  'pro',        'active',    3, '', false, now() - interval '28 days', now() - interval '1 day'),
 ('james.wong@techgadgets.io',   'James Wong',    'email',    true,  'starter',    'active',    1, '', false, now() - interval '25 days', now() - interval '2 days'),
 ('aisha.patel@glowbeauty.co',   'Aisha Patel',   'facebook', true,  'pro',        'active',    3, '', false, now() - interval '20 days', now() - interval '1 day'),
 ('carlos.mendez@fitgearpro.com','Carlos Mendez', 'email',    true,  'enterprise', 'active',   10, '', false, now() - interval '15 days', now() - interval '3 days'),
 ('yuki.tanaka@sakuraboutique.jp','Yuki Tanaka',  'google',   true,  'starter',    'trialing',  1, '', false, now() - interval '10 days', now() - interval '1 day'),
 ('liam.oconnor@pubgrub.ie',     'Liam O''Connor','email',    false, 'trial',      'trialing',  1, '', false, now() - interval '5 days',  now() - interval '5 days'),
 ('nina.volkov@petpalace.shop',  'Nina Volkov',   'email',    true,  'starter',    'past_due',  1, '', false, now() - interval '3 days',  now() - interval '1 day'),
 ('tom.baker@retrovinyl.uk',     'Tom Baker',     'facebook', true,  'pro',        'canceled',  3, '', false, now() - interval '1 day',   now() - interval '1 day'),
 ('spammer@fakeshop.xyz',        'Blocked Seller','email',    false, 'trial',      'canceled',  1, '', true,  now() - interval '2 days',  now() - interval '1 day');

UPDATE users SET suspended_at = now() - interval '1 day',
  suspended_reason = 'Flagged for spam / abuse of WhatsApp messaging policy'
WHERE email = 'spammer@fakeshop.xyz';

-- Trial end dates for trialing users
UPDATE users SET trial_ends_at = now() + interval '4 days'  WHERE email = 'yuki.tanaka@sakuraboutique.jp';
UPDATE users SET trial_ends_at = now() + interval '9 days'  WHERE email = 'liam.oconnor@pubgrub.ie';
UPDATE users SET trial_ends_at = now() + interval '10 days' WHERE email = 'demo@aichathubs.local';

-- ---------------------------------------------------------------------
-- 2. Stripe subscriptions (Payments page)
-- ---------------------------------------------------------------------
INSERT INTO subscriptions (user_id, stripe_subscription_id, stripe_price_id, plan, status, current_period_start, current_period_end, cancel_at_period_end, created_at, updated_at)
SELECT id, 'sub_demo_' || id, 'price_pro',        'pro',        'active',   now() - interval '10 days', now() + interval '20 days', false, now() - interval '10 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'sub_demo_' || id, 'price_pro',        'pro',        'active',   now() - interval '8 days',  now() + interval '22 days', false, now() - interval '28 days', now() FROM users WHERE email='sarah.chen@brightmart.com'
UNION ALL SELECT id, 'sub_demo_' || id, 'price_starter',    'starter',    'active',   now() - interval '5 days',  now() + interval '25 days', false, now() - interval '25 days', now() FROM users WHERE email='james.wong@techgadgets.io'
UNION ALL SELECT id, 'sub_demo_' || id, 'price_pro',        'pro',        'active',   now() - interval '2 days',  now() + interval '28 days', false, now() - interval '20 days', now() FROM users WHERE email='aisha.patel@glowbeauty.co'
UNION ALL SELECT id, 'sub_demo_' || id, 'price_enterprise', 'enterprise', 'active',   now() - interval '15 days', now() + interval '15 days', false, now() - interval '15 days', now() FROM users WHERE email='carlos.mendez@fitgearpro.com'
UNION ALL SELECT id, 'sub_demo_' || id, 'price_starter',    'starter',    'past_due', now() - interval '9 days',  now() - interval '1 day',   false, now() - interval '3 days',  now() FROM users WHERE email='nina.volkov@petpalace.shop'
UNION ALL SELECT id, 'sub_demo_' || id, 'price_pro',        'pro',        'canceled', now() - interval '31 days', now() - interval '1 day',   true,  now() - interval '31 days', now() FROM users WHERE email='tom.baker@retrovinyl.uk';

-- ---------------------------------------------------------------------
-- 3. WhatsApp devices (unique phone used as reference key)
-- ---------------------------------------------------------------------
INSERT INTO whatsapp_devices (user_id, j_id, push_name, phone, status, connected_at, created_at, updated_at)
SELECT id, '61411000001@s.whatsapp.net', 'Demo Store',        '61411000001', 'connected',    now() - interval '9 days',  now() - interval '12 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, '61411000002@s.whatsapp.net', 'Demo Support',      '61411000002', 'disconnected', NULL,                       now() - interval '6 days',  now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, '6591000010@s.whatsapp.net',  'BrightMart',        '6591000010',  'connected',    now() - interval '3 days',  now() - interval '27 days', now() FROM users WHERE email='sarah.chen@brightmart.com'
UNION ALL SELECT id, '6591000011@s.whatsapp.net',  'BrightMart VIP',    '6591000011',  'connected',    now() - interval '1 day',   now() - interval '20 days', now() FROM users WHERE email='sarah.chen@brightmart.com'
UNION ALL SELECT id, '14155000020@s.whatsapp.net', 'TechGadgets',       '14155000020', 'connected',    now() - interval '2 days',  now() - interval '24 days', now() FROM users WHERE email='james.wong@techgadgets.io'
UNION ALL SELECT id, '447700000030@s.whatsapp.net','GlowBeauty',        '447700000030','disconnected', NULL,                       now() - interval '19 days', now() FROM users WHERE email='aisha.patel@glowbeauty.co'
UNION ALL SELECT id, '5215500000040@s.whatsapp.net','FitGear Pro',      '5215500000040','connected',   now() - interval '4 hours', now() - interval '14 days', now() FROM users WHERE email='carlos.mendez@fitgearpro.com'
UNION ALL SELECT id, '819000000050@s.whatsapp.net', 'Sakura Boutique',  '819000000050', 'disconnected',NULL,                       now() - interval '9 days',  now() FROM users WHERE email='yuki.tanaka@sakuraboutique.jp';

-- ---------------------------------------------------------------------
-- 4. Chat sessions (global list; unique customer_phone as reference key)
--    spread across platforms / statuses / 30-day window incl. today
-- ---------------------------------------------------------------------
INSERT INTO chat_sessions (customer_name, customer_phone, customer_avatar, platform, unread_count, last_message, last_message_at, last_sender_type, status, device_id, assigned_to_id, created_at, updated_at)
SELECT 'Emily Roberts',   '61490001001', '', 'whatsapp', 2, 'Do you ship to Perth?',                 now() - interval '15 minutes', 'customer', 'active', d.id, u.id, now() - interval '2 days',  now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='61411000001'
UNION ALL SELECT 'David Kim',       '61490001002', '', 'whatsapp', 0, 'Thanks, order placed! 🎉',    now() - interval '2 hours',    'customer', 'active', d.id, u.id, now() - interval '4 days',  now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='61411000001'
UNION ALL SELECT 'Sophie Martin',   '61490001003', '', 'whatsapp', 1, 'Is the blue one in stock?',    now() - interval '40 minutes', 'customer', 'active', d.id, u.id, now() - interval '1 day',   now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='61411000001'
UNION ALL SELECT 'Michael Brown',   '61490001004', '', 'whatsapp', 0, 'Perfect, see you then.',       now() - interval '6 days',     'agent',    'closed', d.id, u.id, now() - interval '8 days',  now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='61411000001'
UNION ALL SELECT 'Priya Nair',      '61490001005', '', 'whatsapp', 3, 'Can I get a bulk discount?',   now() - interval '5 minutes',  'customer', 'active', d.id, u.id, now() - interval '3 hours', now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='61411000001'
UNION ALL SELECT 'Lucas Silva',     '6590002001',  '', 'whatsapp', 0, 'Great, thank you!',            now() - interval '1 day',      'customer', 'closed', d.id, u.id, now() - interval '10 days', now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='6591000010'
UNION ALL SELECT 'Chloe Tan',       '6590002002',  '', 'whatsapp', 1, 'What are your opening hours?', now() - interval '3 hours',    'customer', 'active', d.id, u.id, now() - interval '5 days',  now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='6591000010'
UNION ALL SELECT 'Ahmed Hassan',    '14155009001', '', 'whatsapp', 0, 'The charger arrived, thanks.', now() - interval '2 days',     'customer', 'closed', d.id, u.id, now() - interval '12 days', now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='14155000020'
UNION ALL SELECT 'Isabella Rossi',  '14155009002', '', 'whatsapp', 4, 'My order hasn''t arrived yet', now() - interval '20 minutes', 'customer', 'active', d.id, u.id, now() - interval '2 days',  now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='14155000020'
UNION ALL SELECT 'Grace Miller',    '447700009001','', 'facebook', 1, 'Do you do gift wrapping?',     now() - interval '1 hour',     'customer', 'active', NULL, u.id, now() - interval '6 days',  now() FROM users u WHERE u.email='aisha.patel@glowbeauty.co'
UNION ALL SELECT 'Oliver Schmidt',  '447700009002','', 'facebook', 0, 'Refund processed, cheers.',    now() - interval '3 days',     'agent',    'closed', NULL, u.id, now() - interval '9 days',  now() FROM users u WHERE u.email='aisha.patel@glowbeauty.co'
UNION ALL SELECT 'Mia Johnson',     '5215509001',  '', 'whatsapp', 2, 'Which protein flavor is best?',now() - interval '30 minutes', 'customer', 'active', d.id, u.id, now() - interval '1 day',   now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='5215500000040'
UNION ALL SELECT 'Noah Williams',   '5215509002',  '', 'whatsapp', 0, 'Subscribed to the plan 💪',   now() - interval '4 hours',    'customer', 'active', d.id, u.id, now() - interval '3 days',  now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='5215500000040'
UNION ALL SELECT 'Hana Suzuki',     '819000009001','', 'telegram', 1, 'Kawaii! Do you ship to Osaka?',now() - interval '2 hours',    'customer', 'active', NULL, u.id, now() - interval '7 days',  now() FROM users u WHERE u.email='yuki.tanaka@sakuraboutique.jp'
UNION ALL SELECT 'Ethan Davis',     '61490001006', '', 'whatsapp', 0, 'No worries, take your time.',  now() - interval '25 days',    'customer', 'closed', d.id, u.id, now() - interval '26 days', now() FROM whatsapp_devices d JOIN users u ON u.id=d.user_id WHERE d.phone='61411000001';

-- ---------------------------------------------------------------------
-- 5. Chat messages (conversation threads incl. today; customer/agent/bot;
--    a few image/file types for the message-type breakdown)
-- ---------------------------------------------------------------------
-- Emily Roberts (unread, active) — demo store
INSERT INTO chat_messages (session_id, sender_type, sender_id, content, message_type, is_read, created_at)
SELECT s.id, 'customer', NULL::bigint,'Hi! Do you have the ceramic mug set in stock?', 'text', true,  now() - interval '90 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001001'
UNION ALL SELECT s.id, 'bot', NULL::bigint,'Hello! Yes, our Ceramic Mug Set (4-pack) is in stock at $29.90. Would you like the link?', 'text', true, now() - interval '89 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001001'
UNION ALL SELECT s.id, 'customer', NULL::bigint,'Yes please', 'text', true, now() - interval '80 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001001'
UNION ALL SELECT s.id, 'bot', NULL::bigint,'Here you go: https://demo.shop/p/ceramic-mug-set 🛒', 'text', true, now() - interval '79 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001001'
UNION ALL SELECT s.id, 'customer', NULL::bigint,'Do you ship to Perth?', 'text', false, now() - interval '15 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001001';

-- David Kim (converted)
INSERT INTO chat_messages (session_id, sender_type, sender_id, content, message_type, is_read, created_at)
SELECT s.id, 'customer', NULL::bigint,'Looking for a birthday gift under $50', 'text', true, now() - interval '3 hours' FROM chat_sessions s WHERE s.customer_phone='61490001002'
UNION ALL SELECT s.id, 'bot', NULL::bigint,'Great! Our Scented Candle Trio ($34.90) and Ceramic Mug Set ($29.90) are popular gifts. Want photos?', 'text', true, now() - interval '175 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001002'
UNION ALL SELECT s.id, 'customer', NULL::bigint,'Show me the candles', 'text', true, now() - interval '170 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001002'
UNION ALL SELECT s.id, 'bot', NULL::bigint,'', 'image', true, now() - interval '169 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001002'
UNION ALL SELECT s.id, 'agent', (SELECT id FROM users WHERE email='demo@aichathubs.local'), 'That trio is hand-poured soy wax — very popular! Shall I reserve one?', 'text', true, now() - interval '160 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001002'
UNION ALL SELECT s.id, 'customer', NULL::bigint,'Thanks, order placed! 🎉', 'text', true, now() - interval '2 hours' FROM chat_sessions s WHERE s.customer_phone='61490001002';

-- Priya Nair (bulk enquiry, today)
INSERT INTO chat_messages (session_id, sender_type, sender_id, content, message_type, is_read, created_at)
SELECT s.id, 'customer', NULL::bigint,'Hi, I run a cafe and need 20 mug sets', 'text', true, now() - interval '3 hours' FROM chat_sessions s WHERE s.customer_phone='61490001005'
UNION ALL SELECT s.id, 'bot', NULL::bigint,'Wonderful! For orders over 10 units we offer 15% off. That would be $508.30 for 20 sets.', 'text', true, now() - interval '175 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001005'
UNION ALL SELECT s.id, 'customer', NULL::bigint,'Can I get a bulk discount?', 'text', false, now() - interval '5 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001005';

-- Isabella Rossi (complaint, today)
INSERT INTO chat_messages (session_id, sender_type, sender_id, content, message_type, is_read, created_at)
SELECT s.id, 'customer', NULL::bigint,'My order #TG-1042 hasn''t arrived yet', 'text', true, now() - interval '2 hours' FROM chat_sessions s WHERE s.customer_phone='14155009002'
UNION ALL SELECT s.id, 'agent', (SELECT id FROM users WHERE email='james.wong@techgadgets.io'), 'So sorry Isabella! Let me check the tracking for you.', 'text', true, now() - interval '110 minutes' FROM chat_sessions s WHERE s.customer_phone='14155009002'
UNION ALL SELECT s.id, 'customer', NULL::bigint,'', 'file', true, now() - interval '100 minutes' FROM chat_sessions s WHERE s.customer_phone='14155009002'
UNION ALL SELECT s.id, 'customer', NULL::bigint,'My order hasn''t arrived yet', 'text', false, now() - interval '20 minutes' FROM chat_sessions s WHERE s.customer_phone='14155009002';

-- Mia Johnson (fitness, today)
INSERT INTO chat_messages (session_id, sender_type, sender_id, content, message_type, is_read, created_at)
SELECT s.id, 'customer', NULL::bigint,'Which protein flavor is best for beginners?', 'text', false, now() - interval '30 minutes' FROM chat_sessions s WHERE s.customer_phone='5215509001'
UNION ALL SELECT s.id, 'bot', NULL::bigint,'Vanilla is our most popular starter flavor — smooth and mixes easily. Want the 1kg or 2kg tub?', 'text', false, now() - interval '29 minutes' FROM chat_sessions s WHERE s.customer_phone='5215509001';

-- Sophie, Chloe, Grace, Hana — short single exchanges
INSERT INTO chat_messages (session_id, sender_type, sender_id, content, message_type, is_read, created_at)
SELECT s.id, 'customer', NULL::bigint,'Is the blue one in stock?', 'text', false, now() - interval '40 minutes' FROM chat_sessions s WHERE s.customer_phone='61490001003'
UNION ALL SELECT s.id, 'customer', NULL::bigint,'What are your opening hours?', 'text', false, now() - interval '3 hours' FROM chat_sessions s WHERE s.customer_phone='6590002002'
UNION ALL SELECT s.id, 'customer', NULL::bigint,'Do you do gift wrapping?', 'text', false, now() - interval '1 hour' FROM chat_sessions s WHERE s.customer_phone='447700009001'
UNION ALL SELECT s.id, 'bot', NULL::bigint,'Yes! Gift wrapping is +$3.50 and includes a handwritten note 🎁', 'text', false, now() - interval '58 minutes' FROM chat_sessions s WHERE s.customer_phone='447700009001'
UNION ALL SELECT s.id, 'customer', NULL::bigint,'Kawaii! Do you ship to Osaka?', 'text', false, now() - interval '2 hours' FROM chat_sessions s WHERE s.customer_phone='819000009001';

-- ---------------------------------------------------------------------
-- 6. Products (demo operator catalog, id=1) + a couple for Sarah
-- ---------------------------------------------------------------------
INSERT INTO products (user_id, sku, name, description, price, currency, stock, category, tags, product_url, active, created_at, updated_at)
SELECT id, 'MUG-SET-01', 'Ceramic Mug Set (4-pack)', 'Hand-glazed stoneware mugs, 350ml, dishwasher safe.', 29.90, 'USD', 120, 'Kitchen',   'mugs,ceramic,gift', 'https://demo.shop/p/ceramic-mug-set', true, now() - interval '20 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'CANDLE-TRIO', 'Scented Candle Trio', 'Hand-poured soy wax candles: lavender, vanilla, sandalwood.', 34.90, 'USD', 65, 'Home', 'candles,soy,gift', 'https://demo.shop/p/candle-trio', true, now() - interval '19 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'TOTE-CANVAS', 'Canvas Tote Bag', 'Heavy 12oz cotton canvas tote, reinforced handles.', 18.00, 'USD', 200, 'Accessories', 'bag,eco,tote', 'https://demo.shop/p/canvas-tote', true, now() - interval '18 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'NOTE-A5', 'A5 Dotted Notebook', '160gsm bleed-proof paper, lay-flat binding, 192 pages.', 14.50, 'USD', 0, 'Stationery', 'notebook,dotted,bujo', 'https://demo.shop/p/a5-notebook', true, now() - interval '17 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'BOTTLE-750', 'Insulated Water Bottle 750ml', 'Double-wall vacuum steel, keeps cold 24h / hot 12h.', 27.00, 'USD', 88, 'Outdoor', 'bottle,steel,insulated', 'https://demo.shop/p/bottle-750', true, now() - interval '15 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'TEE-ORG-BLK', 'Organic Cotton Tee (Black)', 'GOTS-certified organic cotton, unisex fit.', 22.00, 'USD', 150, 'Apparel', 'tshirt,organic,black', 'https://demo.shop/p/tee-black', true, now() - interval '14 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'TEE-ORG-BLU', 'Organic Cotton Tee (Blue)', 'GOTS-certified organic cotton, unisex fit.', 22.00, 'USD', 12, 'Apparel', 'tshirt,organic,blue', 'https://demo.shop/p/tee-blue', true, now() - interval '14 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'SOAP-GIFT', 'Artisan Soap Gift Box', 'Set of 6 cold-process soaps in a kraft gift box.', 39.00, 'USD', 40, 'Beauty', 'soap,gift,artisan', 'https://demo.shop/p/soap-gift', true, now() - interval '12 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'PLANT-POT-M', 'Ceramic Plant Pot (Medium)', 'Matte-glaze pot with drainage tray, 15cm.', 24.50, 'USD', 55, 'Home', 'plant,pot,ceramic', 'https://demo.shop/p/plant-pot-m', true, now() - interval '9 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'DISC-SEASON', 'Seasonal Clearance Bundle', 'Mixed clearance items — while stocks last.', 49.00, 'USD', 8, 'Clearance', 'bundle,sale', 'https://demo.shop/p/clearance', false, now() - interval '6 days', now() FROM users WHERE email='demo@aichathubs.local';

INSERT INTO products (user_id, sku, name, description, price, currency, stock, category, tags, active, created_at, updated_at)
SELECT id, 'BM-HEADPH', 'Wireless Headphones', 'Over-ear, 40h battery, active noise cancelling.', 89.00, 'SGD', 34, 'Electronics', 'audio,anc', true, now() - interval '26 days', now() FROM users WHERE email='sarah.chen@brightmart.com'
UNION ALL SELECT id, 'BM-SPKR', 'Portable Speaker', 'IP67 waterproof Bluetooth speaker, 12h playtime.', 59.00, 'SGD', 5, 'Electronics', 'audio,speaker', true, now() - interval '22 days', now() FROM users WHERE email='sarah.chen@brightmart.com';

-- primary images for the first few demo products
INSERT INTO product_images (product_id, url, is_primary, sort_order)
SELECT id, 'https://picsum.photos/seed/'||sku||'/600/600', true, 0 FROM products WHERE sku IN ('MUG-SET-01','CANDLE-TRIO','TOTE-CANVAS','BOTTLE-750','SOAP-GIFT');

-- ---------------------------------------------------------------------
-- 7. Customer profiles (cross-session memory / lead stages) for demo op
-- ---------------------------------------------------------------------
INSERT INTO customer_profiles (user_id, customer_phone, preferred_lang, interests, last_viewed_sku, lead_stage, notes, created_at, updated_at)
SELECT id, '61490001001', 'en', '["mugs","kitchen","gifts"]', 'MUG-SET-01', 'warm', 'Asked about shipping to Perth. Price-sensitive.', now() - interval '2 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, '61490001002', 'en', '["candles","gifts"]', 'CANDLE-TRIO', 'customer', 'Purchased candle trio. Repeat gift buyer.', now() - interval '4 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, '61490001005', 'en', '["mugs","wholesale"]', 'MUG-SET-01', 'hot', 'Cafe owner — wants 20 sets, negotiating bulk price.', now() - interval '3 hours', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, '61490001003', 'en', '["apparel"]', 'TEE-ORG-BLU', 'warm', 'Waiting on blue tee restock (low stock).', now() - interval '1 day', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, '61490001004', 'en', '["home"]', 'PLANT-POT-M', 'cold', 'Browsed plant pots, no purchase yet.', now() - interval '8 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, '6590002002', 'en', '["general"]', NULL, 'cold', 'Asked opening hours only.', now() - interval '5 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, '61490001006', 'en', '["outdoor"]', 'BOTTLE-750', 'cold', 'Abandoned enquiry weeks ago.', now() - interval '26 days', now() FROM users WHERE email='demo@aichathubs.local';

-- ---------------------------------------------------------------------
-- 8. Leads (captured purchase intent) for demo op — session_id + sku
-- ---------------------------------------------------------------------
INSERT INTO leads (user_id, session_id, sku, quantity, notes, created_at, updated_at)
SELECT (SELECT id FROM users WHERE email='demo@aichathubs.local'), s.id, 'MUG-SET-01', 20, 'Cafe bulk order — 15% wholesale discount quoted.', now() - interval '2 hours', now() FROM chat_sessions s WHERE s.customer_phone='61490001005'
UNION ALL SELECT (SELECT id FROM users WHERE email='demo@aichathubs.local'), s.id, 'CANDLE-TRIO', 1, 'Gift purchase — completed.', now() - interval '2 hours', now() FROM chat_sessions s WHERE s.customer_phone='61490001002'
UNION ALL SELECT (SELECT id FROM users WHERE email='demo@aichathubs.local'), s.id, 'MUG-SET-01', 1, 'Interested, awaiting Perth shipping quote.', now() - interval '10 minutes', now() FROM chat_sessions s WHERE s.customer_phone='61490001001'
UNION ALL SELECT (SELECT id FROM users WHERE email='demo@aichathubs.local'), s.id, 'TEE-ORG-BLU', 2, 'Wants 2 blue tees — waiting on restock.', now() - interval '35 minutes', now() FROM chat_sessions s WHERE s.customer_phone='61490001003';

-- ---------------------------------------------------------------------
-- 9. Tags + session tagging (demo op)
-- ---------------------------------------------------------------------
INSERT INTO tags (user_id, name, color, created_at)
SELECT id, 'VIP', 'gold', now() - interval '20 days' FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'Hot Lead', 'red', now() - interval '20 days' FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'Complaint', 'volcano', now() - interval '20 days' FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'Wholesale', 'purple', now() - interval '20 days' FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'Resolved', 'green', now() - interval '20 days' FROM users WHERE email='demo@aichathubs.local';

INSERT INTO session_tags (session_id, tag_id)
SELECT s.id, t.id FROM chat_sessions s, tags t WHERE s.customer_phone='61490001005' AND t.name='Hot Lead' AND t.user_id=(SELECT id FROM users WHERE email='demo@aichathubs.local')
UNION ALL SELECT s.id, t.id FROM chat_sessions s, tags t WHERE s.customer_phone='61490001005' AND t.name='Wholesale' AND t.user_id=(SELECT id FROM users WHERE email='demo@aichathubs.local')
UNION ALL SELECT s.id, t.id FROM chat_sessions s, tags t WHERE s.customer_phone='61490001002' AND t.name='VIP' AND t.user_id=(SELECT id FROM users WHERE email='demo@aichathubs.local')
UNION ALL SELECT s.id, t.id FROM chat_sessions s, tags t WHERE s.customer_phone='14155009002' AND t.name='Complaint' AND t.user_id=(SELECT id FROM users WHERE email='demo@aichathubs.local')
UNION ALL SELECT s.id, t.id FROM chat_sessions s, tags t WHERE s.customer_phone='61490001004' AND t.name='Resolved' AND t.user_id=(SELECT id FROM users WHERE email='demo@aichathubs.local');

-- ---------------------------------------------------------------------
-- 10. Knowledge base + Q&A (demo op) — enables the AI bot prompt features
-- ---------------------------------------------------------------------
INSERT INTO knowledge_bases (user_id, auto_reply_enabled, system_prompt, last_synced_at, created_at, updated_at)
SELECT id, true, 'You are the friendly assistant for Demo Store. Recommend products from our catalog, quote prices in USD, and offer gift wrapping. Always be concise and warm.', now() - interval '1 day', now() - interval '20 days', now()
FROM users WHERE email='demo@aichathubs.local'
ON CONFLICT (user_id) DO UPDATE SET auto_reply_enabled=EXCLUDED.auto_reply_enabled, system_prompt=EXCLUDED.system_prompt, last_synced_at=EXCLUDED.last_synced_at;

INSERT INTO qa_items (user_id, question, answer, category, created_at, updated_at)
SELECT id, 'Do you ship internationally?', 'Yes! We ship worldwide. Shipping is calculated at checkout based on destination and weight.', 'Shipping', now() - interval '20 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'What is your return policy?', 'We accept returns within 30 days of delivery for a full refund, provided items are unused.', 'Returns', now() - interval '20 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'Do you offer gift wrapping?', 'Yes, gift wrapping is available for $3.50 and includes a handwritten note.', 'Orders', now() - interval '20 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'How long does delivery take?', 'Domestic orders arrive in 2–4 business days; international 7–14 business days.', 'Shipping', now() - interval '20 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'Do you give bulk discounts?', 'Orders over 10 units receive 15% off. Contact us for larger wholesale quotes.', 'Pricing', now() - interval '20 days', now() FROM users WHERE email='demo@aichathubs.local'
UNION ALL SELECT id, 'What payment methods do you accept?', 'We accept all major cards, PayPal, and Apple Pay at checkout.', 'Payments', now() - interval '20 days', now() FROM users WHERE email='demo@aichathubs.local';

COMMIT;

-- ---------------------------------------------------------------------
-- Summary
-- ---------------------------------------------------------------------
SELECT 'users' AS entity, count(*) FROM users
UNION ALL SELECT 'subscriptions', count(*) FROM subscriptions
UNION ALL SELECT 'whatsapp_devices', count(*) FROM whatsapp_devices
UNION ALL SELECT 'chat_sessions', count(*) FROM chat_sessions
UNION ALL SELECT 'chat_messages', count(*) FROM chat_messages
UNION ALL SELECT 'products', count(*) FROM products
UNION ALL SELECT 'product_images', count(*) FROM product_images
UNION ALL SELECT 'customer_profiles', count(*) FROM customer_profiles
UNION ALL SELECT 'leads', count(*) FROM leads
UNION ALL SELECT 'tags', count(*) FROM tags
UNION ALL SELECT 'qa_items', count(*) FROM qa_items;
