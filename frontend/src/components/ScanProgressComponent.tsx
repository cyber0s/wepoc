import React, { useState, useEffect, useRef, useMemo } from 'react';
import { Progress, Statistic, Row, Col, Card, Typography, Tag, Divider, Button } from 'antd';
import { CheckCircleOutlined, BugOutlined, SyncOutlined, FileTextOutlined, AimOutlined, ExclamationCircleOutlined, GlobalOutlined, DownOutlined, UpOutlined } from '@ant-design/icons';
import { ScanProgress } from '../types';

const { Text, Paragraph } = Typography;

interface ScanProgressProps {
  progress: ScanProgress;
  httpLogs?: Array<{
    template_id: string;
    target: string;
    request: string;
    response: string;
    timestamp: string;
  }>;
}

const ScanProgressComponent: React.FC<ScanProgressProps> = ({ progress, httpLogs = [] }) => {
  // 使用状态来平滑进度变化
  const [smoothProgress, setSmoothProgress] = useState(0);
  const [displayProgress, setDisplayProgress] = useState({
    scanned_templates: 0,
    completed_templates: 0,
    found_vulns: 0,
  });
  const prevProgressRef = useRef(0);

  // 控制是否显示完整模板列表
  const [showAllTemplates, setShowAllTemplates] = useState(false);
  const MAX_VISIBLE_TEMPLATES = 20; // 默认只显示20个

  // 使用 useMemo 优化HTTP请求统计计算，避免每次渲染都重新计算
  const { methodCounts, totalRequests } = useMemo(() => {
    const methodCounts: Record<string, number> = {};
    let totalRequests = 0;

    // 限制处理的日志数量，避免性能问题
    const logsToProcess = httpLogs.slice(-100); // 只处理最近100条

    logsToProcess.forEach(log => {
      if (log.request) {
        totalRequests++;
        const firstLine = log.request.split('\n')[0];
        const method = firstLine.split(' ')[0]?.toUpperCase();
        if (method) {
          methodCounts[method] = (methodCounts[method] || 0) + 1;
        }
      }
    });

    return { methodCounts, totalRequests };
  }, [httpLogs]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return '#1890ff';
      case 'completed':
        return '#52c41a';
      case 'failed':
        return '#ff4d4f';
      case 'pending':
        return '#d9d9d9';
      default:
        return '#1890ff';
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'running':
        return '扫描中...';
      case 'completed':
        return '已完成';
      case 'failed':
        return '失败';
      case 'pending':
        return '等待中';
      default:
        return status;
    }
  };

  // 获取HTTP方法对应的颜色
  const getMethodColor = (method: string) => {
    switch (method.toUpperCase()) {
      case 'GET':
        return 'blue';
      case 'POST':
        return 'green';
      case 'PUT':
        return 'orange';
      case 'DELETE':
        return 'red';
      case 'PATCH':
        return 'purple';
      case 'HEAD':
        return 'cyan';
      case 'OPTIONS':
        return 'geekblue';
      default:
        return 'default';
    }
  };

  // 计算POC进度百分比 - 包含已扫描、过滤和跳过的POC，确保进度流畅
  const getPocProgressPercent = () => {
    if (progress.total_templates === 0) return 0;

    // 计算总处理数：已扫描 + 被过滤 + 被跳过
    const scannedCount = progress.scanned_templates || progress.completed_templates || 0;
    const filteredCount = progress.filtered_templates || 0;
    const skippedCount = progress.skipped_templates || 0;
    const totalProcessed = scannedCount + filteredCount + skippedCount;

    const percent = Math.max(0, Math.min(100, Math.round((totalProcessed / progress.total_templates) * 100)));

    // 确保进度只能增加，不能回退
    const currentPercent = Math.max(percent, prevProgressRef.current);
    prevProgressRef.current = currentPercent;

    return currentPercent;
  };

  // 平滑更新进度
  useEffect(() => {
    const targetProgress = getPocProgressPercent();
    
    // 如果目标进度大于当前显示进度，则平滑过渡
    if (targetProgress > smoothProgress) {
      const duration = 500; // 500ms过渡时间
      const steps = 20;
      const stepValue = (targetProgress - smoothProgress) / steps;
      const stepInterval = duration / steps;
      
      let currentStep = 0;
      const timer = setInterval(() => {
        currentStep++;
        if (currentStep >= steps) {
          setSmoothProgress(targetProgress);
          clearInterval(timer);
        } else {
          setSmoothProgress(prev => prev + stepValue);
        }
      }, stepInterval);
      
      return () => clearInterval(timer);
    } else if (targetProgress === 100 && progress.status === 'completed') {
      // 扫描完成时直接设置为100%
      setSmoothProgress(100);
    }
  }, [progress.scanned_templates, progress.completed_templates, progress.total_templates, progress.status]);

  // 平滑更新显示数据
  useEffect(() => {
    const targetScanned = progress.scanned_templates || progress.completed_templates || 0;
    const targetVulns = progress.found_vulns || 0;
    
    // 只有数据增加时才更新显示
    setDisplayProgress(prev => ({
      scanned_templates: Math.max(prev.scanned_templates, targetScanned),
      completed_templates: Math.max(prev.completed_templates, targetScanned),
      found_vulns: Math.max(prev.found_vulns, targetVulns),
    }));
  }, [progress.scanned_templates, progress.completed_templates, progress.found_vulns]);

  return (
    <Card>
      {/* 中心进度圆环 */}
      <div style={{ textAlign: 'center' }}>
        <Progress
          type="circle"
          percent={Math.round(smoothProgress)}
          size={160}
          strokeColor={getStatusColor(progress.status)}
          strokeWidth={8}
          trailColor="#f0f0f0"
          format={() => (
            <div>
              <div style={{ fontSize: 28, fontWeight: 'bold', color: getStatusColor(progress.status) }}>
                {displayProgress.scanned_templates}/{progress.total_templates}
              </div>
              <div style={{ fontSize: 12, color: '#999', marginTop: 6 }}>
                {progress.status === 'running' ? '扫描中' : progress.status === 'completed' ? '已完成' : '等待中'}
              </div>
            </div>
          )}
        />
      </div>

      {/* 扫描中：显示当前进度 */}
      {progress.status === 'running' && (
        <div style={{ marginTop: 20, padding: '12px 16px', backgroundColor: '#e6f7ff', borderRadius: 6, border: '1px solid #91d5ff' }}>
          <div style={{ marginBottom: 8 }}>
            <SyncOutlined spin style={{ color: '#1890ff', marginRight: 6 }} />
            <Text strong style={{ fontSize: 14, color: '#1890ff' }}>
              正在扫描第 {progress.current_index || displayProgress.scanned_templates}/{progress.total_templates} 个POC
            </Text>
          </div>
          {progress.current_template && (
            <div style={{ fontSize: 12, color: '#666', marginBottom: 4 }}>
              <FileTextOutlined style={{ marginRight: 6 }} />
              {progress.current_template}
            </div>
          )}
          {progress.current_target && (
            <div style={{ fontSize: 12, color: '#666' }}>
              <AimOutlined style={{ marginRight: 6 }} />
              {progress.current_target}
            </div>
          )}
        </div>
      )}

      {/* 核心统计 - 3列简洁布局 */}
      <Row gutter={[12, 12]} style={{ marginTop: 20 }}>
        <Col span={8}>
          <div style={{ textAlign: 'center', padding: '10px', background: '#fafafa', borderRadius: 6 }}>
            <div style={{ fontSize: 11, color: '#999', marginBottom: 4 }}>已扫描</div>
            <div style={{ fontSize: 20, fontWeight: 'bold', color: '#1890ff' }}>
              {displayProgress.scanned_templates}
            </div>
          </div>
        </Col>
        <Col span={8}>
          <div style={{ textAlign: 'center', padding: '10px', background: displayProgress.found_vulns > 0 ? '#fff1f0' : '#f6ffed', borderRadius: 6 }}>
            <div style={{ fontSize: 11, color: '#999', marginBottom: 4 }}>发现漏洞</div>
            <div style={{ fontSize: 20, fontWeight: 'bold', color: displayProgress.found_vulns > 0 ? '#ff4d4f' : '#52c41a' }}>
              {displayProgress.found_vulns}
            </div>
          </div>
        </Col>
        <Col span={8}>
          <div style={{ textAlign: 'center', padding: '10px', background: '#fafafa', borderRadius: 6 }}>
            <div style={{ fontSize: 11, color: '#999', marginBottom: 4 }}>HTTP请求</div>
            <div style={{ fontSize: 20, fontWeight: 'bold', color: '#1890ff' }}>
              {progress.completed_requests || totalRequests || 0}
            </div>
          </div>
        </Col>
      </Row>

      {/* 扫描完成后显示详细统计 */}
      {progress.status === 'completed' && (
        <div style={{
          marginTop: 12,
          padding: '10px 12px',
          background: '#f6ffed',
          border: '1px solid #b7eb8f',
          borderRadius: 6,
          fontSize: 12
        }}>
          <Row gutter={12}>
            <Col span={8} style={{ textAlign: 'center' }}>
              <div style={{ color: '#999', marginBottom: 4 }}>被过滤</div>
              <Text strong style={{ fontSize: 16, color: (progress.filtered_templates || 0) > 0 ? '#faad14' : '#52c41a' }}>
                {progress.filtered_templates || 0}
              </Text>
            </Col>
            <Col span={8} style={{ textAlign: 'center' }}>
              <div style={{ color: '#999', marginBottom: 4 }}>被跳过</div>
              <Text strong style={{ fontSize: 16, color: (progress.skipped_templates || 0) > 0 ? '#8c8c8c' : '#52c41a' }}>
                {progress.skipped_templates || 0}
              </Text>
            </Col>
            <Col span={8} style={{ textAlign: 'center' }}>
              <div style={{ color: '#999', marginBottom: 4 }}>POC总数</div>
              <Text strong style={{ fontSize: 16, color: '#666' }}>
                {progress.total_templates}
              </Text>
            </Col>
          </Row>
        </div>
      )}
    </Card>
  );
};

// 使用 React.memo 优化性能，避免不必要的重渲染
export default React.memo(ScanProgressComponent, (prevProps, nextProps) => {
  // 只在关键数据变化时才重新渲染
  return (
    prevProps.progress.percentage === nextProps.progress.percentage &&
    prevProps.progress.scanned_templates === nextProps.progress.scanned_templates &&
    prevProps.progress.found_vulns === nextProps.progress.found_vulns &&
    prevProps.progress.status === nextProps.progress.status &&
    prevProps.httpLogs.length === nextProps.httpLogs.length
  );
});