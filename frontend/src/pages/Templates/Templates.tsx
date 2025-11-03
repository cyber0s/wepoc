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
import { api } from '../../services/api';
import './Templates.css';

const { Search } = Input;
const { Option } = Select;

const Templates = () => {
  const navigate = useNavigate();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [totalTemplates, setTotalTemplates] = useState(0);
  const [loading, setLoading] = useState(false);
  const [selectedRows, setSelectedRows] = useState<Template[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [severityFilter, setSeverityFilter] = useState('');
  const [validating, setValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<any>(null);
  const [showValidationModal, setShowValidationModal] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(100);  // 默认每页100条
  const [showTaskNameModal, setShowTaskNameModal] = useState(false);
  const [taskName, setTaskName] = useState('');
  const [importCount, setImportCount] = useState(0);

  // 过滤和分页逻辑
  const filteredTemplates = templates.filter(template => {
    const matchesSearch = !searchKeyword ||
      template.name?.toLowerCase().includes(searchKeyword.toLowerCase()) ||
      template.tags?.toLowerCase().includes(searchKeyword.toLowerCase());
    const matchesSeverity = !severityFilter || template.severity === severityFilter;
    return matchesSearch && matchesSeverity;
  });

  // 分页切片
  const paginatedTemplates = filteredTemplates.slice(
    (currentPage - 1) * pageSize,
    currentPage * pageSize
  );

  useEffect(() => {
    loadTemplates();
  }, []);

  // 当搜索或过滤条件变化时，重置到第一页
  useEffect(() => {
    setCurrentPage(1);
  }, [searchKeyword, severityFilter]);

  const loadTemplates = async () => {
    setLoading(true);
    try {
      const result = await api.getAllTemplates();
      setTemplates(result || []);
      setTotalTemplates(result?.length || 0);
    } catch (error) {
      console.error('Failed to load templates:', error);
      message.error('加载模板失败');
    } finally {
      setLoading(false);
    }
  };

  const handleImport = async () => {
    try {
      const directory = await api.selectDirectory();
      if (directory) {
        setValidating(true);
        const result = await api.preValidateTemplates(directory);
        setValidationResult(result);
        setShowValidationModal(true);
      }
    } catch (error) {
      console.error('Failed to import templates:', error);
      message.error('导入模板失败');
    } finally {
      setValidating(false);
    }
  };

  const handleConfirmImport = async () => {
    try {
      if (validationResult && validationResult.templates) {
        const result = await api.confirmAndImportTemplates(validationResult.templates);
        message.success(`成功导入 ${result.successful} 个模板`);
        setShowValidationModal(false);
        loadTemplates(); // 重新加载模板列表
      }
    } catch (error) {
      console.error('Failed to confirm import:', error);
      message.error('确认导入失败');
    }
  };

  const handleSearch = async () => {
    // 搜索逻辑已在filteredTemplates中实现
  };

  const handleClearAll = async () => {
    try {
      await api.clearAllTemplates();
      message.success('已清空所有模板');
      loadTemplates(); // 重新加载模板列表
    } catch (error) {
      console.error('Failed to clear all templates:', error);
      message.error('清空模板失败');
    }
  };

  const handleStartScan = () => {
    if (selectedRows.length === 0) {
      message.warning('请先选择要扫描的模板');
      return;
    }
    setShowTaskNameModal(true);
  };

  const handleConfirmTaskName = () => {
    setShowTaskNameModal(false);
    navigate('/tasks', { 
      state: { 
        selectedTemplates: selectedRows,
        taskName: taskName || `扫描任务_${new Date().toLocaleString()}`
      } 
    });
  };

  const handleDownload = (record: Template) => {
    console.log('Download template:', record);
  };

  const handleDelete = async (record: Template) => {
    try {
      message.info('删除功能暂时不可用');
    } catch (error) {
      console.error('Failed to delete template:', error);
      message.error('删除模板失败');
    }
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

  const statistics = {
    total: totalTemplates,
    critical: filteredTemplates.filter(t => t.severity?.toLowerCase() === 'critical').length,
    high: filteredTemplates.filter(t => t.severity?.toLowerCase() === 'high').length,
    medium: filteredTemplates.filter(t => t.severity?.toLowerCase() === 'medium').length,
    low: filteredTemplates.filter(t => t.severity?.toLowerCase() === 'low').length,
  };

  return (
    <div className="template-management">
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
        </div>
      </div>

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
              width: 200,
              ellipsis: true,
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
              width: 300,
            },
            {
              title: '级别',
              dataIndex: 'severity',
              key: 'severity',
              width: 100,
              align: 'center',
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
              width: 150,
              ellipsis: true,
            },
            {
              title: '',
              key: 'actions',
              width: 100,
              align: 'center',
              render: (_, record) => (
                <div style={{ display: 'flex', gap: '8px', justifyContent: 'center' }}>
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
          scroll={{ x: 950 }}
        />
      </div>

      <div className="bottom-toolbar">
        <div className="bottom-left">
          <span className="selection-info">
            已选择 <strong>{selectedRows.length}</strong>/{filteredTemplates.length} 个模板
            {searchKeyword || severityFilter ? (
              <span style={{ marginLeft: 8, color: '#1890ff', fontSize: 12 }}>
                (已过滤，共 {templates.length} 个模板)
              </span>
            ) : null}
          </span>
        </div>

        <div className="bottom-center">
          <Pagination
            current={currentPage}
            pageSize={pageSize}
            total={filteredTemplates.length}
            showSizeChanger
            showQuickJumper
            showTotal={(total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`}
            onChange={(page, size) => {
              setCurrentPage(page);
              if (size !== pageSize) {
                setPageSize(size);
                setCurrentPage(1); // 改变页大小时重置到第一页
              }
            }}
            onShowSizeChange={(current, size) => {
              setPageSize(size);
              setCurrentPage(1);
            }}
            size="small"
            pageSizeOptions={['50', '100', '300', '500', '1000']}
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
