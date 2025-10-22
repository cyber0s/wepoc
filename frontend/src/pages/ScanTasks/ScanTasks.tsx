import { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Typography,
  Button,
  Input,
  Space,
  Row,
  Col,
  List,
  Tag,
  message,
  Modal,
  Divider,
  Badge,
  Table,
  Select,
} from 'antd';
import {
  PlusOutlined,
  PlayCircleOutlined,
  StopOutlined,
  DeleteOutlined,
  ReloadOutlined,
  BugOutlined,
  FileAddOutlined,
} from '@ant-design/icons';
import { TaskConfig, ScanEvent, ScanProgress, ScanLogEntry, Template } from '../../types';
import * as api from '../../services/api';
import ScanProgressComponent from '../../components/ScanProgressComponent';
import ScanLogs from '../../components/ScanLogs';
import './ScanTasks.css';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;

const ScanTasks = () => {
  const [tasks, setTasks] = useState<TaskConfig[]>([]);
  const [selectedTask, setSelectedTask] = useState<TaskConfig | null>(null);
  const [selectedTemplates, setSelectedTemplates] = useState<string[]>([]);
  const [targets, setTargets] = useState('');
  const [taskName, setTaskName] = useState('');
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc');

  // Template selection modal states
  const [templateModalVisible, setTemplateModalVisible] = useState(false);
  const [allTemplates, setAllTemplates] = useState<Template[]>([]);
  const [filteredTemplates, setFilteredTemplates] = useState<Template[]>([]);
  const [templateSearchKeyword, setTemplateSearchKeyword] = useState('');
  const [selectedTemplateRows, setSelectedTemplateRows] = useState<Template[]>([]);
  const [selectedTemplateKeys, setSelectedTemplateKeys] = useState<React.Key[]>([]);

  // Real-time progress and logs for selected task
  const [taskProgress, setTaskProgress] = useState<Record<number, ScanProgress>>({});
  const [taskLogs, setTaskLogs] = useState<Record<number, ScanLogEntry[]>>({});

  // Load tasks on mount
  useEffect(() => {
    loadTasks();
    loadSelectedTemplates();

    // Check if there's a task name from template page (indicates user wants to create task)
    const savedTaskName = sessionStorage.getItem('taskName');
    if (savedTaskName) {
      setTaskName(savedTaskName);
      // Auto-open create modal if coming from template page
      setCreateModalVisible(true);
      // Clear the taskName from session storage
      sessionStorage.removeItem('taskName');
    }
  }, []);

  // Listen to scan events
  useEffect(() => {
    const unsubscribe = api.onScanEvent((event: ScanEvent) => {
      console.log('Received scan event:', event); // 添加日志以便调试
      
      if (event.event_type === 'progress') {
        console.log('Progress event received:', event.data); // 添加调试日志
        // Update progress for the task
        setTaskProgress(prev => ({
          ...prev,
          [event.task_id]: event.data,
        }));

        // Also update task status in the list
        setTasks(prevTasks =>
          prevTasks.map(task =>
            task.id === event.task_id
              ? {
                  ...task,
                  status: event.data.status,
                  completed_requests: event.data.completed_requests,
                  found_vulns: event.data.found_vulns,
                }
              : task
          )
        );
        
        // Force a re-render to ensure UI updates
        forceUpdate();
      } else if (event.event_type === 'vuln_found') {
        // Vulnerability notification is now handled globally by GlobalVulnNotification component
        // No need to handle it here
      } else if (event.event_type === 'completed') {
        // 避免重复显示完成消息
        // message.success(`任务扫描完成！`);
        loadTasks(); // Reload tasks to get final state
      } else if (event.event_type === 'error') {
        message.error(`任务失败: ${event.data}`);
        loadTasks();
      }
      // Note: We no longer handle 'log' events to avoid UI lag
    });

    return () => {
      // Cleanup if needed
    };
  }, []);

  const loadTasks = async () => {
    try {
      const allTasks = await api.getAllScanTasks();
      const sortedTasks = sortTasksByTime(allTasks || [], sortOrder);
      setTasks(sortedTasks);
    } catch (error) {
      console.error('Failed to load tasks:', error);
      message.error('加载任务失败');
    }
  };

  // Sort tasks by creation time
  const sortTasksByTime = (taskList: TaskConfig[], order: 'desc' | 'asc') => {
    return [...taskList].sort((a, b) => {
      const timeA = new Date(a.created_at).getTime();
      const timeB = new Date(b.created_at).getTime();
      return order === 'desc' ? timeB - timeA : timeA - timeB;
    });
  };

  // Handle sort order change
  const handleSortChange = (order: 'desc' | 'asc') => {
    setSortOrder(order);
    const sortedTasks = sortTasksByTime(tasks, order);
    setTasks(sortedTasks);
  };

  const loadSelectedTemplates = () => {
    const templates = sessionStorage.getItem('selectedTemplates');
    if (templates) {
      setSelectedTemplates(JSON.parse(templates));
    }
  };

  const handleCreateTask = async () => {
    if (selectedTemplates.length === 0) {
      message.warning('请先选择 POC 模板');
      return;
    }

    if (!targets.trim()) {
      message.warning('请输入目标地址');
      return;
    }

    const targetList = targets.split('\n').filter(t => t.trim());
    if (targetList.length === 0) {
      message.warning('请输入有效的目标地址');
      return;
    }

    setCreating(true);
    try {
      console.log('Creating task with:', {
        templates: selectedTemplates,
        targets: targetList,
        taskName: taskName || undefined
      });
      
      const task = await api.createScanTask(selectedTemplates, targetList, taskName || undefined);
      
      if (task && task.id) {
        message.success('任务创建成功');
        setCreateModalVisible(false);
        setTargets('');
        setTaskName('');
        setSelectedTemplates([]);
        sessionStorage.removeItem('selectedTemplates');
        await loadTasks();
      } else {
        throw new Error('任务创建失败：返回数据异常');
      }
    } catch (error: any) {
      console.error('Failed to create task:', error);
      const errorMessage = error?.message || error?.toString() || '未知错误';
      message.error(`创建任务失败: ${errorMessage}`);
    } finally {
      setCreating(false);
    }
  };

  const handleStartTask = async (taskId: number) => {
    setLoading(true);
    try {
      await api.startScanTask(taskId);
      message.success('任务已启动');
      await loadTasks();
    } catch (error: any) {
      console.error('Failed to start task:', error);
      message.error(`启动任务失败: ${error.message || error}`);
    } finally {
      setLoading(false);
    }
  };

  const handleRescanTask = async (taskId: number) => {
    setLoading(true);
    try {
      await api.rescanTask(taskId);
      message.success('重新扫描已启动');
      await loadTasks();
    } catch (error: any) {
      console.error('Failed to rescan task:', error);
      message.error(`重新扫描失败: ${error.message || error}`);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteTask = async (taskId: number) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这个任务吗？',
      onOk: async () => {
        try {
          await api.deleteScanTask(taskId);
          message.success('任务已删除');
          await loadTasks();
          if (selectedTask?.id === taskId) {
            setSelectedTask(null);
          }
        } catch (error: any) {
          console.error('Failed to delete task:', error);
          message.error(`删除任务失败: ${error.message || error}`);
        }
      },
    });
  };

  const handleSelectTask = async (task: TaskConfig) => {
    setSelectedTask(task);
    // Don't load logs automatically - user can click "View Logs" button if needed
  };

  const handleViewLogs = async () => {
    if (!selectedTask) return;

    try {
      const logs = await api.getTaskLogsFromFile(selectedTask.id);
      setTaskLogs(prev => ({
        ...prev,
        [selectedTask.id]: logs || [],
      }));
      message.success('日志加载成功');
    } catch (error) {
      console.error('Failed to load task logs:', error);
      message.error('加载日志失败');
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'pending':
        return 'default';
      case 'running':
        return 'processing';
      case 'completed':
        return 'success';
      case 'failed':
        return 'error';
      default:
        return 'default';
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'pending':
        return '等待中';
      case 'running':
        return '运行中';
      case 'completed':
        return '已完成';
      case 'failed':
        return '失败';
      default:
        return status;
    }
  };

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

  // Load all templates for the template selection modal
  const loadAllTemplates = async () => {
    try {
      setLoading(true);
      const templates = await api.getAllTemplates();
      if (templates && Array.isArray(templates)) {
        setAllTemplates(templates);
        setFilteredTemplates(templates);
      } else {
        console.warn('Templates data is not an array:', templates);
        setAllTemplates([]);
        setFilteredTemplates([]);
        message.warning('模板数据格式异常');
      }
    } catch (error) {
      console.error('Failed to load templates:', error);
      message.error(`加载模板失败: ${error instanceof Error ? error.message : '未知错误'}`);
      setAllTemplates([]);
      setFilteredTemplates([]);
    } finally {
      setLoading(false);
    }
  };

  // Handle template search
  const handleTemplateSearch = (keyword: string) => {
    setTemplateSearchKeyword(keyword);
    if (!keyword.trim()) {
      setFilteredTemplates(allTemplates);
      return;
    }

    const filtered = allTemplates.filter(template => {
      const nameMatch = template.name?.toLowerCase().includes(keyword.toLowerCase()) || false;
      const idMatch = String(template.id).toLowerCase().includes(keyword.toLowerCase()) || false;
      
      // Handle tags as comma-separated string
      let tagsMatch = false;
      if (template.tags) {
        if (typeof template.tags === 'string') {
          // Split comma-separated tags and search
          const tagArray = template.tags.split(',').map(tag => tag.trim());
          tagsMatch = tagArray.some(tag => 
            tag.toLowerCase().includes(keyword.toLowerCase())
          );
        } else if (Array.isArray(template.tags)) {
          // Handle as array (fallback)
          tagsMatch = (template.tags as string[]).some((tag: string) => 
            tag.toLowerCase().includes(keyword.toLowerCase())
          );
        }
      }
      
      return nameMatch || idMatch || tagsMatch;
    });
    setFilteredTemplates(filtered);
  };

  // Open template selection modal
  const handleOpenTemplateModal = async () => {
    setTemplateModalVisible(true);
    setSelectedTemplateRows([]);
    setSelectedTemplateKeys([]);
    setTemplateSearchKeyword('');
    await loadAllTemplates();
  };

  // Add selected templates to the task
  const handleAddTemplates = () => {
    if (selectedTemplateRows.length === 0) {
      message.warning('请先选择模板');
      return;
    }

    const newTemplatePaths = selectedTemplateRows.map(t => t.file_path);
    const existingPaths = selectedTemplates;

    // Merge and deduplicate
    const merged = [...new Set([...existingPaths, ...newTemplatePaths])];
    setSelectedTemplates(merged);

    // Update sessionStorage
    sessionStorage.setItem('selectedTemplates', JSON.stringify(merged));

    message.success(`已添加 ${selectedTemplateRows.length} 个模板`);
    setTemplateModalVisible(false);
  };

  // Remove a specific template from selected list
  const handleRemoveTemplate = (templatePath: string) => {
    const updated = selectedTemplates.filter(t => t !== templatePath);
    setSelectedTemplates(updated);
    sessionStorage.setItem('selectedTemplates', JSON.stringify(updated));
  };

  // Add force update function
  const [, updateState] = useState({});
  const forceUpdate = useCallback(() => {
    updateState({});
  }, []);

  return (
    <div style={{ padding: 0, position: 'relative', backgroundColor: '#f5f5f5', minHeight: '100vh' }}>
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
        <Title level={4} style={{ margin: 0, fontSize: '16px', fontWeight: '500' }}>扫描任务</Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadTasks}>
            刷新
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateModalVisible(true)}
          >
            创建任务
          </Button>
        </Space>
      </div>

      <Row gutter={16} style={{ padding: '0 16px' }}>
        {/* Left: Task List */}
        <Col span={8}>
          <Card 
            title={
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span>任务列表</span>
                <Select
                  size="small"
                  value={sortOrder}
                  onChange={handleSortChange}
                  style={{ width: 100, fontSize: 12 }}
                >
                  <Option value="desc">最新</Option>
                  <Option value="asc">最早</Option>
                </Select>
              </div>
            } 
            bodyStyle={{ padding: 0, height: 'calc(100vh - 120px)', display: 'flex', flexDirection: 'column' }}
            style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}
          >
            <div style={{ 
              flex: 1, 
              overflowY: 'auto', 
              overflowX: 'hidden',
              scrollbarWidth: 'thin',
              scrollbarColor: '#d9d9d9 #f0f0f0'
            }} className="custom-scrollbar">
              <List
                dataSource={tasks}
                renderItem={(task) => (
                <List.Item
                  key={task.id}
                  onClick={() => handleSelectTask(task)}
                  className={`task-list-item ${selectedTask?.id === task.id ? 'selected' : ''}`}
                  style={{
                    cursor: 'pointer',
                    backgroundColor: selectedTask?.id === task.id ? '#e6f7ff' : 'transparent',
                    padding: '8px 12px',
                    borderBottom: '1px solid #f0f0f0',
                    transition: 'all 0.2s ease',
                  }}
                >
                  <div style={{ width: '100%' }}>
                    <div style={{ 
                      display: 'flex', 
                      justifyContent: 'space-between', 
                      alignItems: 'center',
                      marginBottom: '4px'
                    }}>
                      <Text strong style={{ fontSize: 14, lineHeight: '20px' }}>
                        {task.name}
                      </Text>
                      <Space size={4}>
                        <Tag 
                          color={getStatusColor(task.status)} 
                          style={{ fontSize: 11, padding: '2px 6px', margin: 0 }}
                        >
                          {getStatusText(task.status)}
                        </Tag>
                        {task.found_vulns > 0 && (
                          <Badge 
                            count={task.found_vulns} 
                            style={{ 
                              backgroundColor: '#ff4d4f',
                              fontSize: 10,
                              minWidth: 16,
                              height: 16,
                              lineHeight: '16px'
                            }} 
                          />
                        )}
                      </Space>
                    </div>
                    <div style={{ 
                      display: 'flex', 
                      justifyContent: 'space-between', 
                      alignItems: 'center',
                      fontSize: 12,
                      color: '#666'
                    }}>
                      <Text type="secondary" style={{ fontSize: 11, lineHeight: '16px' }}>
                        {task.targets.length} 个目标 · {task.pocs.length} 个模板
                      </Text>
                      <Text type="secondary" style={{ fontSize: 10, lineHeight: '16px' }}>
                        {new Date(task.created_at).toLocaleString('zh-CN', {
                          month: '2-digit',
                          day: '2-digit',
                          hour: '2-digit',
                          minute: '2-digit'
                        })}
                      </Text>
                    </div>
                    {/* 实时进度显示 - 始终显示，不仅仅是taskProgress存在时 */}
                    <div style={{ marginTop: 4 }}>
                      <Text type="secondary" style={{ fontSize: 10, lineHeight: '14px' }}>
                        进度: {taskProgress[task.id] ? (
                          <>
                            {taskProgress[task.id].completed_requests}/
                            {taskProgress[task.id].total_requests} (
                            {Math.round(taskProgress[task.id].percentage)}%)
                          </>
                        ) : (
                          <>
                            {task.completed_requests}/
                            {task.total_requests} (
                            {task.total_requests > 0 ? Math.round((task.completed_requests / task.total_requests) * 100) : 0}%)
                          </>
                        )}
                      </Text>
                    </div>
                  </div>
                </List.Item>
              )}
              locale={{ emptyText: '暂无任务' }}
            />
            </div>
          </Card>
        </Col>

        {/* Right: Task Details */}
        <Col span={16}>
          {selectedTask ? (
            <div>
              <Card
                title={
                  <Space>
                    <Text>{selectedTask.name}</Text>
                    <Tag color={getStatusColor(selectedTask.status)}>
                      {getStatusText(selectedTask.status)}
                    </Tag>
                  </Space>
                }
                extra={
                  <Space>
                    {selectedTask.status === 'pending' && (
                      <Button
                        type="primary"
                        icon={<PlayCircleOutlined />}
                        onClick={() => handleStartTask(selectedTask.id)}
                        loading={loading}
                      >
                        开始扫描
                      </Button>
                    )}
                    {(selectedTask.status === 'completed' || selectedTask.status === 'failed') && (
                      <>
                        <Button
                          type="primary"
                          icon={<ReloadOutlined />}
                          onClick={() => handleRescanTask(selectedTask.id)}
                          loading={loading}
                        >
                          重新扫描
                        </Button>
                        <Button
                          icon={<BugOutlined />}
                          onClick={handleViewLogs}
                        >
                          查看日志
                        </Button>
                      </>
                    )}
                    <Button
                      danger
                      icon={<DeleteOutlined />}
                      onClick={() => handleDeleteTask(selectedTask.id)}
                    >
                      删除
                    </Button>
                  </Space>
                }
              >
                {/* Progress - Show first if running */}
                {(taskProgress[selectedTask.id] || selectedTask.status === 'running') && (
                  <div style={{ marginBottom: 16 }}>
                    <ScanProgressComponent progress={taskProgress[selectedTask.id] || {
                      task_id: selectedTask.id,
                      total_requests: selectedTask.total_requests,
                      completed_requests: selectedTask.completed_requests,
                      found_vulns: selectedTask.found_vulns,
                      percentage: selectedTask.total_requests > 0 ? (selectedTask.completed_requests / selectedTask.total_requests) * 100 : 0,
                      status: selectedTask.status
                    }} />
                  </div>
                )}

                {/* Task Info - Compact layout */}
                <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
                  <Col span={12}>
                    <div style={{
                      padding: '8px 12px',
                      background: '#fafafa',
                      borderRadius: 4,
                      border: '1px solid #f0f0f0'
                    }}>
                      <Text type="secondary" style={{ fontSize: 12 }}>目标列表</Text>
                      <div style={{ marginTop: 4 }}>
                        {selectedTask.targets.slice(0, 3).map((target, index) => (
                          <Tag key={index} style={{ marginBottom: 2, fontSize: 11 }}>
                            {target}
                          </Tag>
                        ))}
                        {selectedTask.targets.length > 3 && (
                          <Tag style={{ fontSize: 11 }}>+{selectedTask.targets.length - 3} more</Tag>
                        )}
                      </div>
                    </div>
                  </Col>
                  <Col span={12}>
                    <div style={{
                      padding: '8px 12px',
                      background: '#fafafa',
                      borderRadius: 4,
                      border: '1px solid #f0f0f0'
                    }}>
                      <Text type="secondary" style={{ fontSize: 12 }}>发现漏洞</Text>
                      <div style={{ marginTop: 4 }}>
                        <Text strong style={{ fontSize: 20, color: '#ff4d4f' }}>
                          {selectedTask.found_vulns || 0}
                        </Text>
                        <Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>个</Text>
                      </div>
                    </div>
                  </Col>
                </Row>

                {/* POC Templates - Collapsed by default */}
                <div style={{
                  padding: '8px 12px',
                  background: '#fafafa',
                  borderRadius: 4,
                  border: '1px solid #f0f0f0',
                  marginBottom: 12
                }}>
                  <Text type="secondary" style={{ fontSize: 12 }}>POC 模板 ({selectedTask.pocs.length})</Text>
                  <div style={{ marginTop: 4, maxHeight: 60, overflow: 'auto' }}>
                    {selectedTask.pocs.map((poc, index) => {
                      const pocName = poc.split('/').pop()?.replace('.yaml', '') || poc;
                      return (
                        <Tag key={index} color="blue" style={{ marginBottom: 2, fontSize: 11 }}>
                          {pocName}
                        </Tag>
                      );
                    })}
                  </div>
                </div>

                {/* Logs */}
                {taskLogs[selectedTask.id] && taskLogs[selectedTask.id].length > 0 && (
                  <div style={{ marginTop: 12 }}>
                    <ScanLogs logs={taskLogs[selectedTask.id]} maxHeight={350} />
                  </div>
                )}
              </Card>
            </div>
          ) : (
            <Card>
              <div style={{ textAlign: 'center', padding: 40 }}>
                <Text type="secondary">请从左侧选择一个任务查看详情</Text>
              </div>
            </Card>
          )}
        </Col>
      </Row>

      {/* Create Task Modal */}
      <Modal
        title="创建扫描任务"
        open={createModalVisible}
        onOk={handleCreateTask}
        onCancel={() => setCreateModalVisible(false)}
        confirmLoading={creating}
        width={700}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <div>
            <Text strong>任务名称 (可选):</Text>
            <Input
              placeholder="留空自动生成"
              value={taskName}
              onChange={(e) => setTaskName(e.target.value)}
              style={{ marginTop: 8 }}
            />
          </div>

          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
              <Text strong>已选择 POC ({selectedTemplates.length}):</Text>
              <Button
                type="primary"
                size="small"
                icon={<FileAddOutlined />}
                onClick={handleOpenTemplateModal}
              >
                加载模板
              </Button>
            </div>
            <div style={{ marginTop: 8, maxHeight: 200, overflow: 'auto', border: '1px solid #f0f0f0', borderRadius: 4, padding: 8 }}>
              {selectedTemplates.length > 0 ? (
                selectedTemplates.map((tpl, index) => {
                  const templateName = tpl.split('/').pop()?.replace('.yaml', '') || tpl;
                  return (
                    <Tag
                      key={index}
                      closable
                      onClose={() => handleRemoveTemplate(tpl)}
                      color="blue"
                      style={{ marginBottom: 4 }}
                    >
                      {templateName}
                    </Tag>
                  );
                })
              ) : (
                <Text type="secondary">请点击"加载模板"按钮选择 POC 模板</Text>
              )}
            </div>
          </div>

          <div>
            <Text strong>目标地址 (每行一个):</Text>
            <TextArea
              placeholder={'http://example.com\nhttp://example2.com'}
              value={targets}
              onChange={(e) => setTargets(e.target.value)}
              rows={6}
              style={{ marginTop: 8 }}
            />
          </div>
        </Space>
      </Modal>

      {/* Template Selection Modal */}
      <Modal
        title="选择 POC 模板"
        open={templateModalVisible}
        onOk={handleAddTemplates}
        onCancel={() => setTemplateModalVisible(false)}
        width={900}
        okText="添加选中模板"
        confirmLoading={loading}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Input.Search
            placeholder="搜索模板 (名称、ID 或标签)"
            value={templateSearchKeyword}
            onChange={(e) => handleTemplateSearch(e.target.value)}
            allowClear
          />
          <Table
            dataSource={filteredTemplates}
            rowKey="id"
            size="small"
            loading={loading}
            rowSelection={{
              type: 'checkbox',
              selectedRowKeys: selectedTemplateKeys,
              onChange: (selectedKeys, selectedRows) => {
                setSelectedTemplateKeys(selectedKeys);
                setSelectedTemplateRows(selectedRows);
              },
            }}
            pagination={{
              pageSize: 10,
              showSizeChanger: false,
              showTotal: (total) => `共 ${total} 个模板`,
            }}
            columns={[
              {
                title: '模板名称',
                dataIndex: 'name',
                key: 'name',
                width: 250,
                ellipsis: true,
              },
              {
                title: 'ID',
                dataIndex: 'id',
                key: 'id',
                width: 200,
                ellipsis: true,
              },
              {
                title: '严重等级',
                dataIndex: 'severity',
                key: 'severity',
                width: 100,
                render: (severity: string) => (
                  <Tag color={getSeverityColor(severity)}>
                    {severity?.toUpperCase() || 'UNKNOWN'}
                  </Tag>
                ),
              },
              {
                title: '标签',
                dataIndex: 'tags',
                key: 'tags',
                ellipsis: true,
                render: (tags: string | string[]) => {
                  // Handle tags as comma-separated string or array
                  let tagArray: string[] = [];
                  if (typeof tags === 'string') {
                    tagArray = tags.split(',').map(tag => tag.trim()).filter(tag => tag);
                  } else if (Array.isArray(tags)) {
                    tagArray = tags;
                  }
                  
                  return (
                    <>
                      {tagArray.slice(0, 3).map((tag, index) => (
                        <Tag key={index} style={{ fontSize: 11 }}>
                          {tag}
                        </Tag>
                      ))}
                      {tagArray.length > 3 && <Tag style={{ fontSize: 11 }}>+{tagArray.length - 3}</Tag>}
                    </>
                  );
                },
              },
            ]}
          />
        </Space>
      </Modal>
    </div>
  );
};

export default ScanTasks;
