import { useEffect } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Button, Typography, Space } from 'antd';
import {
  DashboardOutlined,
  UserOutlined,
  MobileOutlined,
  CreditCardOutlined,
  BarChartOutlined,
  SettingOutlined,
  LogoutOutlined,
  SafetyCertificateOutlined,
  RobotOutlined,
  HourglassOutlined,
} from '@ant-design/icons';
import { useAdminStore } from '../store/adminStore';

const { Header, Sider, Content } = Layout;

const menuItems = [
  { key: '/admin/dashboard', icon: <DashboardOutlined />, label: 'Dashboard' },
  { key: '/admin/users', icon: <UserOutlined />, label: 'Users' },
  { key: '/admin/devices', icon: <MobileOutlined />, label: 'WhatsApp Devices' },
  { key: '/admin/payments', icon: <CreditCardOutlined />, label: 'Payments' },
  { key: '/admin/analytics', icon: <BarChartOutlined />, label: 'Chat Analytics' },
  { key: '/admin/bot-prompt', icon: <RobotOutlined />, label: 'AI Bot Prompt' },
  { key: '/admin/ai-reply-delay', icon: <HourglassOutlined />, label: 'AI Reply Delay' },
  { key: '/admin/system', icon: <SettingOutlined />, label: 'System' },
];

export default function AdminLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { isAdminAuthenticated, isLoading, checkAdmin, adminLogout } = useAdminStore();

  useEffect(() => {
    checkAdmin();
  }, [checkAdmin]);

  useEffect(() => {
    if (!isLoading && !isAdminAuthenticated) {
      navigate('/admin/login');
    }
  }, [isLoading, isAdminAuthenticated, navigate]);

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Typography.Text>Loading...</Typography.Text>
      </div>
    );
  }

  if (!isAdminAuthenticated) return null;

  const handleLogout = () => {
    adminLogout();
    navigate('/admin/login');
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={240} theme="dark" breakpoint="lg" collapsedWidth={80}>
        <div style={{ padding: '20px 16px', textAlign: 'center', borderBottom: '1px solid rgba(255,255,255,0.1)' }}>
          <Space>
            <SafetyCertificateOutlined style={{ fontSize: 24, color: '#1890ff' }} />
            <Typography.Text strong style={{ color: '#fff', fontSize: 16 }}>Admin Panel</Typography.Text>
          </Space>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ marginTop: 8 }}
        />
      </Sider>
      <Layout>
        <Header style={{
          background: '#fff',
          padding: '0 24px',
          display: 'flex',
          justifyContent: 'flex-end',
          alignItems: 'center',
          borderBottom: '1px solid #f0f0f0',
          boxShadow: '0 1px 4px rgba(0,0,0,0.05)',
        }}>
          <Button
            type="text"
            icon={<LogoutOutlined />}
            onClick={handleLogout}
            danger
          >
            Logout
          </Button>
        </Header>
        <Content style={{ margin: 24, minHeight: 280 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
