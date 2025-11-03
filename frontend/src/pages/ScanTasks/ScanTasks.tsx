import { useState, useEffect, useCallback, useRef } from 'react';
import { useLocation } from 'react-router-dom';
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
  Tabs,
} from 'antd';
import {
  PlusOutlined,
  PlayCircleOutlined,
  StopOutlined,
  DeleteOutlined,
  ReloadOutlined,
  BugOutlined,
  FileAddOutlined,
  DownloadOutlined,
} from '@ant-design/icons';
import { TaskConfig, ScanEvent, ScanProgress, ScanLogEntry, Template, HTTPRequestLog } from '../../types';
import { api } from '../../services/api';
import ScanProgressComponent from '../../components/ScanProgressComponent';
import ScanLogs from '../../components/ScanLogs';
import HTTPRequestTable from '../../components/HTTPRequestTable';
import './ScanTasks.css';

const { TabPane } = Tabs;

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;

// 新增：HTTP请求/响应事件类型
type HttpEvent = {
  template_id: string;
  target: string;
  request: string;
  response: string;
  timestamp: string;
};

const ScanTasks = () => {
  const location = useLocation();
  const [tasks, setTasks] = useState<TaskConfig[]>([]);
  const [selectedTask, setSelectedTask] = useState<TaskConfig | null>(null);
  const [selectedTemplates, setSelectedTemplates] = useState<string[]>([]);
  const [targets, setTargets] = useState('');
  const [taskName, setTaskName] = useState('');
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [editingTask, setEditingTask] = useState<TaskConfig | null>(null); // For editing existing task
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc');

  // Template selection modal states
  const [templateModalVisible, setTemplateModalVisible] = useState(false);
  const [allTemplates, setAllTemplates] = useState<Template[]>([]);
  const [filteredTemplates, setFilteredTemplates] = useState<Template[]>([]);
  const [templateSearchKeyword, setTemplateSearchKeyword] = useState('');
  const [selectedTemplateRows, setSelectedTemplateRows] = useState<Template[]>([]);
  const [selectedTemplateKeys, setSelectedTemplateKeys] = useState<React.Key[]>([]);

  // Real-time progress and logs for selected task
  const [taskProgress, setTaskProgress] = useState<Record<number, ScanProgress>>(() => {
    // 从localStorage恢复进度数据
    try {
      const saved = localStorage.getItem('wepoc_task_progress');
      return saved ? JSON.parse(saved) : {};
    } catch {
      return {};
    }
  });
  const [taskLogs, setTaskLogs] = useState<Record<number, ScanLogEntry[]>>({});

  // 新增：完整HTTP请求日志（从后端加载）
  const [taskHTTPLogs, setTaskHTTPLogs] = useState<Record<number, HTTPRequestLog[]>>({});
  const [loadingHTTPLogs, setLoadingHTTPLogs] = useState(false);

  // 新增：跟踪已完成的任务，避免重复提示 - 使用 useRef 避免闭包问题
  const completedTasksRef = useRef<Set<number>>(new Set());
  // 防抖保存进度数据到localStorage
  useEffect(() => {
    const timer = setTimeout(() => {
      try {
        // 保存所有任务的进度数据，包括已完成的任务（需要显示统计信息）
        // 已完成任务的统计数据（被过滤、被跳过、HTTP请求数）对用户很重要
        localStorage.setItem('wepoc_task_progress', JSON.stringify(taskProgress));
      } catch (error) {
        console.warn('Failed to save task progress to localStorage:', error);
      }
    }, 1000); // 防抖1秒

    return () => clearTimeout(timer);
  }, [taskProgress]);

  // Load tasks on mount
  useEffect(() => {
    const initializePage = async () => {
      try {
        await loadTasks();
        loadSelectedTemplates();

        // 请求浏览器通知权限（仅在首次访问时）
        if ('Notification' in window && Notification.permission === 'default') {
          try {
            await Notification.requestPermission();
          } catch (err) {
            console.warn('Failed to request notification permission:', err);
          }
        }

        // Check if there's data from template page navigation
        if (location.state) {
          const { selectedTemplates: templatesFromNav, taskName: taskNameFromNav } = location.state as any;
          
          if (templatesFromNav && templatesFromNav.length > 0) {
            // Set the selected templates from navigation
            const templatePaths = templatesFromNav.map((template: Template) => template.file_path);
            setSelectedTemplates(templatePaths);
            
            // Set task name if provided
            if (taskNameFromNav) {
              setTaskName(taskNameFromNav);
            }
            
            // Auto-open create modal
            setCreateModalVisible(true);
            
            message.success(`已选择 ${templatesFromNav.length} 个模板，请添加扫描目标`);
          }
        } else {
          // Check if there's a task name from template page (indicates user wants to create task)
          const savedTaskName = sessionStorage.getItem('taskName');
          const autoCreateTask = sessionStorage.getItem('autoCreateTask');
          
          if (savedTaskName && autoCreateTask === 'true') {
            setTaskName(savedTaskName);
            // Auto-create task directly if coming from template page with auto-create flag
            setTimeout(() => {
              handleAutoCreateTask(savedTaskName);
            }, 500); // Small delay to ensure templates are loaded
            
            // Clear the flags from session storage
            sessionStorage.removeItem('taskName');
            sessionStorage.removeItem('autoCreateTask');
          } else if (savedTaskName) {
            setTaskName(savedTaskName);
            // Auto-open create modal if coming from template page
            setCreateModalVisible(true);
            // Clear the taskName from session storage
            sessionStorage.removeItem('taskName');
          }
        }
      } catch (error) {
        console.error('Failed to initialize page:', error);
        message.error('页面初始化失败，请刷新重试');
      }
    };

    initializePage();
  }, [location.state]);

  // 去抖动控制 - 缓存最新事件，延迟更新但不丢失数据
  const pendingEventsRef = useRef<Record<number, ScanEvent>>({});
  const debounceTimersRef = useRef<Record<number, number>>({});
  const UPDATE_DEBOUNCE_MS = 150; // 150ms去抖动间隔（比之前的200ms更快）

  // 进度事件处理函数 - 提取为独立函数以便复用
  const processProgressEvent = useCallback((event: ScanEvent) => {
    console.log('Processing scan event:', event.event_type, event.data.status);

    // 特别打印完成状态的事件
    if (event.data.status === 'completed') {
      console.log(`✅ Received COMPLETED event for task ${event.task_id}:`, event.data);
      console.log(`   - Status: ${event.data.status}`);
      console.log(`   - Scanned: ${event.data.scanned_templates}/${event.data.total_templates}`);
      console.log(`   - Found Vulns: ${event.data.found_vulns}`);
    }

    // 检测是否是初始化事件（任务刚启动，所有进度为0）
    const isInitializing = event.data.status === 'running' &&
                           event.data.scanned_templates === 0 &&
                           event.data.completed_requests === 0;

    if (isInitializing) {
      console.log(`🔄 收到任务 ${event.task_id} 的初始化事件，清零旧数据`);
    }

    // 获取当前任务的进度数据
    const currentProgress = taskProgress[event.task_id];

    // 如果是初始化事件，强制使用新数据（不进行Math.max）
    // 否则确保数据单调递增
    const newScannedTemplates = isInitializing ? 0 : Math.max(
      event.data.scanned_templates || 0,
      currentProgress?.scanned_templates || 0
    );
    const newCompletedTemplates = isInitializing ? 0 : Math.max(
      event.data.completed_templates || 0,
      currentProgress?.completed_templates || 0
    );
    const newFoundVulns = isInitializing ? 0 : Math.max(
      event.data.found_vulns || 0,
      currentProgress?.found_vulns || 0
    );
    const newCompletedRequests = isInitializing ? 0 : Math.max(
      event.data.completed_requests || 0,
      currentProgress?.completed_requests || 0
    );

    // Update progress for the task
    setTaskProgress(prev => ({
      ...prev,
      [event.task_id]: {
        task_id: event.task_id,
        total_requests: event.data.total_requests || prev[event.task_id]?.total_requests || 0,
        completed_requests: newCompletedRequests,
        found_vulns: newFoundVulns,
        percentage: event.data.percentage || 0,
        status: event.data.status || 'pending',
        current_template: event.data.current_template || '',
        current_target: event.data.current_target || '',
        total_templates: event.data.total_templates || prev[event.task_id]?.total_templates || 0,
        completed_templates: newCompletedTemplates,
        scanned_templates: newScannedTemplates,
        failed_templates: Math.max(
          event.data.failed_templates || 0,
          prev[event.task_id]?.failed_templates || 0
        ),
        filtered_templates: Math.max(
          event.data.filtered_templates || 0,
          prev[event.task_id]?.filtered_templates || 0
        ),
        skipped_templates: Math.max(
          event.data.skipped_templates || 0,
          prev[event.task_id]?.skipped_templates || 0
        ),
        current_index: event.data.current_index || newScannedTemplates,
        selected_templates: event.data.selected_templates || prev[event.task_id]?.selected_templates || [],
        scanned_template_ids: event.data.scanned_template_ids || prev[event.task_id]?.scanned_template_ids || [],
        failed_template_ids: event.data.failed_template_ids || prev[event.task_id]?.failed_template_ids || [],
        filtered_template_ids: event.data.filtered_template_ids || prev[event.task_id]?.filtered_template_ids || [],
        skipped_template_ids: event.data.skipped_template_ids || prev[event.task_id]?.skipped_template_ids || [],
      },
    }));

    // Also update task status in the list
    setTasks(prevTasks =>
      prevTasks.map(task =>
        task.id === event.task_id
          ? {
              ...task,
              status: event.data.status,
              found_vulns: Math.max(newFoundVulns, task.found_vulns || 0),
              completed_requests: Math.max(newCompletedRequests, task.completed_requests || 0),
              total_requests: event.data.total_requests || task.total_requests,
            }
          : task
      )
    );

    // Also update selected task if it's the current one
    setSelectedTask(prev => {
      if (prev && prev.id === event.task_id) {
        return {
          ...prev,
          status: event.data.status,
          found_vulns: Math.max(newFoundVulns, prev.found_vulns || 0),
          completed_requests: Math.max(newCompletedRequests, prev.completed_requests || 0),
          total_requests: event.data.total_requests || prev.total_requests,
        };
      }
      return prev;
    });

    // 检查扫描是否完成，显示全局提示并刷新任务列表
    if (event.data.status === 'completed' && !completedTasksRef.current.has(event.task_id)) {
      // 立即标记为已完成，防止重复触发
      completedTasksRef.current.add(event.task_id);

      console.log(`🔄 任务 ${event.task_id} 完成，准备刷新任务列表...`);

      // 获取任务信息 - 使用 setTasks 的回调来获取最新的 tasks 值
      let taskName = `任务 ${event.task_id}`;
      setTasks(currentTasks => {
        const task = currentTasks.find(t => t.id === event.task_id);
        if (task) {
          taskName = task.name;
        }
        return currentTasks; // 不修改 tasks，只是读取
      });

      // 延迟刷新任务列表，确保后端已保存最终状态
      setTimeout(() => {
        console.log(`🔄 强制刷新任务列表以同步完成状态...`);
        loadTasks();
      }, 500);

      const vulnCount = newFoundVulns;
      const scannedCount = newScannedTemplates;
      const totalCount = event.data.total_templates || 0;
      const filteredCount = event.data.filtered_templates || 0;
      const skippedCount = event.data.skipped_templates || 0;

      // 保留已完成任务的最终统计数据，不要删除
      // 用户需要查看完整的统计信息（被过滤、被跳过、HTTP请求数等）

      // 显示扫描完成提示 - 只显示一次，简洁版
      setTimeout(() => {
        const notificationConfig = {
          content: (
            <div>
              <div style={{
                fontWeight: 'bold',
                marginBottom: 6,
                fontSize: 15,
                color: vulnCount > 0 ? '#ff4d4f' : '#52c41a'
              }}>
                {vulnCount > 0 ? '⚠️ 扫描完成 - 发现漏洞！' : '✅ 扫描完成！'}
              </div>
              <div style={{ fontSize: 13, color: '#333', marginBottom: 4, fontWeight: 500 }}>
                {taskName}
              </div>
              <div style={{
                fontSize: 13,
                color: '#666',
                padding: '6px 0',
                borderTop: '1px solid #f0f0f0',
                marginTop: 4
              }}>
                扫描 <Text strong style={{ color: '#1890ff' }}>{scannedCount}/{totalCount}</Text> 个POC
                {vulnCount > 0 && (
                  <span>，发现 <Text strong style={{ color: '#ff4d4f' }}>{vulnCount}</Text> 个漏洞</span>
                )}
              </div>
            </div>
          ),
          duration: vulnCount > 0 ? 10 : 6,
          style: {
            marginTop: 60,
          }
        };

        // 统一使用 success，通过内容颜色区分
        message.success(notificationConfig);

        // 尝试发送浏览器通知（如果用户授权）
        if ('Notification' in window && Notification.permission === 'granted') {
          try {
            new Notification('WePOC - 扫描完成', {
              body: `${taskName}\n扫描 ${scannedCount}/${totalCount} 个POC${vulnCount > 0 ? `，发现 ${vulnCount} 个漏洞` : ''}`,
              icon: '/favicon.ico',
              tag: `scan-complete-${event.task_id}`,
              requireInteraction: vulnCount > 0 // 发现漏洞时需要用户交互
            });
          } catch (err) {
            console.warn('Failed to show browser notification:', err);
          }
        }
      }, 500); // 延迟500ms显示，确保数据更新完成
    }
  }, [taskProgress]);

  // Listen to scan events
  useEffect(() => {
    const unsubscribe = api.onScanEvent((event: ScanEvent) => {
      // 对于进度事件，使用去抖动策略：缓存最新事件，延迟处理
      if (event.event_type === 'progress' && event.data.status !== 'completed') {
        // 缓存最新事件
        pendingEventsRef.current[event.task_id] = event;

        // 清除之前的定时器
        if (debounceTimersRef.current[event.task_id]) {
          clearTimeout(debounceTimersRef.current[event.task_id]);
        }

        // 设置新的延迟处理定时器
        debounceTimersRef.current[event.task_id] = setTimeout(() => {
          const latestEvent = pendingEventsRef.current[event.task_id];
          if (latestEvent) {
            // 处理缓存的最新事件
            processProgressEvent(latestEvent);
            delete pendingEventsRef.current[event.task_id];
          }
        }, UPDATE_DEBOUNCE_MS);

        return; // 暂不处理，等待去抖动定时器触发
      }

      // 非进度事件或完成状态，立即处理
      if (event.event_type === 'progress') {
        processProgressEvent(event);
      }

      // Tasks state will automatically trigger re-render
    });

    // 清理函数：组件卸载时清除所有定时器
    return () => {
      unsubscribe();

      // 清除所有待处理的去抖动定时器
      Object.values(debounceTimersRef.current).forEach(timer => {
        if (timer) clearTimeout(timer);
      });
      debounceTimersRef.current = {};

      // 处理所有待处理的事件（防止数据丢失）
      Object.entries(pendingEventsRef.current).forEach(([taskId, event]) => {
        processProgressEvent(event);
      });
      pendingEventsRef.current = {};
    };
  }, [processProgressEvent]);

  const loadTasks = async () => {
    try {
      const allTasks = await api.getAllScanTasks();
      if (allTasks && Array.isArray(allTasks)) {
        const sortedTasks = sortTasksByTime(allTasks, sortOrder);
        setTasks(sortedTasks);
      } else {
        console.warn('Tasks data is not an array:', allTasks);
        setTasks([]);
        // 只有在确实有数据但格式错误时才显示警告，空数据不显示警告
        if (allTasks !== null && allTasks !== undefined) {
          message.warning('任务数据格式异常');
        }
      }
    } catch (error) {
      console.error('Failed to load tasks:', error);
      message.error(`加载任务失败: ${error instanceof Error ? error.message : '未知错误'}`);
      setTasks([]);
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
    try {
      const stored = sessionStorage.getItem('selectedTemplates');
      if (stored) {
        const templates = JSON.parse(stored);
        if (Array.isArray(templates)) {
          setSelectedTemplates(templates);
        }
      }
    } catch (error) {
      console.error('Failed to load selected templates:', error);
      sessionStorage.removeItem('selectedTemplates');
    }
  };

  // Auto-create task when coming from template page
  const handleAutoCreateTask = async (autoTaskName: string) => {
    if (selectedTemplates.length === 0) {
      message.info('已加载模板，请填写目标或选择更多POC');
      setCreateModalVisible(true);
      return;
    }

    // Create task with auto-generated name and empty targets (user can add later)
    setCreating(true);
    try {
      console.log('Auto-creating task with:', {
        templates: selectedTemplates,
        targets: [], // Empty targets initially
        taskName: autoTaskName
      });
      
      // Convert arrays to JSON strings for backend
      const pocsJSON = JSON.stringify(selectedTemplates);
      const targetsJSON = JSON.stringify([]);
      
      const task = await api.createScanTask(pocsJSON, targetsJSON, autoTaskName);
      
      if (task && task.id) {
        message.success(`任务 "${autoTaskName}" 创建成功，请添加目标地址后开始扫描`);
        await loadTasks();
        // Select the newly created task
        setSelectedTask(task);
        // Show info message to guide user to add targets
        setTimeout(() => {
          message.info('请在右侧添加目标地址，然后点击开始扫描', 5);
        }, 1000);
      } else {
        throw new Error('任务创建失败：返回数据异常');
      }
    } catch (error: any) {
      console.error('Failed to auto-create task:', error);
      const errorMessage = error?.message || error?.toString() || '未知错误';
      message.error(`自动创建任务失败: ${errorMessage}`);
      // Fallback to manual creation modal
      setCreateModalVisible(true);
    } finally {
      setCreating(false);
    }
  };

  const handleCreateTask = async () => {
    if (!taskName.trim()) {
      message.error('请输入任务名称');
      return;
    }

    if (selectedTemplates.length === 0) {
      message.error('请选择至少一个模板');
      return;
    }

    if (!targets.trim()) {
      message.error('请输入至少一个目标');
      return;
    }

    const targetList = targets.split('\n').filter(t => t.trim());
    if (targetList.length === 0) {
      message.error('请输入有效的目标地址');
      return;
    }

    setCreating(true);
    try {
      console.log('Creating task with:', {
        templates: selectedTemplates,
        targets: targetList,
        taskName: taskName
      });
      
      // Convert arrays to JSON strings for backend
      const pocsJSON = JSON.stringify(selectedTemplates);
      const targetsJSON = JSON.stringify(targetList);
      
      let task;
      if (editingTask) {
        // Update existing task
        task = await api.updateScanTask(editingTask.id, pocsJSON, targetsJSON, taskName);
        message.success('任务更新成功');
      } else {
        // Create new task
        task = await api.createScanTask(pocsJSON, targetsJSON, taskName);
        message.success('任务创建成功');
      }
      
      if (task && task.id) {
        // Reset form and close modal
        setCreateModalVisible(false);
        setTargets('');
        setTaskName('');
        setSelectedTemplates([]);
        setEditingTask(null);
        
        // Clear sessionStorage
        sessionStorage.removeItem('selectedTemplates');
        
        // Reload tasks to reflect changes
        await loadTasks();
        
        // Select the newly created/updated task
        setSelectedTask(task);
      } else {
        throw new Error('任务操作失败：返回数据异常');
      }
    } catch (error: any) {
      console.error('Failed to create/update task:', error);
      const errorMessage = error?.message || error?.toString() || '未知错误';
      message.error(`${editingTask ? '更新' : '创建'}任务失败: ${errorMessage}`);
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

    // Auto-load HTTP logs for completed/failed tasks to show count in tab
    if ((task.status === 'completed' || task.status === 'failed') && !taskHTTPLogs[task.id]) {
      handleLoadHTTPLogs(task.id);
    }
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

  // 新增：加载HTTP请求日志
  const handleLoadHTTPLogs = async (taskId: number) => {
    try {
      setLoadingHTTPLogs(true);
      const logs = await api.getTaskHTTPLogs(taskId);
      setTaskHTTPLogs(prev => ({
        ...prev,
        [taskId]: logs || [],
      }));
      console.log(`✅ 加载了 ${logs?.length || 0} 条HTTP请求日志`);
    } catch (error) {
      console.error('Failed to load HTTP logs:', error);
      message.error('加载HTTP请求日志失败');
    } finally {
      setLoadingHTTPLogs(false);
    }
  };

  // 新增：导出扫描结果
  const handleExportResult = async () => {
    if (!selectedTask) return;

    try {
      const filePath = await api.exportTaskResultAsJSON(selectedTask.id);
      if (filePath) {
        message.success(`导出成功: ${filePath}`);
      }
    } catch (error) {
      console.error('Failed to export result:', error);
      message.error('导出失败');
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
    
    // Update state using functional update to ensure latest state
    setSelectedTemplates(prevTemplates => {
      const updatedTemplates = [...new Set([...prevTemplates, ...newTemplatePaths])];
      // Update sessionStorage with the latest state
      sessionStorage.setItem('selectedTemplates', JSON.stringify(updatedTemplates));
      return updatedTemplates;
    });

    message.success(`已添加 ${selectedTemplateRows.length} 个模板`);
    setTemplateModalVisible(false);
    
    // Clear selection state
    setSelectedTemplateRows([]);
    setSelectedTemplateKeys([]);
  };

  // Remove a specific template from selected list
  const handleRemoveTemplate = (templatePath: string) => {
    setSelectedTemplates(prevTemplates => {
      const updatedTemplates = prevTemplates.filter(t => t !== templatePath);
      // Update sessionStorage with the latest state
      sessionStorage.setItem('selectedTemplates', JSON.stringify(updatedTemplates));
      return updatedTemplates;
    });
  };

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <div style={{
        padding: '16px 20px',
        backgroundColor: '#fff',
        borderBottom: '1px solid #e8e8e8',
        boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        flexShrink: 0
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

      <Row gutter={16} style={{ padding: '16px 16px 0', flex: 1, minHeight: 0 }}>
        {/* Left: Task List */}
        <Col span={8} style={{ height: '100%' }}>
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
            bodyStyle={{ 
              padding: 0, 
              height: 'calc(100vh - 140px)', 
              display: 'flex', 
              flexDirection: 'column',
              position: 'relative'
            }}
            style={{ 
              height: '100%', 
              display: 'flex', 
              flexDirection: 'column'
            }}
          >
            {/* Fixed header area */}
            <div style={{
              position: 'sticky',
              top: 0,
              zIndex: 10,
              backgroundColor: '#fafafa',
              borderBottom: '1px solid #f0f0f0',
              padding: '8px 12px',
              fontSize: 12,
              color: '#666',
              fontWeight: 500
            }}>
              共 {tasks.length} 个任务
            </div>
            
            {/* Scrollable content area */}
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
                  </div>
                </List.Item>
              )}
              locale={{ emptyText: '暂无任务' }}
            />
            </div>
          </Card>
        </Col>

        {/* Right: Task Details */}
        <Col span={16} style={{ height: '100%', overflow: 'hidden' }}>
          {selectedTask ? (
            <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
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
                style={{ 
                  height: '100%', 
                  display: 'flex', 
                  flexDirection: 'column' 
                }}
                bodyStyle={{
                  flex: 1,
                  overflow: 'auto',  // 改为auto以支持滚动
                  display: 'flex',
                  flexDirection: 'column',
                  padding: '16px'
                }}
              >
                {/* 扫描状态文字提示 - 只在运行中时显示 */}
                {selectedTask.status === 'running' && (
                  <div style={{
                    marginBottom: 16,
                    flexShrink: 0,
                    padding: '12px 16px',
                    background: '#e6f7ff',
                    border: '1px solid #91d5ff',
                    borderRadius: '4px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '12px'
                  }}>
                    <span style={{ fontSize: '14px', color: '#1890ff', fontWeight: 500 }}>
                      🔄 正在扫描中...
                    </span>
                    {taskProgress[selectedTask.id] && taskProgress[selectedTask.id].found_vulns > 0 && (
                      <span style={{ fontSize: '13px', color: '#ff4d4f', fontWeight: 600 }}>
                        发现 {taskProgress[selectedTask.id].found_vulns} 个漏洞
                      </span>
                    )}
                  </div>
                )}

                {/* Task Info - 移除了 HTTP 请求 tab */}
                <div style={{
                  flex: 1,
                  overflow: 'auto'
                }}>
                  <div style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '12px'
                  }}>
                      <Row gutter={[12, 12]} style={{ marginBottom: 12, flexShrink: 0 }}>
                    <Col span={12}>
                      <div style={{
                        padding: '8px 12px',
                        background: selectedTask.targets.length === 0 ? '#fff2e8' : '#fafafa',
                        borderRadius: 4,
                        border: selectedTask.targets.length === 0 ? '1px solid #ffbb96' : '1px solid #f0f0f0'
                      }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>目标列表</Text>
                        {selectedTask.targets.length === 0 ? (
                          <div style={{ marginTop: 4 }}>
                            <Text type="warning" style={{ fontSize: 12 }}>
                              ⚠️ 请添加目标地址后开始扫描
                            </Text>
                            <br />
                            <Button 
                               type="link" 
                               size="small" 
                               style={{ padding: 0, height: 'auto', fontSize: 11 }}
                               onClick={() => {
                                 setEditingTask(selectedTask);
                                 setTaskName(selectedTask.name);
                                 setTargets(selectedTask.targets.join('\n'));
                                 setSelectedTemplates(selectedTask.pocs);
                                 setCreateModalVisible(true);
                               }}
                             >
                               点击添加目标地址
                             </Button>
                          </div>
                        ) : (
                          <div style={{ marginTop: 4 }}>
                            {selectedTask.targets.slice(0, 3).map((target, index) => (
                              <Tag key={index} style={{ marginBottom: 2, fontSize: 11 }}>
                                {target}
                              </Tag>
                            ))}
                            {selectedTask.targets.length > 3 && (
                               <Text type="secondary" style={{ fontSize: 11 }}>
                                 +{selectedTask.targets.length - 3} 更多
                               </Text>
                             )}
                          </div>
                        )}
                      </div>
                    </Col>
                    <Col span={12}>
                      <div style={{
                         padding: '8px 12px',
                         background: '#fafafa',
                         borderRadius: 4,
                         border: '1px solid #f0f0f0'
                       }}>
                         <Text type="secondary" style={{ fontSize: 12 }}>POC 模板</Text>
                         <div style={{ marginTop: 4 }}>
                           {selectedTask.pocs.slice(0, 3).map((poc, index) => {
                             const pocName = poc.split('/').pop()?.replace('.yaml', '') || poc;
                             return (
                               <Tag key={index} style={{ marginBottom: 2, fontSize: 11 }}>
                                 {pocName}
                               </Tag>
                             );
                           })}
                           {selectedTask.pocs.length > 3 && (
                             <Text type="secondary" style={{ fontSize: 11 }}>
                               +{selectedTask.pocs.length - 3} 更多
                             </Text>
                           )}
                         </div>
                       </div>
                    </Col>
                  </Row>

                  {/* 扫描结果摘要 - 紧凑版 */}
                   {selectedTask.status !== 'pending' && (selectedTask.status === 'completed' || taskProgress[selectedTask.id]) && (
                     <div style={{
                       padding: '14px',
                       background: '#fafafa',
                       borderRadius: 8,
                       marginBottom: 12,
                       flexShrink: 0,
                       border: '1px solid #e8e8e8'
                     }}>
                       <Row gutter={12} style={{ marginBottom: 12 }}>
                         <Col span={6}>
                           <div style={{ textAlign: 'center', padding: '8px', background: '#fff', borderRadius: 4 }}>
                             <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>已扫描</Text>
                             <Text strong style={{ fontSize: 18, color: '#1890ff' }}>
                               {taskProgress[selectedTask.id]?.scanned_templates || (selectedTask.status === 'completed' ? selectedTask.pocs.length : 0)}
                             </Text>
                           </div>
                         </Col>
                         <Col span={6}>
                           <div style={{ textAlign: 'center', padding: '8px', background: '#fff', borderRadius: 4 }}>
                             <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>发现漏洞</Text>
                             <Text strong style={{ fontSize: 18, color: selectedTask.found_vulns > 0 ? '#ff4d4f' : '#52c41a' }}>
                               {selectedTask.found_vulns || 0}
                             </Text>
                           </div>
                         </Col>
                         <Col span={6}>
                           <div style={{ textAlign: 'center', padding: '8px', background: '#fff7e6', borderRadius: 4 }}>
                             <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>被过滤</Text>
                             <Text strong style={{ fontSize: 18, color: '#faad14' }}>
                               {taskProgress[selectedTask.id]?.filtered_templates || 0}
                             </Text>
                           </div>
                         </Col>
                         <Col span={6}>
                           <div style={{ textAlign: 'center', padding: '8px', background: '#f5f5f5', borderRadius: 4 }}>
                             <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>被跳过</Text>
                             <Text strong style={{ fontSize: 18, color: '#8c8c8c' }}>
                               {taskProgress[selectedTask.id]?.skipped_templates || 0}
                             </Text>
                           </div>
                         </Col>
                       </Row>

                       {/* HTTP请求统计 */}
                       <div style={{
                         padding: '8px 12px',
                         background: '#fff',
                         borderRadius: 4,
                         display: 'flex',
                         justifyContent: 'space-between',
                         alignItems: 'center'
                       }}>
                         <Text type="secondary" style={{ fontSize: 12 }}>HTTP请求数：</Text>
                         <Text strong style={{ fontSize: 14, color: '#1890ff' }}>
                           {selectedTask.completed_requests || 0}
                         </Text>
                       </div>

                       {/* POC总数验证 */}
                       {selectedTask.status === 'completed' && taskProgress[selectedTask.id] && (
                         <div style={{
                           marginTop: 8,
                           padding: '6px 12px',
                           background: '#e6f7ff',
                           borderRadius: 4,
                           fontSize: 11,
                           color: '#666',
                           textAlign: 'center'
                         }}>
                           ✓ 验证：{taskProgress[selectedTask.id].scanned_templates} + {taskProgress[selectedTask.id].filtered_templates} + {taskProgress[selectedTask.id].skipped_templates} = {selectedTask.pocs.length} 个POC
                         </div>
                       )}
                     </div>
                   )}

                   {/* 日志查看器 */}
                   <div style={{ flexShrink: 0 }}>
                     <ScanLogs logs={taskLogs[selectedTask.id] || []} />
                   </div>
                  </div>
                </div>
              </Card>
            </div>
          ) : (
            <Card style={{ height: '100%' }}>
              <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Text type="secondary">请选择左侧任务查看详情</Text>
              </div>
            </Card>
          )}
        </Col>
      </Row>

      {/* Create Task Modal */}
      <Modal
        title={editingTask ? "编辑扫描任务" : "创建扫描任务"}
        open={createModalVisible}
        onOk={handleCreateTask}
        onCancel={() => {
           setCreateModalVisible(false);
           setEditingTask(null);
           setTaskName('');
           setTargets('');
           setSelectedTemplates([]);
         }}
        confirmLoading={creating}
        width={700}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <div>
            <Text strong>任务名称 (可选):</Text>
            <Input
              placeholder={editingTask ? "编辑任务名称" : "留空自动生成"}
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
                添加更多POC
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
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Input.Search
              placeholder="搜索模板 (名称、ID 或标签)"
              value={templateSearchKeyword}
              onChange={(e) => handleTemplateSearch(e.target.value)}
              allowClear
              style={{ flex: 1, marginRight: 16 }}
            />
            <Button
              type="primary"
              size="small"
              onClick={() => {
                const allFilteredKeys = filteredTemplates.map(template => template.id);
                setSelectedTemplateKeys(allFilteredKeys);
                setSelectedTemplateRows(filteredTemplates);
                message.success(`已选择所有过滤后的 ${filteredTemplates.length} 个模板`);
              }}
              disabled={filteredTemplates.length === 0}
            >
              全选过滤结果 ({filteredTemplates.length})
            </Button>
          </div>
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
              pageSize: 50,
              showSizeChanger: true,
              showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 个模板`,
              pageSizeOptions: ['20', '50', '100', '200', '500'],
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
