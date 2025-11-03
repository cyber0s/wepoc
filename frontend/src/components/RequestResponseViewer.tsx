import { Modal, Tabs, Typography, Tag, Space, Button } from 'antd';
import { CopyOutlined, DownloadOutlined } from '@ant-design/icons';
import { HTTPRequestLog } from '../types';
import './RequestResponseViewer.css';

const { TabPane } = Tabs;
const { Title, Text } = Typography;

interface RequestResponseViewerProps {
  visible: boolean;
  log: HTTPRequestLog;
  onClose: () => void;
}

const RequestResponseViewer = ({ visible, log, onClose }: RequestResponseViewerProps) => {
  const handleCopy = (text: string, type: string) => {
    navigator.clipboard.writeText(text).then(() => {
      // 使用Ant Design的message需要在实际使用时导入
      console.log(`${type} 已复制到剪贴板`);
    });
  };

  const getSeverityColor = (severity: string) => {
    switch (severity?.toLowerCase()) {
      case 'critical': return 'red';
      case 'high': return 'orange';
      case 'medium': return 'gold';
      case 'low': return 'blue';
      default: return 'default';
    }
  };

  const formatTimestamp = (time: string) => {
    try {
      return new Date(time).toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    } catch {
      return time;
    }
  };

  return (
    <Modal
      title="HTTP请求详情"
      open={visible}
      onCancel={onClose}
      width={1200}
      footer={[
        <Button key="close" onClick={onClose}>
          关闭
        </Button>,
      ]}
      className="request-response-viewer"
    >
      <div className="viewer-header">
        <Space direction="vertical" size="small" style={{ width: '100%' }}>
          <Space>
            <Text strong>POC ID:</Text>
            <Text code>{log.template_id}</Text>
            <Tag color={getSeverityColor(log.severity)}>
              {log.severity?.toUpperCase()}
            </Tag>
            {log.is_vuln_found && <Tag color="red">发现漏洞</Tag>}
          </Space>
          <Space>
            <Text strong>目标:</Text>
            <Text>{log.target}</Text>
          </Space>
          <Space>
            <Text strong>时间:</Text>
            <Text type="secondary">{formatTimestamp(log.timestamp)}</Text>
            <Text strong>耗时:</Text>
            <Text type="secondary">{log.duration_ms > 0 ? `${log.duration_ms}ms` : '-'}</Text>
          </Space>
        </Space>
      </div>

      <Tabs defaultActiveKey="1" className="viewer-tabs">
        <TabPane tab="HTTP请求" key="1">
          <div className="code-container">
            <div className="code-toolbar">
              <Space>
                <Tag>{log.method}</Tag>
                <Text type="secondary">状态码: {log.status_code}</Text>
              </Space>
              <Button
                size="small"
                icon={<CopyOutlined />}
                onClick={() => handleCopy(log.request, '请求')}
              >
                复制
              </Button>
            </div>
            <pre className="code-content">{log.request || '无请求数据'}</pre>
          </div>
        </TabPane>

        <TabPane tab="HTTP响应" key="2">
          <div className="code-container">
            <div className="code-toolbar">
              <Space>
                <Tag color={log.status_code >= 200 && log.status_code < 300 ? 'success' : 'error'}>
                  {log.status_code}
                </Tag>
                <Text type="secondary">响应大小: {log.response?.length || 0} 字节</Text>
              </Space>
              <Button
                size="small"
                icon={<CopyOutlined />}
                onClick={() => handleCopy(log.response, '响应')}
              >
                复制
              </Button>
            </div>
            <pre className="code-content">{log.response || '无响应数据'}</pre>
          </div>
        </TabPane>

        <TabPane tab="请求信息" key="3">
          <div className="info-container">
            <Space direction="vertical" size="middle" style={{ width: '100%' }}>
              <div>
                <Text strong>请求ID:</Text> <Text>{log.id}</Text>
              </div>
              <div>
                <Text strong>任务ID:</Text> <Text>{log.task_id}</Text>
              </div>
              <div>
                <Text strong>模板ID:</Text> <Text code>{log.template_id}</Text>
              </div>
              <div>
                <Text strong>模板名称:</Text> <Text>{log.template_name}</Text>
              </div>
              <div>
                <Text strong>目标URL:</Text> <Text copyable>{log.target}</Text>
              </div>
              <div>
                <Text strong>HTTP方法:</Text> <Tag>{log.method}</Tag>
              </div>
              <div>
                <Text strong>状态码:</Text> <Tag color={log.status_code >= 200 && log.status_code < 300 ? 'success' : 'error'}>{log.status_code}</Tag>
              </div>
              <div>
                <Text strong>严重程度:</Text> <Tag color={getSeverityColor(log.severity)}>{log.severity?.toUpperCase()}</Tag>
              </div>
              <div>
                <Text strong>发现漏洞:</Text> {log.is_vuln_found ? <Tag color="red">是</Tag> : <Tag>否</Tag>}
              </div>
              <div>
                <Text strong>请求时间:</Text> <Text>{formatTimestamp(log.timestamp)}</Text>
              </div>
              <div>
                <Text strong>耗时:</Text> <Text>{log.duration_ms > 0 ? `${log.duration_ms}ms` : '未测量'}</Text>
              </div>
            </Space>
          </div>
        </TabPane>
      </Tabs>
    </Modal>
  );
};

export default RequestResponseViewer;
