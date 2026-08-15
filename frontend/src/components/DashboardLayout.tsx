import { useEffect } from 'react';
import { Outlet, useNavigate } from 'react-router-dom';
import { Layout, Spin } from 'antd';
import Sidebar from './Sidebar';
import { useAuthStore } from '../store/authStore';
import { useChatStore } from '../store/chatStore';
import wsService from '../services/websocket';
import './DashboardLayout.css';

const { Content } = Layout;

export default function DashboardLayout() {
  const navigate = useNavigate();
  const { user, isAuthenticated, isLoading, fetchUser } = useAuthStore();
  const { fetchStats, fetchSessions, addMessage, updateSessionFromWS } = useChatStore();

  useEffect(() => {
    fetchUser();
  }, []);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate('/login');
    }
  }, [isLoading, isAuthenticated, navigate]);

  const trialExpired = !!(user?.trial_ends_at && new Date(user.trial_ends_at) < new Date());
  const hasActiveSubscription = user?.subscription_status === 'active';
  const subscriptionBlocked = !!user && trialExpired && !hasActiveSubscription;

  useEffect(() => {
    if (subscriptionBlocked) {
      navigate('/settings?tab=billing&expired=true');
    }
  }, [subscriptionBlocked, navigate]);

  useEffect(() => {
    if (isAuthenticated && user && !subscriptionBlocked) {
      // Fetch initial stats
      fetchStats();

      // Connect to WebSocket
      wsService.connect(user.id);

      // Listen for new messages
      const unsubMessage = wsService.onMessage((message) => {
        addMessage(message);
      });

      // Listen for session updates (new chats, new messages on any session)
      const unsubSessionUpdate = wsService.onSessionUpdate((payload) => {
        updateSessionFromWS(payload);
        fetchSessions();
      });

      return () => {
        unsubMessage();
        unsubSessionUpdate();
        wsService.disconnect();
      };
    }
  }, [isAuthenticated, user, subscriptionBlocked]);

  if (isLoading) {
    return (
      <div className="loading-container">
        <Spin size="large" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  return (
    <Layout className="dashboard-layout">
      <Sidebar />
      <Layout className="main-layout">
        <Content className="main-content">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}

