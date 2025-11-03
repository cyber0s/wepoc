import { useState, useEffect } from 'react';
import { Table, Tag, Button, Input, Select, Space, message, Modal } from 'antd';
import { SearchOutlined, EyeOutlined, DownloadOutlined } from '@ant-design/icons';
import { HTTPRequestLog } from '../types';
import RequestResponseViewer from './RequestResponseViewer';
import './HTTPRequestTable.css';

const { Search } = Input;
const { Option } = Select;

interface HTTPRequestTableProps {
  taskId: number;
  httpLogs: HTTPRequestLog[];
  loading?: boolean;
  onExport?: () => void;
}

const HTTPRequestTable = ({ taskId, httpLogs, loading = false, onExport }: HTTPRequestTableProps) => {
  const [filteredLogs, setFilteredLogs] = useState<HTTPRequestLog[]>(httpLogs);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [severityFilter, setSeverityFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [selectedLog, setSelectedLog] = useState<HTTPRequestLog | null>(null);
  const [viewerVisible, setViewerVisible] = useState(false);

  useEffect(() => {
    setFilteredLogs(httpLogs);
  }, [httpLogs]);

  // 过滤逻辑
  useEffect(() => {
    let filtered = [...httpLogs];

    // 关键词搜索
    if (searchKeyword) {
      filtered = filtered.filter(log =>
        log.template_id.toLowerCase().includes(searchKeyword.toLowerCase()) ||
        log.target.toLowerCase().includes(searchKeyword.toLowerCase())
      );
    }

    // 严重度过滤
    if (severityFilter) {
      filtered = filtered.filter(log => log.severity === severityFilter);
    }

    // 状态码过滤
    if (statusFilter) {
      if (statusFilter === '2xx') {
        filtered = filtered.filter(log => log.status_code >= 200 && log.status_code < 300);
      } else if (statusFilter === '4xx') {
        filtered = filtered.filter(log => log.status_code >= 400 && log.status_code < 500);
      } else if (statusFilter === '5xx') {
        filtered = filtered.filter(log => log.status_code >= 500);
      }
    }

    setFilteredLogs(filtered);
  }, [searchKeyword, severityFilter, statusFilter, httpLogs]);

  const getSeverityColor = (severity: string) => {
    switch (severity?.toLowerCase()) {
      case 'critical': return 'red';
      case 'high': return 'orange';
      case 'medium': return 'gold';
      case 'low': return 'blue';
      default: return 'default';
    }
  };

  const getStatusCodeColor = (code: number) => {
    if (code >= 200 && code < 300) return 'success';
    if (code >= 300 && code < 400) return 'processing';
    if (code >= 400 && code < 500) return 'warning';
    if (code >= 500) return 'error';
    return 'default';
  };

  const handleViewRequest = (record: HTTPRequestLog) => {
    setSelectedLog(record);
    setViewerVisible(true);
  };

  const columns = [
    {
      title: '序号',
      dataIndex: 'id',
      key: 'id',
      width: 70,
      fixed: 'left' as const,
    },
    {
      title: 'POC ID',
      dataIndex: 'template_id',
      key: 'template_id',
      width: 200,
      ellipsis: true,
      render: (id: string) => (
        <span style={{ color: '#1890ff', fontSize: '12px' }}>{id}</span>
      ),
    },
    {
      title: '名称',
      dataIndex: 'template_name',
      key: 'template_name',
      width: 200,
      ellipsis: true,
    },
    {
      title: '严重性',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      align: 'center' as const,
      render: (severity: string) => (
        <Tag color={getSeverityColor(severity)}>
          {severity?.toUpperCase() || 'UNKNOWN'}
        </Tag>
      ),
    },
    {
      title: '目标',
      dataIndex: 'target',
      key: 'target',
      width: 250,
      ellipsis: true,
    },
    {
      title: '方法',
      dataIndex: 'method',
      key: 'method',
      width: 80,
      align: 'center' as const,
    },
    {
      title: '状态码',
      dataIndex: 'status_code',
      key: 'status_code',
      width: 90,
      align: 'center' as const,
      render: (code: number) => (
        <Tag color={getStatusCodeColor(code)}>{code}</Tag>
      ),
    },
    {
      title: '漏洞',
      dataIndex: 'is_vuln_found',
      key: 'is_vuln_found',
      width: 80,
      align: 'center' as const,
      render: (found: boolean) => (
        found ? <Tag color="red">是</Tag> : <Tag>否</Tag>
      ),
    },
    {
      title: '耗时(ms)',
      dataIndex: 'duration_ms',
      key: 'duration_ms',
      width: 100,
      align: 'center' as const,
      render: (ms: number) => ms > 0 ? ms : '-',
    },
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (time: string) => {
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
      },
    },
    {
      title: '操作',
      key: 'actions',
      width: 100,
      align: 'center' as const,
      fixed: 'right' as const,
      render: (_: any, record: HTTPRequestLog) => (
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => handleViewRequest(record)}
        >
          查看
        </Button>
      ),
    },
  ];

  return (
    <div className="http-request-table">
      <div className="table-toolbar">
        <Space>
          <Search
            placeholder="搜索POC ID或目标..."
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
            style={{ width: 250 }}
            size="small"
            allowClear
          />
          <Select
            placeholder="严重度"
            value={severityFilter || undefined}
            onChange={setSeverityFilter}
            allowClear
            size="small"
            style={{ width: 120 }}
          >
            <Option value="critical">Critical</Option>
            <Option value="high">High</Option>
            <Option value="medium">Medium</Option>
            <Option value="low">Low</Option>
            <Option value="info">Info</Option>
          </Select>
          <Select
            placeholder="状态码"
            value={statusFilter || undefined}
            onChange={setStatusFilter}
            allowClear
            size="small"
            style={{ width: 100 }}
          >
            <Option value="2xx">2xx</Option>
            <Option value="4xx">4xx</Option>
            <Option value="5xx">5xx</Option>
          </Select>
        </Space>

        <Space>
          <span style={{ fontSize: '12px', color: '#666' }}>
            显示 <strong style={{ color: '#1890ff' }}>{filteredLogs.length}</strong> / {httpLogs.length} 条记录
          </span>
          {onExport && (
            <Button
              icon={<DownloadOutlined />}
              onClick={onExport}
              size="small"
            >
              导出结果
            </Button>
          )}
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={filteredLogs}
        rowKey="id"
        loading={loading}
        pagination={{
          pageSize: 50,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          pageSizeOptions: ['20', '50', '100', '200'],
        }}
        size="small"
        scroll={{ x: 1400, y: 'calc(100vh - 300px)' }}
        rowClassName={(record) => record.is_vuln_found ? 'vuln-row' : ''}
      />

      {selectedLog && (
        <RequestResponseViewer
          visible={viewerVisible}
          log={selectedLog}
          onClose={() => {
            setViewerVisible(false);
            setSelectedLog(null);
          }}
        />
      )}
    </div>
  );
};

export default HTTPRequestTable;
