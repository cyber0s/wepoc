import { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, theme } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import AppLayout from './components/layout/AppLayout';
import GlobalVulnNotification from './components/GlobalVulnNotification';
import CybersecurityLawModal from './components/CybersecurityLawModal';
import Templates from './pages/Templates/Templates';
import ScanTasks from './pages/ScanTasks/ScanTasks';
import Results from './pages/Results/Results';
import Settings from './pages/Settings/Settings';
import './App.css';

function App() {
  const [showLawModal, setShowLawModal] = useState(false);
  const [lawAccepted, setLawAccepted] = useState(false);

  useEffect(() => {
    // 检查是否已经同意过网络安全法协议
    const accepted = localStorage.getItem('cybersecurity-law-accepted');
    if (accepted === 'true') {
      setLawAccepted(true);
    } else {
      setShowLawModal(true);
    }
  }, []);

  const handleLawAccept = () => {
    localStorage.setItem('cybersecurity-law-accepted', 'true');
    setLawAccepted(true);
    setShowLawModal(false);
  };

  const handleLawReject = () => {
    // 用户不同意协议，退出程序
    window.close();
  };

  // 如果用户还没有同意协议，显示协议确认弹窗
  if (!lawAccepted) {
    return (
      <ConfigProvider
        locale={zhCN}
        theme={{
          algorithm: theme.defaultAlgorithm,
          token: {
            colorPrimary: '#1890ff',
            borderRadius: 6,
          },
        }}
      >
        <CybersecurityLawModal
          visible={showLawModal}
          onAccept={handleLawAccept}
          onReject={handleLawReject}
        />
      </ConfigProvider>
    );
  }

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: '#1890ff',
          borderRadius: 6,
        },
      }}
    >
      <Router>
        {/* Global Vulnerability Notification - visible on all pages */}
        <GlobalVulnNotification />

        <AppLayout>
          <Routes>
            <Route path="/" element={<Navigate to="/templates" replace />} />
            <Route path="/templates" element={<Templates />} />
            <Route path="/tasks" element={<ScanTasks />} />
            <Route path="/results" element={<Results />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </AppLayout>
      </Router>
    </ConfigProvider>
  );
}

export default App;
