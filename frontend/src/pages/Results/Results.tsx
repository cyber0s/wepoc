import { useState, useEffect } from 'react';
import {
  Card,
  Typography,
  Table,
  Tag,
  Button,
  Space,
  Drawer,
  Descriptions,
  message,
  Row,
  Col,
  Statistic,
  Collapse,
  Empty,
  Tabs,
  Select,
} from 'antd';
import {
  EyeOutlined,
  ReloadOutlined,
  BugOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import { TaskResult, NucleiResult } from '../../types';
import * as api from '../../services/api';
import './Results.css';

const { Title, Text, Paragraph } = Typography;
const { Panel } = Collapse;
const { Option } = Select;

const Results = () => {
  const [results, setResults] = useState<TaskResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedResult, setSelectedResult] = useState<TaskResult | null>(null);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc');

  useEffect(() => {
    loadResults();
  }, []);

  const loadResults = async () => {
    setLoading(true);
    try {
      const allResults = await api.getAllScanResults();
      const sortedResults = sortResultsByTime(allResults || [], sortOrder);
      setResults(sortedResults);
    } catch (error) {
      console.error('Failed to load results:', error);
      message.error('加载扫描结果失败');
    } finally {
      setLoading(false);
    }
  };

  // Sort results by start time
  const sortResultsByTime = (resultList: TaskResult[], order: 'desc' | 'asc') => {
    return [...resultList].sort((a, b) => {
      const timeA = new Date(a.start_time).getTime();
      const timeB = new Date(b.start_time).getTime();
      return order === 'desc' ? timeB - timeA : timeA - timeB;
    });
  };

  // Handle sort order change
  const handleSortChange = (order: 'desc' | 'asc') => {
    setSortOrder(order);
    const sortedResults = sortResultsByTime(results, order);
    setResults(sortedResults);
  };

  const handleViewDetails = (result: TaskResult) => {
    setSelectedResult(result);
    setDrawerVisible(true);
  };

  // Format duration to show seconds with 2 decimal places
  const formatDuration = (duration: string) => {
    // Parse duration like "1.234567s" or "1m2.345s"
    const match = duration.match(/(\d+\.?\d*)s/);
    if (match) {
      const seconds = parseFloat(match[1]);
      return `${seconds.toFixed(2)}秒`;
    }

    // Handle minutes
    const minMatch = duration.match(/(\d+)m(\d+\.?\d*)s/);
    if (minMatch) {
      const minutes = parseInt(minMatch[1]);
      const seconds = parseFloat(minMatch[2]);
      return `${minutes}分${seconds.toFixed(2)}秒`;
    }

    return duration;
  };

  const getSeverityColor = (severity: string) => {
    switch (severity?.toLowerCase()) {
      case 'critical':
        return '#ff4d4f';
      case 'high':
        return '#ff7a45';
      case 'medium':
        return '#faad14';
      case 'low':
        return '#52c41a';
      case 'info':
        return '#1890ff';
      default:
        return '#d9d9d9';
    }
  };

  const columns = [
    {
      title: '任务名称',
      dataIndex: 'task_name',
      key: 'task_name',
      width: 200,
      render: (text: string) => <Text strong style={{ fontSize: 13 }}>{text}</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => (
        <Tag 
          color={status === 'completed' ? 'success' : 'default'}
          style={{ fontSize: 11, padding: '2px 6px' }}
        >
          {status === 'completed' ? '已完成' : status}
        </Tag>
      ),
    },
    {
      title: '目标',
      dataIndex: 'target_count',
      key: 'target_count',
      width: 60,
      render: (count: number) => <Text style={{ fontSize: 12 }}>{count}</Text>,
    },
    {
      title: '模板',
      dataIndex: 'template_count',
      key: 'template_count',
      width: 60,
      render: (count: number) => <Text style={{ fontSize: 12 }}>{count}</Text>,
    },
    {
      title: '漏洞',
      dataIndex: 'found_vulns',
      key: 'found_vulns',
      width: 80,
      render: (count: number) => (
        <Tag 
          color={count > 0 ? 'error' : 'default'}
          style={{ fontSize: 11, padding: '2px 6px' }}
        >
          <BugOutlined /> {count}
        </Tag>
      ),
    },
    {
      title: '耗时',
      dataIndex: 'duration',
      key: 'duration',
      width: 100,
      render: (duration: string) => (
        <Text type="secondary" style={{ fontSize: 11 }}>
          {formatDuration(duration)}
        </Text>
      ),
    },
    {
      title: '扫描时间',
      dataIndex: 'start_time',
      key: 'start_time',
      width: 150,
      render: (time: string) => (
        <Text style={{ fontSize: 11 }}>
          {new Date(time).toLocaleString('zh-CN', {
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
          })}
        </Text>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 80,
      render: (_: any, record: TaskResult) => (
        <Button
          type="link"
          icon={<EyeOutlined />}
          onClick={() => handleViewDetails(record)}
          style={{ fontSize: 12, padding: '4px 8px' }}
        >
          详情
        </Button>
      ),
    },
  ];

  return (
    <div style={{ padding: 0, backgroundColor: '#f5f5f5', minHeight: '100vh' }}>
      <div style={{
        position: 'sticky',
        top: 0,
        zIndex: 100,
        marginBottom: 8,
        padding: '8px 16px',
        backgroundColor: 'white',
        borderBottom: '1px solid #e8e8e8',
        boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center'
      }}>
        <Title level={4} style={{ margin: 0, fontSize: '16px', fontWeight: '500' }}>扫描结果</Title>
        <Space>
          <Select
            size="small"
            value={sortOrder}
            onChange={handleSortChange}
            style={{ width: 100, fontSize: 12 }}
          >
            <Option value="desc">最新</Option>
            <Option value="asc">最早</Option>
          </Select>
          <Button icon={<ReloadOutlined />} onClick={loadResults} loading={loading}>
            刷新
          </Button>
        </Space>
      </div>

      <div style={{ padding: '0 16px', height: 'calc(100vh - 80px)', display: 'flex', flexDirection: 'column' }}>
        <Card 
          bodyStyle={{ 
            padding: '8px', 
            height: '100%', 
            display: 'flex', 
            flexDirection: 'column' 
          }}
          style={{ height: '100%' }}
        >
          <div style={{ 
            flex: 1, 
            overflowY: 'auto', 
            overflowX: 'hidden',
            scrollbarWidth: 'thin',
            scrollbarColor: '#d9d9d9 #f0f0f0'
          }} className="custom-scrollbar">
            <Table
              dataSource={results}
              columns={columns}
              rowKey="task_id"
              loading={loading}
              locale={{ emptyText: '暂无扫描结果' }}
              size="small"
              pagination={false}
              scroll={{ y: 'calc(100vh - 200px)' }}
              className="compact-table"
            />
          </div>
          <div className="pagination-bar" style={{ 
            flexShrink: 0, 
            padding: '8px 0', 
            borderTop: '1px solid #f0f0f0',
            backgroundColor: '#fafafa'
          }}>
            <div style={{ 
              display: 'flex', 
              justifyContent: 'space-between', 
              alignItems: 'center',
              fontSize: 12
            }}>
              <span>
                共 {results.length} 条记录
              </span>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <span>每页显示:</span>
                <Select
                  size="small"
                  defaultValue={20}
                  style={{ width: 80 }}
                  onChange={(value) => {
                    // 这里可以添加分页逻辑
                    console.log('Page size changed to:', value);
                  }}
                >
                  <Option value={10}>10</Option>
                  <Option value={20}>20</Option>
                  <Option value={50}>50</Option>
                  <Option value={100}>100</Option>
                </Select>
                <span>条</span>
              </div>
            </div>
          </div>
        </Card>

      {/* Result Details Drawer */}
      <Drawer
        title={
          <div style={{ textAlign: 'left' }}>
            <Text strong style={{ fontSize: 16 }}>{selectedResult?.task_name}</Text>
          </div>
        }
        placement="right"
        width={720}
        onClose={() => setDrawerVisible(false)}
        open={drawerVisible}
        bodyStyle={{ padding: '16px' }}
      >
        {selectedResult && (
          <Space direction="vertical" style={{ width: '100%' }} size="large">
            {/* Summary Statistics */}
            <Row gutter={16}>
              <Col span={8}>
                <Statistic
                  title="总请求数"
                  value={selectedResult.total_requests}
                  prefix={<CheckCircleOutlined />}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="发现漏洞"
                  value={selectedResult.found_vulns}
                  valueStyle={{ color: selectedResult.found_vulns > 0 ? '#ff4d4f' : '#666' }}
                  prefix={<BugOutlined />}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="成功率"
                  value={selectedResult.success_rate}
                  precision={1}
                  suffix="%"
                />
              </Col>
            </Row>

            {/* Task Info */}
            <Card 
              title={
                <div style={{ textAlign: 'left' }}>
                  <Text strong style={{ fontSize: 14 }}>任务信息</Text>
                </div>
              } 
              size="small"
              bodyStyle={{ padding: '12px' }}
            >
              <Descriptions column={1} size="small">
                <Descriptions.Item label="状态">
                  <Tag color={selectedResult.status === 'completed' ? 'success' : 'default'}>
                    {selectedResult.status === 'completed' ? '已完成' : selectedResult.status}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="开始时间">
                  {new Date(selectedResult.start_time).toLocaleString('zh-CN')}
                </Descriptions.Item>
                <Descriptions.Item label="结束时间">
                  {new Date(selectedResult.end_time).toLocaleString('zh-CN')}
                </Descriptions.Item>
                <Descriptions.Item label="耗时">
                  <ClockCircleOutlined /> {formatDuration(selectedResult.duration)}
                </Descriptions.Item>
              </Descriptions>
            </Card>

            {/* Targets */}
            <Card 
              title={
                <div style={{ textAlign: 'left' }}>
                  <Text strong style={{ fontSize: 14 }}>扫描目标</Text>
                </div>
              } 
              size="small"
              bodyStyle={{ padding: '12px' }}
            >
              <Space wrap>
                {selectedResult.targets.map((target, index) => (
                  <Tag key={index}>{target}</Tag>
                ))}
              </Space>
            </Card>

            {/* Vulnerabilities */}
            <Card 
              title={
                <div style={{ textAlign: 'left' }}>
                  <Text strong style={{ fontSize: 14 }}>
                    发现的漏洞 ({selectedResult.vulnerabilities.length})
                  </Text>
                </div>
              } 
              size="small"
              bodyStyle={{ padding: '12px' }}
            >
              {selectedResult.vulnerabilities.length > 0 ? (
                <Collapse accordion size="small">
                  {selectedResult.vulnerabilities.map((vuln, index) => (
                    <Panel
                      key={index}
                      header={
                        <div style={{ textAlign: 'left' }}>
                          <Space>
                            <Tag 
                              color={getSeverityColor(vuln.info.severity)}
                              style={{ fontSize: 10, padding: '1px 4px' }}
                            >
                              {vuln.info.severity?.toUpperCase()}
                            </Tag>
                            <Text strong style={{ fontSize: 12 }}>{vuln.info.name}</Text>
                            <Text type="secondary" style={{ fontSize: 11 }}>@ {vuln.host}</Text>
                          </Space>
                        </div>
                      }
                    >
                      <Space direction="vertical" style={{ width: '100%' }} size="middle">
                        <Descriptions column={2} size="small" bordered>
                          <Descriptions.Item label="模板 ID" span={2}>
                            <Text code>{vuln['template-id']}</Text>
                          </Descriptions.Item>
                          <Descriptions.Item label="目标地址" span={2}>
                            <Text>{vuln.host}</Text>
                          </Descriptions.Item>
                          <Descriptions.Item label="匹配位置" span={2}>
                            <Text code>{vuln['matched-at']}</Text>
                          </Descriptions.Item>
                          {vuln.info.description && (
                            <Descriptions.Item label="描述" span={2}>
                              {vuln.info.description}
                            </Descriptions.Item>
                          )}
                          {vuln.info.tags && vuln.info.tags.length > 0 && (
                            <Descriptions.Item label="标签" span={2}>
                              <Space wrap>
                                {vuln.info.tags.map((tag, i) => (
                                  <Tag key={i} color="blue">
                                    {tag}
                                  </Tag>
                                ))}
                              </Space>
                            </Descriptions.Item>
                          )}
                          {vuln['extracted-results'] && vuln['extracted-results'].length > 0 && (
                            <Descriptions.Item label="提取结果" span={2}>
                              <Space wrap>
                                {vuln['extracted-results'].map((result, i) => (
                                  <Tag key={i} color="green">
                                    {result}
                                  </Tag>
                                ))}
                              </Space>
                            </Descriptions.Item>
                          )}
                        </Descriptions>

                        {/* Request/Response Tabs */}
                        {(vuln.request || vuln.response) && (
                          <div style={{ textAlign: 'left' }}>
                            <Tabs
                              defaultActiveKey="response"
                              size="small"
                              items={[
                                ...(vuln.request
                                  ? [
                                      {
                                        key: 'request',
                                        label: <span style={{ fontSize: 12 }}>请求 (Request)</span>,
                                        children: (
                                          <div
                                            style={{
                                              backgroundColor: '#f5f5f5',
                                              color: '#333',
                                              padding: 12,
                                              borderRadius: 4,
                                              fontFamily: 'Monaco, Consolas, monospace',
                                              fontSize: 11,
                                              lineHeight: 1.4,
                                              whiteSpace: 'pre-wrap',
                                              wordBreak: 'break-all',
                                              maxHeight: 300,
                                              overflow: 'auto',
                                              textAlign: 'left',
                                              border: '1px solid #d9d9d9',
                                            }}
                                          >
                                            {vuln.request}
                                          </div>
                                        ),
                                      },
                                    ]
                                  : []),
                                ...(vuln.response
                                  ? [
                                      {
                                        key: 'response',
                                        label: <span style={{ fontSize: 12 }}>响应 (Response)</span>,
                                        children: (
                                          <div
                                            style={{
                                              backgroundColor: '#f5f5f5',
                                              color: '#333',
                                              padding: 12,
                                              borderRadius: 4,
                                              fontFamily: 'Monaco, Consolas, monospace',
                                              fontSize: 11,
                                              lineHeight: 1.4,
                                              whiteSpace: 'pre-wrap',
                                              wordBreak: 'break-all',
                                              maxHeight: 300,
                                              overflow: 'auto',
                                              textAlign: 'left',
                                              border: '1px solid #d9d9d9',
                                            }}
                                          >
                                            {vuln.response}
                                          </div>
                                        ),
                                      },
                                    ]
                                  : []),
                              ]}
                            />
                          </div>
                        )}
                      </Space>
                    </Panel>
                  ))}
                </Collapse>
              ) : (
                <Empty description="未发现漏洞" />
              )}
            </Card>
          </Space>
        )}
      </Drawer>
      </div>
    </div>
  );
};

export default Results;
