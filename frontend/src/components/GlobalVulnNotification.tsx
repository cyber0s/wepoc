import { useState, useEffect } from 'react';
import { Typography, Tag } from 'antd';
import { BugOutlined } from '@ant-design/icons';
import * as api from '../services/api';
import { ScanEvent } from '../types';

const { Text } = Typography;

interface VulnNotification {
  id: string;
  vuln_number: number;
  template_id: string;
  name: string;
  severity: string;
  host: string;
  timestamp: string;
}

const GlobalVulnNotification = () => {
  const [vulnNotifications, setVulnNotifications] = useState<VulnNotification[]>([]);

  useEffect(() => {
    // Listen to global scan events
    const unsubscribe = api.onScanEvent((event: ScanEvent) => {
      if (event.event_type === 'vuln_found') {
        // Show real-time notification when vulnerability is found
        const vulnData = event.data;
        const notification: VulnNotification = {
          id: `${event.task_id}-${vulnData.vuln_number}-${Date.now()}`,
          vuln_number: vulnData.vuln_number,
          template_id: vulnData.template_id,
          name: vulnData.name,
          severity: vulnData.severity,
          host: vulnData.host,
          timestamp: vulnData.timestamp,
        };

        // Add notification to the queue (only keep the latest one visible)
        setVulnNotifications([notification]);

        // Auto-dismiss after 3 seconds with slide-out animation
        setTimeout(() => {
          setVulnNotifications([]);
        }, 3000);
      }
    });

    return () => {
      // Cleanup if needed
    };
  }, []);

  const getSeverityColor = (severity: string) => {
    switch (severity?.toLowerCase()) {
      case 'critical':
        return '#d32f2f';
      case 'high':
        return '#f57c00';
      case 'medium':
        return '#fbc02d';
      case 'low':
        return '#1976d2';
      default:
        return '#757575';
    }
  };

  if (vulnNotifications.length === 0) {
    return null;
  }

  return (
    <>
      {/* Vulnerability Notifications - Slide in from right like WeChat */}
      <div
        style={{
          position: 'fixed',
          top: 80,
          right: 24,
          zIndex: 9999, // Very high z-index to ensure it's above everything
          width: 340,
          maxWidth: '90vw',
        }}
      >
        {vulnNotifications.map(notification => (
          <div
            key={notification.id}
            style={{
              animation: 'slideInFromRight 0.4s cubic-bezier(0.16, 1, 0.3, 1)',
              marginBottom: 8,
            }}
          >
            <div
              style={{
                background: '#fff',
                borderRadius: 8,
                padding: '12px 16px',
                boxShadow: '0 4px 12px rgba(0,0,0,0.15), 0 0 1px rgba(0,0,0,0.1)',
                borderLeft: `4px solid ${getSeverityColor(notification.severity)}`,
                cursor: 'pointer',
                transition: 'transform 0.2s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateX(-4px)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateX(0)';
              }}
            >
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
                <BugOutlined style={{ color: getSeverityColor(notification.severity), fontSize: 20, marginTop: 2 }} />
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                    <Text strong style={{ fontSize: 14 }}>
                      发现漏洞 #{notification.vuln_number}
                    </Text>
                    <Tag
                      color={getSeverityColor(notification.severity)}
                      style={{ margin: 0, fontSize: 10, padding: '0 6px', lineHeight: '18px' }}
                    >
                      {notification.severity?.toUpperCase() || 'UNKNOWN'}
                    </Tag>
                  </div>
                  <div style={{ fontSize: 13, marginBottom: 4, color: '#333', fontWeight: 500 }}>
                    {notification.name || notification.template_id}
                  </div>
                  <Text type="secondary" style={{ fontSize: 11 }}>
                    {notification.host}
                  </Text>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      <style>
        {`
          @keyframes slideInFromRight {
            from {
              opacity: 0;
              transform: translateX(400px);
            }
            to {
              opacity: 1;
              transform: translateX(0);
            }
          }
        `}
      </style>
    </>
  );
};

export default GlobalVulnNotification;
