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
  DownloadOutlined,
  EditOutlined,
} from '@ant-design/icons';
import { TaskResult, NucleiResult } from '../../types';
import { api } from '../../services/api';
import POCEditor from '../../components/POCEditor';
import { SaveCSVFile } from '../../../wailsjs/go/main/App';
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

  // POC Editor state
  const [pocEditorVisible, setPocEditorVisible] = useState(false);
  const [selectedPOC, setSelectedPOC] = useState<{
    templateId: string;
    templatePath: string;
    target: string;
  } | null>(null);

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

  // 导出单个任务结果为 CSV
  const handleExportCSV = async (result: TaskResult) => {
    try {
      if (!result.vulnerabilities || result.vulnerabilities.length === 0) {
        message.warning('该任务没有发现漏洞，无需导出');
        return;
      }

      // CSV 表头
      const headers = ['模板ID', '模板名称', '严重程度', '目标URL', '提取数据', '扫描时间'];

      // CSV 数据行
      const rows = result.vulnerabilities.map((vuln: NucleiResult) => {
        const extractedData = vuln['extracted-results']?.join('; ') || '';

        return [
          vuln['template-id'] || '',
          vuln.info?.name || '',
          vuln.info?.severity || '',
          vuln['matched-at'] || vuln.host || '',
          extractedData,
          new Date(result.start_time).toLocaleString('zh-CN')
        ];
      });

      // 构建 CSV 内容
      const csvContent = [
        headers.join(','),
        ...rows.map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(','))
      ].join('\n');

      // 使用保存对话框
      const defaultFilename = `扫描结果_${result.task_name}_${new Date().getTime()}.csv`;
      const savedPath = await SaveCSVFile(defaultFilename, csvContent);

      message.success(`导出成功: ${savedPath}`);
    } catch (error: any) {
      console.error('导出失败:', error);
      if (error.message && error.message.includes('用户取消')) {
        // 用户取消，不显示错误
        return;
      }
      message.error(`导出失败: ${error.message || error}`);
    }
  };

  // 导出所有结果为 CSV
  const handleExportAllCSV = async () => {
    try {
      if (results.length === 0) {
        message.warning('暂无扫描结果，无需导出');
        return;
      }

      const allVulns = results.flatMap(result => result.vulnerabilities || []);

      if (allVulns.length === 0) {
        message.warning('所有任务均未发现漏洞，无需导出');
        return;
      }

      // CSV 表头
      const headers = ['任务名称', '模板ID', '模板名称', '严重程度', '目标URL', '提取数据', '扫描时间'];

      // CSV 数据行
      const rows: string[][] = [];
      results.forEach(result => {
        (result.vulnerabilities || []).forEach((vuln: NucleiResult) => {
          const extractedData = vuln['extracted-results']?.join('; ') || '';

          rows.push([
            result.task_name || '',
            vuln['template-id'] || '',
            vuln.info?.name || '',
            vuln.info?.severity || '',
            vuln['matched-at'] || vuln.host || '',
            extractedData,
            new Date(result.start_time).toLocaleString('zh-CN')
          ]);
        });
      });

      // 构建 CSV 内容
      const csvContent = [
        headers.join(','),
        ...rows.map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(','))
      ].join('\n');

      // 使用保存对话框
      const defaultFilename = `所有扫描结果_${new Date().getTime()}.csv`;
      const savedPath = await SaveCSVFile(defaultFilename, csvContent);

      message.success(`导出成功: ${savedPath}，共 ${rows.length} 条漏洞记录`);
    } catch (error: any) {
      console.error('导出失败:', error);
      if (error.message && error.message.includes('用户取消')) {
        // 用户取消，不显示错误
        return;
      }
      message.error(`导出失败: ${error.message || error}`);
    }
  };

  // Handle POC editor open
  const handleEditPOC = (vuln: NucleiResult) => {
    setSelectedPOC({
      templateId: vuln['template-id'],
      templatePath: vuln['template-path'],
      target: vuln.host || vuln['matched-at'] || '',
    });
    setPocEditorVisible(true);
  };

  // Handle POC editor close
  const handleClosePOCEditor = () => {
    setPocEditorVisible(false);
    setSelectedPOC(null);
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
      width: 150,
      render: (_: any, record: TaskResult) => (
        <Space size="small">
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => handleViewDetails(record)}
            style={{ fontSize: 12, padding: '4px 8px' }}
          >
            详情
          </Button>
          <Button
            type="link"
            icon={<DownloadOutlined />}
            onClick={() => handleExportCSV(record)}
            style={{ fontSize: 12, padding: '4px 8px' }}
            disabled={!record.vulnerabilities || record.vulnerabilities.length === 0}
          >
            导出
          </Button>
        </Space>
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
          <Button
            icon={<DownloadOutlined />}
            onClick={handleExportAllCSV}
            disabled={results.length === 0}
          >
            导出所有结果
          </Button>
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
              <Col span={6}>
                <Statistic
                  title="HTTP请求数"
                  value={selectedResult.http_requests || selectedResult.completed_requests}
                  prefix={<CheckCircleOutlined />}
                  valueStyle={{ fontSize: 20 }}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="发现漏洞"
                  value={selectedResult.found_vulns}
                  valueStyle={{ color: selectedResult.found_vulns > 0 ? '#ff4d4f' : '#666', fontSize: 20 }}
                  prefix={<BugOutlined />}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="成功率"
                  value={selectedResult.success_rate}
                  precision={1}
                  suffix="%"
                  valueStyle={{ fontSize: 20 }}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="总耗时"
                  value={formatDuration(selectedResult.duration)}
                  valueStyle={{ fontSize: 20 }}
                  prefix={<ClockCircleOutlined />}
                />
              </Col>
            </Row>

            {/* 模板扫描统计 */}
            <Card
              title={
                <div style={{ textAlign: 'left' }}>
                  <Text strong style={{ fontSize: 14 }}>模板扫描统计</Text>
                </div>
              }
              size="small"
              bodyStyle={{ padding: '12px' }}
            >
              <Row gutter={16}>
                <Col span={6}>
                  <Statistic
                    title="选择的POC"
                    value={selectedResult.template_count}
                    valueStyle={{ fontSize: 16 }}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="实际扫描"
                    value={selectedResult.scanned_templates || 0}
                    valueStyle={{ fontSize: 16, color: '#52c41a' }}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="被过滤"
                    value={selectedResult.filtered_templates || 0}
                    valueStyle={{ fontSize: 16, color: '#faad14' }}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="被跳过"
                    value={selectedResult.skipped_templates || 0}
                    valueStyle={{ fontSize: 16, color: '#999' }}
                  />
                </Col>
              </Row>

              {/* 友好提示信息 */}
              {selectedResult.filtered_templates && selectedResult.filtered_templates > 0 && (
                <div style={{ marginTop: 12, padding: '8px 12px', backgroundColor: '#fffbe6', borderRadius: 4, border: '1px solid #ffe58f' }}>
                  <Text type="warning" style={{ fontSize: 12 }}>
                    ⚠️ <strong>{selectedResult.filtered_templates}</strong> 个POC被Nuclei过滤（通常是因为需要code执行、headless浏览器等特殊环境）
                  </Text>
                  {selectedResult.filtered_template_ids && selectedResult.filtered_template_ids.length > 0 && (
                    <div style={{ marginTop: 6 }}>
                      <Text type="secondary" style={{ fontSize: 11 }}>
                        被过滤的POC: {selectedResult.filtered_template_ids.slice(0, 5).join(', ')}
                        {selectedResult.filtered_template_ids.length > 5 && ` 等${selectedResult.filtered_template_ids.length}个`}
                      </Text>
                    </div>
                  )}
                </div>
              )}

              {selectedResult.skipped_templates && selectedResult.skipped_templates > 0 && (
                <div style={{ marginTop: 8, padding: '8px 12px', backgroundColor: '#f5f5f5', borderRadius: 4, border: '1px solid #d9d9d9' }}>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    ℹ️ <strong>{selectedResult.skipped_templates}</strong> 个POC被跳过（目标不符合扫描条件，如协议不匹配、端口不开放等）
                  </Text>
                  {selectedResult.skipped_template_ids && selectedResult.skipped_template_ids.length > 0 && (
                    <div style={{ marginTop: 6 }}>
                      <Text type="secondary" style={{ fontSize: 11 }}>
                        被跳过的POC: {selectedResult.skipped_template_ids.slice(0, 5).join(', ')}
                        {selectedResult.skipped_template_ids.length > 5 && ` 等${selectedResult.skipped_template_ids.length}个`}
                      </Text>
                    </div>
                  )}
                </div>
              )}

              {selectedResult.failed_templates && selectedResult.failed_templates > 0 && (
                <div style={{ marginTop: 8, padding: '8px 12px', backgroundColor: '#fff2f0', borderRadius: 4, border: '1px solid #ffccc7' }}>
                  <Text type="danger" style={{ fontSize: 12 }}>
                    ❌ <strong>{selectedResult.failed_templates}</strong> 个POC扫描失败（可能是网络错误、超时等原因）
                  </Text>
                  {selectedResult.failed_template_ids && selectedResult.failed_template_ids.length > 0 && (
                    <div style={{ marginTop: 6 }}>
                      <Text type="secondary" style={{ fontSize: 11 }}>
                        失败的POC: {selectedResult.failed_template_ids.slice(0, 5).join(', ')}
                        {selectedResult.failed_template_ids.length > 5 && ` 等${selectedResult.failed_template_ids.length}个`}
                      </Text>
                    </div>
                  )}
                </div>
              )}
            </Card>

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
                        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
                          <Button
                            type="primary"
                            size="small"
                            icon={<EditOutlined />}
                            onClick={() => handleEditPOC(vuln)}
                          >
                            编辑 POC
                          </Button>
                        </div>
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

      {/* POC Editor Modal */}
      {selectedPOC && (
        <POCEditor
          visible={pocEditorVisible}
          onClose={handleClosePOCEditor}
          templateId={selectedPOC.templateId}
          templatePath={selectedPOC.templatePath}
          defaultTarget={selectedPOC.target}
        />
      )}
      </div>
    </div>
  );
};

export default Results;
