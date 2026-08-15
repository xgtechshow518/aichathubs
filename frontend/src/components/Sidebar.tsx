import { useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Layout, Menu, Badge, Avatar, Typography, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import {
  HomeOutlined,
  MessageOutlined,
  TeamOutlined,
  RobotOutlined,
  ShoppingOutlined,
  FundOutlined,
  BarChartOutlined,
  ApiOutlined,
  SettingOutlined,
  LogoutOutlined,
  UserOutlined,
  CrownOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../store/authStore';
import { useChatStore } from '../store/chatStore';
import api from '../services/api';
import wsService from '../services/websocket';
import './Sidebar.css';

const { Sider } = Layout;
const { Text } = Typography;

const menuItems = [
  { key: '/dashboard', icon: <HomeOutlined />, label: 'Home' },
  { key: '/chats', icon: <MessageOutlined />, label: 'Chats', badge: true },
  { key: '/customers', icon: <TeamOutlined />, label: 'Customers' },
  { key: '/bots', icon: <RobotOutlined />, label: 'Bots' },
  { key: '/products', icon: <ShoppingOutlined />, label: 'Products' },
  { key: '/leads', icon: <FundOutlined />, label: 'Leads' },
  { key: '/reports', icon: <BarChartOutlined />, label: 'Reports' },
  { key: '/integrations', icon: <ApiOutlined />, label: 'Integrations' },
  { key: '/settings', icon: <SettingOutlined />, label: 'Settings' },
];

export default function Sidebar() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { stats } = useChatStore();
  const [connectedDevices, setConnectedDevices] = useState(0);

  useEffect(() => {
    const loadDevices = async () => {
      try {
        const { devices } = await api.getWhatsAppDevices();
        setConnectedDevices(devices.filter((d) => d.status === 'connected').length);
      } catch { /* ignore */ }
    };
    loadDevices();

    const unsub = wsService.onWhatsAppStatus(() => {
      loadDevices();
    });
    return unsub;
  }, []);

  const handleMenuClick = ({ key }: { key: string }) => {
    navigate(key);
  };

  const userMenuItems: MenuProps['items'] = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: 'Profile',
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: 'Settings',
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: 'Logout',
      danger: true,
    },
  ];

  const handleUserMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'logout') {
      logout();
      navigate('/login');
    } else if (key === 'settings') {
      navigate('/settings');
    }
  };

  const items = menuItems.map((item) => ({
    key: item.key,
    icon: item.icon,
    label: (
      <span className="menu-label">
        {item.label}
        {item.badge && stats && stats.unread_chats > 0 && (
          <Badge count={stats.unread_chats} size="small" className="menu-badge" />
        )}
      </span>
    ),
  }));

  return (
    <Sider className="app-sidebar" width={220} theme="light">
      {/* Logo */}
      <div className="sidebar-logo">
        <span className="logo-icon">💬</span>
        <span className="logo-text">AIChatsHub</span>
      </div>

      {/* Upgrade Banner - hide for active/trialing users */}
      {user?.subscription_status !== 'active' && user?.subscription_status !== 'trialing' && (
        <div className="upgrade-banner" onClick={() => navigate('/settings')} style={{ cursor: 'pointer' }}>
          <CrownOutlined className="upgrade-icon" />
          <div className="upgrade-text">
            <Text strong>Upgrade Plan</Text>
            <Text type="secondary" className="upgrade-price">From $9.99/mo</Text>
          </div>
        </div>
      )}

      {/* Navigation Menu */}
      <Menu
        mode="inline"
        selectedKeys={[location.pathname]}
        items={items}
        onClick={handleMenuClick}
        className="sidebar-menu"
      />

      {/* WhatsApp Devices Info */}
      <div className="project-info">
        <div className="project-header">
          <Text type="secondary">WhatsApp Devices</Text>
        </div>
        <div className="project-stats">
          <div className="stat-item">
            <Text type="secondary">Connected</Text>
            <Text>{connectedDevices}/{user?.max_devices ?? 1}</Text>
          </div>
          <div className="stat-item">
            <Text type="secondary">Plan</Text>
            <Text style={{ textTransform: 'capitalize' }}>{user?.subscription_plan || 'Trial'}</Text>
          </div>
        </div>
      </div>

      {/* User Profile */}
      <div className="sidebar-footer">
        <Dropdown
          menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
          trigger={['click']}
          placement="topRight"
        >
          <div className="user-profile">
            <Avatar
              size={36}
              src={user?.avatar_url}
              icon={!user?.avatar_url && <UserOutlined />}
            />
            <div className="user-info">
              <Text strong className="user-name">{user?.name || 'User'}</Text>
              <Text type="secondary" className="user-status">Online</Text>
            </div>
          </div>
        </Dropdown>
      </div>
    </Sider>
  );
}

