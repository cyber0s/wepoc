import { Layout, Menu, Button } from 'antd';
import {
  FileTextOutlined,
  ScanOutlined,
  BarChartOutlined,
  SettingOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';

const { Content, Sider } = Layout;

interface AppLayoutProps {
  children: React.ReactNode;
}

const AppLayout = ({ children }: AppLayoutProps) => {
  const navigate = useNavigate();
  const location = useLocation();
  const [collapsed, setCollapsed] = useState(false);

  const menuItems: MenuProps['items'] = [
    {
      key: '/templates',
      icon: <FileTextOutlined />,
      label: '模板管理',
    },
    {
      key: '/tasks',
      icon: <ScanOutlined />,
      label: '扫描任务',
    },
    {
      key: '/results',
      icon: <BarChartOutlined />,
      label: '扫描结果',
    },
    {
      key: '/settings',
      icon: <SettingOutlined />,
      label: '设置',
    },
  ];

  const handleMenuClick: MenuProps['onClick'] = (e) => {
    navigate(e.key);
  };

  const toggleCollapsed = () => {
    setCollapsed(!collapsed);
  };

  return (
    <Layout className="app-layout" style={{ height: '100vh', backgroundColor: '#f5f5f5' }}>
      <style>{`
        /* 展开状态的菜单项 */
        .sidebar-menu .ant-menu-item {
          padding-left: 16px !important;
          padding-right: 16px !important;
          margin: 4px 8px !important;
          height: 44px !important;
          line-height: 44px !important;
          border-radius: 6px !important;
          display: flex !important;
          align-items: center !important;
          width: auto !important;
          justify-content: flex-start !important;
        }
        
        /* 图标样式 - 展开状态 */
        .sidebar-menu .ant-menu-item .anticon {
          font-size: 18px !important;
          width: 18px !important;
          min-width: 18px !important;
          margin-right: 12px !important;
          display: inline-flex !important;
          align-items: center !important;
          justify-content: center !important;
          flex-shrink: 0 !important;
        }
        
        /* 文字样式 - 展开状态 */
        .sidebar-menu .ant-menu-item .ant-menu-title-content {
          font-size: 14px !important;
          margin-left: 0 !important;
          flex: 1 !important;
        }
        
        /* 收缩状态的菜单项 */
        .sidebar-menu.ant-menu-inline-collapsed .ant-menu-item {
          padding: 0 !important;
          margin: 4px auto !important;
          width: 48px !important;
          height: 48px !important;
          display: flex !important;
          justify-content: center !important;
          align-items: center !important;
          border-radius: 6px !important;
        }
        
        .sidebar-menu.ant-menu-inline-collapsed .ant-menu-item .anticon {
          margin: 0 !important;
          font-size: 20px !important;
          width: 20px !important;
          height: 20px !important;
          display: flex !important;
          align-items: center !important;
          justify-content: center !important;
        }
        
        /* 收缩状态下隐藏文字 */
        .sidebar-menu.ant-menu-inline-collapsed .ant-menu-item .ant-menu-title-content {
          display: none !important;
        }
        
        /* 选中状态 */
        .sidebar-menu .ant-menu-item-selected {
          background-color: #e6f4ff !important;
        }
        
        .sidebar-menu .ant-menu-item-selected .anticon,
        .sidebar-menu .ant-menu-item-selected .ant-menu-title-content {
          color: #1890ff !important;
        }
        
        /* 悬停状态 */
        .sidebar-menu .ant-menu-item:hover:not(.ant-menu-item-selected) {
          background-color: #f5f5f5 !important;
        }
        
        .sidebar-menu .ant-menu-item::after {
          display: none !important;
        }
        
        /* 确保收缩状态下图标居中 */
        .sidebar-menu.ant-menu-inline-collapsed .ant-menu-item .ant-menu-item-icon {
          margin: 0 !important;
        }
      `}</style>
      
      <Layout style={{ height: '100vh' }}>
        {/* 左侧导航栏 */}
        <Sider
          width={collapsed ? 80 : 200}
          collapsed={collapsed}
          collapsible
          trigger={null}
          style={{
            background: 'white',
            borderRight: '1px solid #e8e8e8',
            boxShadow: '2px 0 8px rgba(0,0,0,0.05)',
            transition: 'all 0.2s'
          }}
        >
          {/* Logo和标题 */}
          <div style={{
            padding: '20px 16px',
            borderBottom: '1px solid #f0f0f0',
            marginBottom: '8px',
            display: 'flex',
            justifyContent: collapsed ? 'center' : 'flex-start',
            alignItems: 'center',
            transition: 'all 0.2s'
          }}>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: '10px'
            }}>
              <div style={{
                width: '36px',
                height: '36px',
                borderRadius: '8px',
                backgroundColor: '#1890ff',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'white',
                fontSize: '18px',
                fontWeight: 'bold',
                flexShrink: 0,
                boxShadow: '0 2px 8px rgba(24, 144, 255, 0.3)'
              }}>
                W
              </div>
              {!collapsed && (
                <span style={{
                  fontSize: '18px',
                  fontWeight: '600',
                  color: '#262626',
                  whiteSpace: 'nowrap',
                  letterSpacing: '0.5px'
                }}>
                  wepoc
                </span>
              )}
            </div>
          </div>

          <Menu
            mode="inline"
            selectedKeys={[location.pathname]}
            items={menuItems}
            onClick={handleMenuClick}
            style={{
              height: 'calc(100% - 140px)',
              borderRight: 0,
              padding: '8px 0',
              backgroundColor: 'white'
            }}
            theme="light"
            inlineCollapsed={collapsed}
            className="sidebar-menu"
          />

          {/* 收缩按钮 */}
          <div style={{
            position: 'absolute',
            bottom: '16px',
            left: '12px',
            right: '12px',
            display: 'flex',
            justifyContent: 'center'
          }}>
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={toggleCollapsed}
              style={{
                width: collapsed ? '56px' : '100%',
                height: '36px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '13px',
                color: '#595959',
                border: '1px solid #d9d9d9',
                borderRadius: '6px',
                backgroundColor: 'white',
                transition: 'all 0.2s',
                fontWeight: '500'
              }}
            >
              {!collapsed && <span style={{ marginLeft: '6px' }}>收起</span>}
            </Button>
          </div>
        </Sider>

        {/* 主内容区域 */}
        <Layout style={{ backgroundColor: '#f5f5f5' }}>
          <Content
            style={{
              margin: 0,
              padding: 0,
              height: '100%',
              overflow: 'auto',
            }}
          >
            {children}
          </Content>
        </Layout>
      </Layout>
    </Layout>
  );
};

export default AppLayout;