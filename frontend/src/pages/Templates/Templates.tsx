import { useState, useEffect } from 'react';
import {
  Button,
  Table,
  Space,
  Tag,
  Input,
  Select,
  message,
  Modal,
  Popconfirm,
  Card,
  Statistic,
  Row,
  Col,
  Progress,
  Alert,
  Pagination,
} from 'antd';
import {
  FolderOpenOutlined,
  ReloadOutlined,
  DeleteOutlined,
  ScanOutlined,
  SearchOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Template } from '../../types';
import * as api from '../../services/api';
// import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import './Templates.css';

const { Search } = Input;
const { Option } = Select;

const Templates = () => {
  const navigate = useNavigate();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedRows, setSelectedRows] = useState<Template[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [severityFilter, setSeverityFilter] = useState('');
  const [validating, setValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<any>(null);
  const [showValidationModal, setShowValidationModal] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(50); // 默认50条/页
  const [showTaskNameModal, setShowTaskNameModal] = useState(false);
  const [taskName, setTaskName] = useState('');
  
  // 导入进度相关状态
  const [importCount, setImportCount] = useState(0);

  // 过滤和分页逻辑
  const filteredTemplates = templates.filter(template => {
    const matchesSearch = !searchKeyword || 
      template.name?.toLowerCase().includes(searchKeyword.toLowerCase()) ||
      template.tags?.toLowerCase().includes(searchKeyword.toLowerCase());
    const matchesSeverity = !severityFilter || template.severity === severityFilter;
    return matchesSearch && matchesSeverity;
  });

  const paginatedTemplates = filteredTemplates.slice(
    (currentPage - 1) * pageSize,
    currentPage * pageSize
  );

  const totalTemplates = filteredTemplates.length;

  useEffect(() => {
    loadTemplates();
    // 检查POC模板可用性，如果没有则提示用户
    checkPOCAvailability();

    // 监听模板导入进度事件
    const unsubscribe = api.onTemplateImportProgress((data: any) => {
      if (data && data.data) {
        // 如果导入完成，显示导入数量并刷新模板列表
        if (data.data.status === '导入完成!') {
          setImportCount(data.data.successful || 0);
          loadTemplates();
          // 只在手动导入时显示成功提示，避免自动导入时的重复提示
          // message.success(`成功导入 ${data.data.successful || 0} 个模板`);
        }
      }
    });

    return () => {
      if (unsubscribe) {
        unsubscribe();
      }
    };
  }, []);

  // 当模板加载完成后，恢复上次的选择状态
  useEffect(() => {
    if (templates.length === 0) return; // 等待模板加载完成
    
    const savedSelection = sessionStorage.getItem('selectedTemplates');
    if (savedSelection) {
      try {
        const parsed = JSON.parse(savedSelection);
        
        // 兼容两种数据格式：
        // 1. 来自扫描任务页面的简单数组格式: ["template1", "template2"]
        // 2. 模板管理页面的对象格式: {rows: [], keys: []}
        if (Array.isArray(parsed)) {
          // 如果是数组格式（来自扫描任务页面），需要根据模板路径找到对应的模板对象
          const matchedTemplates = templates.filter(template => 
            parsed.includes(template.file_path)
          );
          setSelectedRows(matchedTemplates);
          setSelectedRowKeys(matchedTemplates.map(t => t.id));
        } else if (parsed && typeof parsed === 'object') {
          // 如果是对象格式（模板管理页面自己的格式）
          setSelectedRows(parsed.rows || []);
          setSelectedRowKeys(parsed.keys || []);
        }
      } catch (e) {
        console.error('Failed to restore selection:', e);
        // 清除无效的sessionStorage数据
        sessionStorage.removeItem('selectedTemplates');
      }
    }
  }, [templates]); // 依赖templates，确保模板加载完成后再恢复选择状态

  // 保存选择状态到sessionStorage
  useEffect(() => {
    if (selectedRows.length > 0) {
      sessionStorage.setItem('selectedTemplates', JSON.stringify({
        rows: selectedRows,
        keys: selectedRowKeys,
      }));
    }
  }, [selectedRows, selectedRowKeys]);

  // 当搜索或过滤条件改变时，重置到第一页
  useEffect(() => {
    setCurrentPage(1);
  }, [searchKeyword, severityFilter]);

  // 检查是否有POC模板，如果没有则提示用户
  const checkPOCAvailability = async () => {
    try {
      const templates = await api.getAllTemplates();
      if (!templates || templates.length === 0) {
        Modal.info({
          title: '欢迎使用 wepoc',
          content: (
            <div>
              <p>检测到您还没有导入任何 POC 模板。</p>
              <p>请前往 <strong>设置</strong> 页面配置 Nuclei 路径和 POC 目录，然后导入模板。</p>
              <p>或者点击下方的"导入模板"按钮手动导入。</p>
            </div>
          ),
          okText: '前往设置',
          onOk: () => navigate('/settings'),
        });
      }
    } catch (error) {
      console.error('检查POC模板失败:', error);
    }
  };

  const loadTemplates = async () => {
    setLoading(true);
    try {
      const data = await api.getAllTemplates();
      setTemplates(data || []);
    } catch (error: any) {
      message.error(`加载模板失败: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleImport = async () => {
    try {
      const dir = await api.selectDirectory();
      if (!dir) return;

      setValidating(true);
      const result = await api.preValidateTemplates(dir);

      // Store validation result and show modal
      setValidationResult(result);
      setShowValidationModal(true);
    } catch (error: any) {
      message.error(`验证失败: ${error.message}`);
    } finally {
      setValidating(false);
    }
  };

  const handleConfirmImport = async () => {
    if (!validationResult || !validationResult.validTemplates) {
      message.error('没有可导入的模板');
      return;
    }

    try {
      setLoading(true);
      const result = await api.confirmAndImportTemplates(validationResult.validTemplates);

      // 设置导入数量并显示简单提示
      setImportCount(result.validated);
      message.success(`成功导入 ${result.validated} 个模板`);

      setShowValidationModal(false);
      await loadTemplates();
    } catch (error: any) {
      message.error(`导入失败: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async () => {
    setLoading(true);
    try {
      const data = await api.searchTemplates(searchKeyword, severityFilter);
      setTemplates(data || []);
    } catch (error: any) {
      message.error(`Search failed: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };


  const handleClearAll = async () => {
    try {
      await api.clearAllTemplates();
      message.success('All templates cleared');
      await loadTemplates();
      setSelectedRows([]);
    } catch (error: any) {
      message.error(`Clear failed: ${error.message}`);
    }
  };

  const handleStartScan = () => {
    if (selectedRows.length === 0) {
      message.warning('Please select at least one template');
      return;
    }

    // Show task name input modal
    setShowTaskNameModal(true);
  };

  const handleConfirmTaskName = () => {
    // Generate task name if not provided
    const finalTaskName = taskName.trim() || `POC扫描任务-${new Date().toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    }).replace(/\//g, '').replace(/:/g, '').replace(' ', '-')}`;
    
    // Navigate to scan tasks page with selected templates and task name
    const templateIds = selectedRows.map(t => t.file_path);
    sessionStorage.setItem('selectedTemplates', JSON.stringify(templateIds));
    sessionStorage.setItem('taskName', finalTaskName);
    sessionStorage.setItem('autoCreateTask', 'true'); // 标记需要自动创建任务
    setShowTaskNameModal(false);
    setTaskName('');
    navigate('/tasks');
  };

  const handleDownload = (record: Template) => {
    message.info(`下载模板: ${record.template_id}`);
    // TODO: 实现下载功能
  };

  const handleDelete = async (record: Template) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除模板 "${record.template_id}" 吗？此操作将同时删除文件系统中的模板文件。`,
      onOk: async () => {
        try {
          await api.deleteTemplate(record.template_id);
          message.success('模板删除成功');
          await loadTemplates();
        } catch (error: any) {
          message.error(`删除失败: ${error.message}`);
        }
      },
    });
  };

  const getSeverityColor = (severity: string) => {
    const severityLower = severity?.toLowerCase() || '';
    const colorMap: Record<string, string> = {
      critical: '#ff4d4f',
      high: '#ff7a45',
      medium: '#ffa940',
      low: '#ffc53d',
      info: '#69c0ff',
    };
    return colorMap[severityLower] || '#d9d9d9';
  };

  const columns = [
    {
      title: 'Template ID',
      dataIndex: 'template_id',
      key: 'template_id',
      width: 200,
      ellipsis: true,
    },
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: 'Severity',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: string) => (
        <Tag color={getSeverityColor(severity)}>{severity?.toUpperCase() || 'UNKNOWN'}</Tag>
      ),
    },
    {
      title: 'Author',
      dataIndex: 'author',
      key: 'author',
      width: 150,
      ellipsis: true,
    },
    {
      title: 'Tags',
      dataIndex: 'tags',
      key: 'tags',
      width: 200,
      ellipsis: true,
    },
    {
      title: 'Action',
      key: 'action',
      width: 100,
      render: (_: any, record: Template) => (
        <Popconfirm
          title="Delete template?"
          description="Are you sure to delete this template?"
          onConfirm={() => handleDelete(record)}
          okText="Yes"
          cancelText="No"
        >
          <Button type="link" danger size="small" icon={<DeleteOutlined />}>
            Delete
          </Button>
        </Popconfirm>
      ),
    },
  ];

  const statistics = {
    total: totalTemplates,
    critical: filteredTemplates.filter(t => t.severity?.toLowerCase() === 'critical').length,
    high: filteredTemplates.filter(t => t.severity?.toLowerCase() === 'high').length,
    medium: filteredTemplates.filter(t => t.severity?.toLowerCase() === 'medium').length,
    low: filteredTemplates.filter(t => t.severity?.toLowerCase() === 'low').length,
  };

  return (
    <div className="template-management">
      {/* 顶部操作栏 */}
      <div className="top-toolbar">
        <div className="toolbar-left">
          <Button
            type="primary"
            icon={<FolderOpenOutlined />}
            onClick={handleImport}
            loading={validating}
            size="small"
          >
            导入{importCount > 0 && ` (${importCount})`}
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={loadTemplates}
            loading={loading}
            size="small"
          >
            刷新
          </Button>
          <div className="toolbar-stats">
            <span>已加载 <strong>{statistics.total}</strong> 个</span>
            <span>已选择 <strong>{selectedRows.length}</strong> 个</span>
          </div>
        </div>
        
        <div className="toolbar-right">
          <Search
            placeholder="搜索模板..."
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
            onSearch={handleSearch}
            size="small"
          />
          <Select
            placeholder="严重度"
            value={severityFilter || undefined}
            onChange={setSeverityFilter}
            allowClear
            size="small"
          >
            <Option value="critical">Critical</Option>
            <Option value="high">High</Option>
            <Option value="medium">Medium</Option>
            <Option value="low">Low</Option>
            <Option value="info">Info</Option>
          </Select>
          <Select
            placeholder="状态"
            size="small"
          >
            <Option value="validated">已验证</Option>
            <Option value="invalid">验证失败</Option>
          </Select>
        </div>
      </div>

      {/* 导入进度显示 */}

      {/* 模板列表 */}
      <div className="template-list-container">
        <Table
          rowSelection={{
            type: 'checkbox',
            selectedRowKeys: selectedRowKeys,
            onChange: (selectedKeys, selectedRows) => {
              setSelectedRowKeys(selectedKeys);
              setSelectedRows(selectedRows);
            },
          }}
          columns={[
            {
              title: 'ID',
              dataIndex: 'template_id',
              key: 'template_id',
              width: 180,
              render: (id: string) => (
                <span style={{ color: '#1890ff', cursor: 'pointer', fontSize: '12px' }}>
                  {id}
                </span>
              ),
            },
            {
              title: '名称',
              dataIndex: 'name',
              key: 'name',
              ellipsis: true,
              width: 250,
            },
            {
              title: '标签',
              dataIndex: 'tags',
              key: 'tags',
              width: 200,
              render: (tags: string) => {
                if (!tags) return '-';
                const tagList = tags.split(',').slice(0, 2);
                return (
                  <div style={{ display: 'flex', gap: '2px', flexWrap: 'wrap' }}>
                    {tagList.map((tag, idx) => (
                      <Tag key={idx} className="template-tag">
                        {tag.trim()}
                      </Tag>
                    ))}
                    {tags.split(',').length > 2 && (
                      <span style={{ fontSize: '10px', color: '#999' }}>...</span>
                    )}
                  </div>
                );
              },
            },
            {
              title: '级别',
              dataIndex: 'severity',
              key: 'severity',
              width: 80,
              render: (severity: string) => (
                <Tag 
                  color={getSeverityColor(severity)} 
                  className="severity-tag"
                >
                  {severity?.toUpperCase() || 'UNKNOWN'}
                </Tag>
              ),
            },
            {
              title: '作者',
              dataIndex: 'author',
              key: 'author',
              width: 120,
              ellipsis: true,
            },
            {
              title: '',
              key: 'actions',
              width: 80,
              render: (_, record) => (
                <div style={{ display: 'flex', gap: '8px' }}>
                  <Button
                    type="text"
                    size="small"
                    icon={<ReloadOutlined />}
                    style={{ color: '#1890ff' }}
                    onClick={() => handleDownload(record)}
                  />
                  <Button
                    type="text"
                    size="small"
                    icon={<DeleteOutlined />}
                    style={{ color: '#ff4d4f' }}
                    onClick={() => handleDelete(record)}
                  />
                </div>
              ),
            },
          ]}
          dataSource={paginatedTemplates}
          rowKey="id"
          loading={loading}
          pagination={false}
          size="small"
        />
      </div>

      {/* 底部操作栏 */}
      <div className="bottom-toolbar">
        <div className="bottom-left">
          <span className="selection-info">
            已选择 <strong>{selectedRows.length}</strong>/{totalTemplates} 个模板
          </span>
        </div>
        
        <div className="bottom-center">
          <Pagination
            current={currentPage}
            pageSize={pageSize}
            total={totalTemplates}
            showSizeChanger
            showTotal={(total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`}
            onChange={(page, size) => {
              setCurrentPage(page);
              setPageSize(size || 50);
            }}
            onShowSizeChange={(current, size) => {
              setCurrentPage(1);
              setPageSize(size);
            }}
            size="small"
            pageSizeOptions={['50', '100', '200', '300', '500']}
          />
        </div>
        
        <div className="bottom-right">
          <Button
            type="primary"
            icon={<ScanOutlined />}
            onClick={handleStartScan}
            disabled={selectedRows.length === 0}
            style={{ height: '32px' }}
          >
            开始扫描 ({selectedRows.length})
          </Button>
        </div>
      </div>

      {/* Validation Modal */}
      <Modal
        title="模板验证结果"
        open={showValidationModal}
        onCancel={() => setShowValidationModal(false)}
        width={700}
        footer={[
          <Button key="cancel" onClick={() => setShowValidationModal(false)}>
            取消
          </Button>,
          <Button
            key="confirm"
            type="primary"
            onClick={handleConfirmImport}
            loading={loading}
            disabled={!validationResult || validationResult.validated === 0}
          >
            确认导入 ({validationResult?.validated || 0} 个)
          </Button>,
        ]}
      >
        {validationResult && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Row gutter={16}>
                <Col span={6}>
                  <Statistic title="发现模板" value={validationResult.totalFound} />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="验证成功"
                    value={validationResult.validated}
                    valueStyle={{ color: '#52c41a' }}
                    prefix={<CheckCircleOutlined />}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="验证失败"
                    value={validationResult.failed}
                    valueStyle={{ color: '#ff4d4f' }}
                    prefix={<ExclamationCircleOutlined />}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title="已存在"
                    value={validationResult.alreadyExists}
                    valueStyle={{ color: '#faad14' }}
                  />
                </Col>
              </Row>
            </div>

            {validationResult.validated > 0 && (
              <div style={{ marginBottom: 16 }}>
                <Progress
                  percent={Math.round((validationResult.validated / validationResult.totalFound) * 100)}
                  status={validationResult.validated === validationResult.totalFound ? 'success' : 'active'}
                  format={(percent?: number) => `${percent || 0}% 验证通过`}
                />
              </div>
            )}

            {validationResult.errors.length > 0 && (
              <div>
                <h4>验证错误详情：</h4>
                <div style={{ maxHeight: '200px', overflow: 'auto', backgroundColor: '#f5f5f5', padding: 8, borderRadius: 4 }}>
                  {validationResult.errors.slice(0, 10).map((err: string, idx: number) => (
                    <div key={idx} style={{ fontSize: '12px', color: '#ff4d4f', marginBottom: 4 }}>
                      {err}
                    </div>
                  ))}
                  {validationResult.errors.length > 10 && (
                    <div style={{ fontSize: '12px', color: '#999', textAlign: 'center' }}>
                      ... 还有 {validationResult.errors.length - 10} 个错误
                    </div>
                  )}
                </div>
              </div>
            )}

            <Alert
              message="验证完成"
              description={`发现 ${validationResult.totalFound} 个模板，其中 ${validationResult.validated} 个验证通过，可以安全导入。`}
              type={validationResult.validated > 0 ? 'success' : 'warning'}
              showIcon
              style={{ marginTop: 16 }}
            />
          </div>
        )}
      </Modal>

      {/* Task Name Modal */}
      <Modal
        title="输入任务名称"
        open={showTaskNameModal}
        onOk={handleConfirmTaskName}
        onCancel={() => {
          setShowTaskNameModal(false);
          setTaskName('');
        }}
        okText="确认"
        cancelText="取消"
        width={400}
      >
        <div style={{ marginBottom: 16 }}>
          <p>为扫描任务输入一个名称 (可选):</p>
          <Input
            placeholder="输入任务名称"
            value={taskName}
            onChange={(e) => setTaskName(e.target.value)}
            onPressEnter={handleConfirmTaskName}
          />
          <p style={{ marginTop: 8, color: '#666', fontSize: 12 }}>
            如果不输入名称，系统将自动生成任务名称
          </p>
        </div>
      </Modal>
    </div>
  );
};

export default Templates;
