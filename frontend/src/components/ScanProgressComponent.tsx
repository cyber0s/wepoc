import { Progress, Statistic, Row, Col, Card } from 'antd';
import { CheckCircleOutlined, BugOutlined, SyncOutlined } from '@ant-design/icons';
import { ScanProgress } from '../types';

interface ScanProgressProps {
  progress: ScanProgress;
}

const ScanProgressComponent: React.FC<ScanProgressProps> = ({ progress }) => {
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

  // Add animation to the progress display
  const getProgressPercent = () => {
    // Ensure the percentage is between 0 and 100
    const percent = Math.max(0, Math.min(100, Math.round(progress.percentage)));
    return percent;
  };

  return (
    <Card>
      <div style={{ textAlign: 'center' }}>
        <Progress
          type="circle"
          percent={getProgressPercent()}
          size={180}
          strokeColor={getStatusColor(progress.status)}
          format={() => (
            <div>
              <div style={{ fontSize: 32, fontWeight: 'bold' }}>
                {progress.completed_requests}/{progress.total_requests}
              </div>
              <div style={{ fontSize: 14, color: '#666', marginTop: 8 }}>
                {getStatusText(progress.status)}
              </div>
            </div>
          )}
        />
      </div>

      <Row gutter={16} style={{ marginTop: 24 }}>
        <Col span={12}>
          <Statistic
            title="已完成请求"
            value={progress.completed_requests}
            suffix={`/ ${progress.total_requests}`}
            prefix={<SyncOutlined spin={progress.status === 'running'} />}
          />
        </Col>
        <Col span={12}>
          <Statistic
            title="发现漏洞"
            value={progress.found_vulns}
            valueStyle={{ color: progress.found_vulns > 0 ? '#ff4d4f' : '#666' }}
            prefix={<BugOutlined />}
          />
        </Col>
      </Row>

      <Row gutter={16} style={{ marginTop: 16 }}>
        <Col span={12}>
          <Statistic
            title="完成度"
            value={progress.percentage}
            precision={1}
            suffix="%"
          />
        </Col>
        <Col span={12}>
          <Statistic
            title="状态"
            value={getStatusText(progress.status)}
            prefix={
              progress.status === 'completed' ? (
                <CheckCircleOutlined style={{ color: '#52c41a' }} />
              ) : progress.status === 'running' ? (
                <SyncOutlined spin style={{ color: '#1890ff' }} />
              ) : null
            }
          />
        </Col>
      </Row>
    </Card>
  );
};

export default ScanProgressComponent;