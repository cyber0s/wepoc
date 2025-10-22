import { useEffect, useRef, useState } from 'react';
import { Card, List, Tag, Collapse, Typography, Input, Select, Badge, Empty } from 'antd';
import {
  BugOutlined,
  InfoCircleOutlined,
  ExclamationCircleOutlined,
  WarningOutlined,
  CodeOutlined,
  SearchOutlined
} from '@ant-design/icons';
import { ScanLogEntry } from '../types';

const { Panel } = Collapse;
const { Text, Paragraph } = Typography;
const { Search } = Input;

interface ScanLogsProps {
  logs: ScanLogEntry[];
  autoScroll?: boolean;
  maxHeight?: number;
}

const ScanLogs: React.FC<ScanLogsProps> = ({ logs, autoScroll = true, maxHeight = 400 }) => {
  const [filteredLogs, setFilteredLogs] = useState<ScanLogEntry[]>(logs);
  const [levelFilter, setLevelFilter] = useState<string>('all');
  const [searchText, setSearchText] = useState<string>('');
  const logsEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setFilteredLogs(logs);
  }, [logs]);

  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [filteredLogs, autoScroll]);

  useEffect(() => {
    let filtered = logs;

    // Filter by level
    if (levelFilter !== 'all') {
      filtered = filtered.filter(log => log.level === levelFilter);
    }

    // Filter by search text
    if (searchText) {
      filtered = filtered.filter(log =>
        log.message.toLowerCase().includes(searchText.toLowerCase()) ||
        log.template_id?.toLowerCase().includes(searchText.toLowerCase()) ||
        log.target?.toLowerCase().includes(searchText.toLowerCase())
      );
    }

    setFilteredLogs(filtered);
  }, [logs, levelFilter, searchText]);

  const getLevelIcon = (level: string) => {
    switch (level) {
      case 'INFO':
        return <InfoCircleOutlined style={{ color: '#1890ff' }} />;
      case 'WARN':
        return <WarningOutlined style={{ color: '#faad14' }} />;
      case 'ERROR':
        return <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />;
      case 'DEBUG':
        return <CodeOutlined style={{ color: '#52c41a' }} />;
      default:
        return <InfoCircleOutlined />;
    }
  };

  const getLevelColor = (level: string) => {
    switch (level) {
      case 'INFO':
        return 'blue';
      case 'WARN':
        return 'orange';
      case 'ERROR':
        return 'red';
      case 'DEBUG':
        return 'green';
      default:
        return 'default';
    }
  };

  const renderLogItem = (log: ScanLogEntry, index: number) => {
    const hasDetails = log.request || log.response;
    const timestamp = new Date(log.timestamp).toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    });

    return (
      <List.Item
        key={index}
        style={{
          padding: '6px 12px',
          borderLeft: log.is_vuln_found ? '3px solid #ff4d4f' : 'none',
          backgroundColor: log.is_vuln_found ? '#fff1f0' : 'transparent',
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <div style={{ width: '100%' }}>
          <div style={{ 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'space-between',
            marginBottom: 2
          }}>
            <div style={{ display: 'flex', alignItems: 'center', flex: 1 }}>
              {getLevelIcon(log.level)}
              <Tag 
                color={getLevelColor(log.level)} 
                style={{ 
                  marginLeft: 6, 
                  fontSize: 10, 
                  padding: '1px 4px',
                  lineHeight: '16px'
                }}
              >
                {log.level}
              </Tag>
              {log.template_id && (
                <Tag 
                  color="purple" 
                  style={{ 
                    marginLeft: 6, 
                    fontSize: 10, 
                    padding: '1px 4px',
                    lineHeight: '16px'
                  }}
                >
                  {log.template_id}
                </Tag>
              )}
              {log.is_vuln_found && (
                <Badge 
                  count="漏洞" 
                  style={{ 
                    backgroundColor: '#ff4d4f', 
                    marginLeft: 6,
                    fontSize: 9,
                    minWidth: 28,
                    height: 16,
                    lineHeight: '16px'
                  }} 
                />
              )}
            </div>
            <Text type="secondary" style={{ fontSize: 10, lineHeight: '16px' }}>
              {timestamp}
            </Text>
          </div>

          <div style={{ marginLeft: 20 }}>
            <Text style={{ fontSize: 12, lineHeight: '18px' }}>{log.message}</Text>
            {log.target && (
              <Text type="secondary" style={{ 
                fontSize: 10, 
                lineHeight: '16px',
                display: 'block',
                marginTop: 2
              }}>
                目标: {log.target}
              </Text>
            )}
          </div>

          {hasDetails && (
            <Collapse
              ghost
              size="small"
              style={{ marginTop: 4, marginLeft: 20 }}
              items={[
                {
                  key: 'details',
                  label: <Text strong style={{ fontSize: 11 }}>查看详情</Text>,
                  children: (
                    <div>
                      {log.request && (
                        <div style={{ marginBottom: 12 }}>
                          <Text strong style={{ fontSize: 11 }}>请求:</Text>
                          <Paragraph
                            code
                            copyable
                            style={{
                              marginTop: 4,
                              backgroundColor: '#f5f5f5',
                              padding: 8,
                              borderRadius: 3,
                              whiteSpace: 'pre-wrap',
                              fontSize: 10,
                              maxHeight: 150,
                              overflow: 'auto',
                            }}
                          >
                            {log.request}
                          </Paragraph>
                        </div>
                      )}
                      {log.response && (
                        <div>
                          <Text strong style={{ fontSize: 11 }}>响应:</Text>
                          <Paragraph
                            code
                            copyable
                            style={{
                              marginTop: 4,
                              backgroundColor: '#f5f5f5',
                              padding: 8,
                              borderRadius: 3,
                              whiteSpace: 'pre-wrap',
                              fontSize: 10,
                              maxHeight: 150,
                              overflow: 'auto',
                            }}
                          >
                            {log.response}
                          </Paragraph>
                        </div>
                      )}
                    </div>
                  ),
                },
              ]}
            />
          )}
        </div>
      </List.Item>
    );
  };

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span style={{ fontSize: 14, fontWeight: 500 }}>
            <CodeOutlined style={{ marginRight: 6, fontSize: 14 }} />
            实时日志 ({filteredLogs.length})
          </span>
          <div style={{ display: 'flex', gap: 6 }}>
            <Select
              value={levelFilter}
              onChange={setLevelFilter}
              style={{ width: 100, fontSize: 12 }}
              size="small"
            >
              <Select.Option value="all">全部级别</Select.Option>
              <Select.Option value="INFO">INFO</Select.Option>
              <Select.Option value="DEBUG">DEBUG</Select.Option>
              <Select.Option value="WARN">WARN</Select.Option>
              <Select.Option value="ERROR">ERROR</Select.Option>
            </Select>
            <Search
              placeholder="搜索日志..."
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 160, fontSize: 12 }}
              size="small"
              prefix={<SearchOutlined style={{ fontSize: 12 }} />}
            />
          </div>
        </div>
      }
      style={{ marginTop: 12 }}
      bodyStyle={{ padding: '8px 12px' }}
    >
      <div style={{ maxHeight, overflow: 'auto' }}>
        {filteredLogs.length > 0 ? (
          <>
            <List
              size="small"
              dataSource={filteredLogs}
              renderItem={renderLogItem}
              style={{ backgroundColor: '#fafafa', borderRadius: 4 }}
            />
            <div ref={logsEndRef} />
          </>
        ) : (
          <Empty 
            description="暂无日志" 
            style={{ padding: '20px 0' }}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        )}
      </div>
    </Card>
  );
};

export default ScanLogs;
