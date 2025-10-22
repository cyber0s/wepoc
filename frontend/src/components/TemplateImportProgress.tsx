import React from 'react';
import { Progress, Row, Col, Statistic } from 'antd';

interface ImportProgressData {
  current: number;
  total: number;
  percentage: number;
  status: string;
  totalFound: number;
  imported: number;
  successful: number;
  errors: number;
  duplicates: number;
}

interface TemplateImportProgressProps {
  visible: boolean;
  progressData: ImportProgressData | null;
}

const TemplateImportProgress: React.FC<TemplateImportProgressProps> = ({
  visible,
  progressData,
}) => {
  if (!visible || !progressData) {
    return null;
  }

  return (
    <div style={{
      padding: '20px',
      backgroundColor: '#fafafa',
      borderRadius: '8px',
      border: '1px solid #e8e8e8',
      margin: '16px 0'
    }}>
      <Row gutter={24} align="middle">
        {/* 进度圆环 */}
        <Col span={8} style={{ textAlign: 'center' }}>
          <Progress
            type="circle"
            percent={Math.round(progressData.percentage)}
            size={120}
            strokeColor={{
              '0%': '#108ee9',
              '100%': '#87d068',
            }}
            format={() => (
              <div style={{ fontSize: '16px', fontWeight: 'bold' }}>
                {Math.round(progressData.percentage)}%
              </div>
            )}
          />
          <div style={{ 
            marginTop: '8px', 
            fontSize: '14px', 
            color: progressData.status === '导入完成!' ? '#52c41a' : '#1890ff',
            fontWeight: '500'
          }}>
            {progressData.status}
          </div>
        </Col>

        {/* 统计信息 */}
        <Col span={16}>
          <Row gutter={16}>
            <Col span={6}>
              <Statistic
                title="模版总数"
                value={progressData.totalFound}
                valueStyle={{ color: '#1890ff' }}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="上传成功"
                value={progressData.successful}
                valueStyle={{ color: '#52c41a' }}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="解析错误"
                value={progressData.errors}
                valueStyle={{ color: '#ff4d4f' }}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="重复模版"
                value={progressData.duplicates}
                valueStyle={{ color: '#faad14' }}
              />
            </Col>
          </Row>
        </Col>
      </Row>
    </div>
  );
};

export default TemplateImportProgress;
