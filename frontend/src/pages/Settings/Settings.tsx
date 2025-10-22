import { useState, useEffect } from 'react';
import { Card, Typography, Form, Input, InputNumber, Button, message, Space, Alert, Progress, Row, Col, Statistic, Divider, Tooltip, Tag } from 'antd';
import { SaveOutlined, FolderOpenOutlined, ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, WarningOutlined, InfoCircleOutlined, PlayCircleOutlined, FileOutlined, SettingOutlined, GithubOutlined, StarOutlined } from '@ant-design/icons';
import { Config } from '../../types';
import * as api from '../../services/api';

const { Title, Text, Paragraph } = Typography;

const Settings = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [config, setConfig] = useState<Config | null>(null);
  const [testingNuclei, setTestingNuclei] = useState(false);
  const [nucleiStatus, setNucleiStatus] = useState<{ valid: boolean; version: string } | null>(null);
  const [currentNucleiPath, setCurrentNucleiPath] = useState<string>('');
  
  // 导入进度状态
  const [importing, setImporting] = useState(false);
  const [importProgress, setImportProgress] = useState({
    total: 0,
    success: 0,
    error: 0,
    duplicate: 0,
    percent: 0,
    status: '准备中...'
  });
  const [importStartedFromSettings, setImportStartedFromSettings] = useState(false);

  useEffect(() => {
    loadConfig();
    
    // 监听模板导入进度事件
    const unsubscribe = api.onTemplateImportProgress((data: any) => {
      if (data && data.data) {
        // 映射后端数据结构到前端状态
        const progressData = {
          total: data.data.totalFound || data.data.total || 0,
          success: data.data.successful || 0,
          error: data.data.errors || 0,
          duplicate: data.data.duplicates || 0,
          percent: Math.round(data.data.percentage || 0),
          status: data.data.status || '准备中...'
        };
        
        setImportProgress(progressData);
        setImporting(true);
        
        // 如果导入完成，隐藏进度显示
        if (data.data.status === '导入完成!') {
          setTimeout(() => {
            setImporting(false);
            setImportStartedFromSettings(false);
            // 暂时禁用所有成功提示，避免重复显示
            // if (importStartedFromSettings) {
            //   message.success(`成功导入 ${data.data.successful || 0} 个模板`);
            // }
          }, 2000);
        }
      }
    });

    return () => {
      if (unsubscribe) {
        unsubscribe();
      }
    };
  }, []);

  const loadConfig = async () => {
    try {
      const cfg = await api.getConfig();
      setConfig(cfg);
      form.setFieldsValue(cfg);
      
      // 设置当前 nuclei 路径
      if (cfg.nuclei_path) {
        setCurrentNucleiPath(cfg.nuclei_path);
      }
      
      // 自动检查当前 nuclei 路径是否有效（跳过 Windows 以避免黑框）
      if (cfg.nuclei_path && navigator.platform.indexOf('Win') === -1) {
        try {
          const result = await api.testNucleiPath(cfg.nuclei_path);
          setNucleiStatus(result);
        } catch (error) {
          // 静默失败，不显示错误
          setNucleiStatus({ valid: false, version: '' });
        }
      } else if (cfg.nuclei_path) {
        // Windows 上不自动测试，避免黑框
        setNucleiStatus({ valid: true, version: '已配置' });
      }
    } catch (error: any) {
      message.error(`加载配置失败: ${error.message}`);
    }
  };

  const handleSave = async (values: Config) => {
    setLoading(true);
    try {
      await api.saveConfig(values);
      message.success('设置保存成功！');
      setConfig(values);
    } catch (error: any) {
      message.error(`保存失败: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectDirectory = async () => {
    try {
      const dir = await api.selectDirectory();
      if (dir) {
        form.setFieldsValue({ poc_directory: dir });
        message.success(`已选择目录: ${dir}`);
      }
    } catch (error: any) {
      message.error(`选择目录失败: ${error.message}`);
    }
  };

  const handleSelectNucleiDirectory = async () => {
    try {
      console.log('Opening directory selection dialog...');
      const nucleiPath = await api.selectNucleiDirectory();
      console.log('Nuclei path found:', nucleiPath);
      
      if (nucleiPath && nucleiPath.trim() !== '') {
        form.setFieldsValue({ nuclei_path: nucleiPath });
        setCurrentNucleiPath(nucleiPath);
        message.success(`已找到 Nuclei 可执行文件: ${nucleiPath}`);
        
        // 自动测试找到的文件
        setTimeout(() => {
          handleTestNucleiPath();
        }, 500);
      } else {
        message.info('未选择目录或未找到 nuclei 文件');
      }
    } catch (error: any) {
      console.error('Directory selection error:', error);
      
      // 提供更详细的错误信息和解决建议
      let errorMessage = `选择目录失败: ${error.message}`;
      
      if (error.message && error.message.includes('undefined')) {
        errorMessage = '目录选择对话框打开失败，请尝试手动输入路径';
      } else if (error.message && error.message.includes('未找到')) {
        errorMessage = '在选择的目录中未找到 nuclei 可执行文件，请确保目录包含 nuclei.exe 或 nuclei 文件';
      }
      
      message.error(errorMessage);
      
      // 显示手动输入的建议
      message.info('建议：可以手动输入 nuclei 路径，如 C:\\Users\\username\\go\\bin\\nuclei.exe');
    }
  };

  const handleImportTemplates = async () => {
    const pocDir = form.getFieldValue('poc_directory');
    if (!pocDir) {
      message.warning('请先设置 POC 目录');
      return;
    }

    setImportStartedFromSettings(true);
    setImporting(true);
    setImportProgress({
      total: 0,
      success: 0,
      error: 0,
      duplicate: 0,
      percent: 0,
      status: '准备中...'
    });
    
    try {
      // 调用真实的导入 API
      await api.importTemplates(pocDir);
    } catch (error: any) {
      message.error(`导入失败: ${error.message}`);
      setImporting(false);
      setImportStartedFromSettings(false);
    }
  };

  const handleTestNucleiPath = async () => {
    const nucleiPath = currentNucleiPath || form.getFieldValue('nuclei_path');
    if (!nucleiPath) {
      message.warning('请输入 Nuclei 路径');
      return;
    }

    setTestingNuclei(true);
    setNucleiStatus(null);
    
    try {
      const result = await api.testNucleiPath(nucleiPath);
      setNucleiStatus(result);
      
      if (result.valid) {
        message.success(`Nuclei 路径有效！版本: ${result.version}`);
        // 更新表单值
        form.setFieldsValue({ nuclei_path: nucleiPath });
      } else {
        message.error('Nuclei 路径无效或无法执行');
      }
    } catch (error: any) {
      console.error('Test nuclei path error:', error);
      
      // 提供更详细的错误信息
      let errorMessage = `测试失败: ${error.message}`;
      if (error.message && error.message.includes('not found')) {
        errorMessage = 'Nuclei 文件不存在，请检查路径是否正确';
      } else if (error.message && error.message.includes('not executable')) {
        errorMessage = 'Nuclei 文件不可执行，请检查文件权限';
      } else if (error.message && error.message.includes('not working')) {
        errorMessage = 'Nuclei 文件无法运行，请检查是否为有效的可执行文件';
      }
      
      message.error(errorMessage);
      setNucleiStatus({ valid: false, version: '' });
    } finally {
      setTestingNuclei(false);
    }
  };

  const handleSetNucleiPath = async () => {
    const nucleiPath = currentNucleiPath || form.getFieldValue('nuclei_path');
    if (!nucleiPath) {
      message.warning('请输入 Nuclei 路径');
      return;
    }

    try {
      console.log('Setting nuclei path:', nucleiPath);
      await api.setNucleiPath(nucleiPath);
      message.success('Nuclei 路径设置成功！');
      setNucleiStatus({ valid: true, version: '已设置' });
      
      // 重新加载配置以更新显示
      await loadConfig();
      
      // 重新加载应用配置以确保扫描器使用新路径
      try {
        await api.reloadConfig();
        console.log('Configuration reloaded successfully');
      } catch (reloadError: any) {
        console.warn('Failed to reload config:', reloadError);
        // Don't show error to user as the main operation succeeded
      }
    } catch (error: any) {
      console.error('Set nuclei path error:', error);
      message.error(`设置失败: ${error.message}`);
    }
  };

  return (
    <div style={{ padding: 0, backgroundColor: '#f5f5f5', minHeight: '100vh' }}>
      {/* 顶部导航栏 */}
      <div style={{
        position: 'sticky',
        top: 0,
        zIndex: 100,
        marginBottom: 16,
        padding: '16px 24px',
        backgroundColor: 'white',
        borderBottom: '1px solid #e8e8e8',
        boxShadow: '0 2px 8px rgba(0,0,0,0.06)',
      }}>
        <Space align="center">
          <InfoCircleOutlined style={{ fontSize: 20, color: '#1890ff' }} />
          <Title level={4} style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>系统设置</Title>
        </Space>
        <Text type="secondary" style={{ fontSize: 13 }}>配置 POC 模板路径、导入规则和系统参数</Text>
      </div>
      
      <div style={{ padding: '0 24px', maxWidth: 1200, margin: '0 auto' }}>
        {/* 模板导入进度卡片 */}
        {importing && (
          <Card 
            style={{ marginBottom: 16 }}
            styles={{ body: { padding: '24px' } }}
          >
            <Space direction="vertical" style={{ width: '100%' }} size="large">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Space>
                  <ReloadOutlined spin style={{ fontSize: 20, color: '#1890ff' }} />
                  <Title level={5} style={{ margin: 0 }}>正在导入 POC 模板</Title>
                </Space>
                <Tag color="processing">{importProgress.status}</Tag>
              </div>
              
              <Progress 
                percent={importProgress.percent} 
                status={importProgress.percent === 100 ? 'success' : 'active'}
                strokeColor={{
                  '0%': '#108ee9',
                  '100%': '#87d068',
                }}
              />
              
              <Row gutter={16}>
                <Col span={6}>
                  <Statistic 
                    title="模板总数" 
                    value={importProgress.total}
                    valueStyle={{ color: '#1890ff', fontSize: 24 }}
                  />
                </Col>
                <Col span={6}>
                  <Statistic 
                    title="上传成功" 
                    value={importProgress.success}
                    valueStyle={{ color: '#52c41a', fontSize: 24 }}
                    prefix={<CheckCircleOutlined />}
                  />
                </Col>
                <Col span={6}>
                  <Statistic 
                    title="解析错误" 
                    value={importProgress.error}
                    valueStyle={{ color: '#ff4d4f', fontSize: 24 }}
                    prefix={<CloseCircleOutlined />}
                  />
                </Col>
                <Col span={6}>
                  <Statistic 
                    title="重复模板" 
                    value={importProgress.duplicate}
                    valueStyle={{ color: '#faad14', fontSize: 24 }}
                    prefix={<WarningOutlined />}
                  />
                </Col>
              </Row>
            </Space>
          </Card>
        )}

        {/* 主配置卡片 */}
        <Card 
          title={
            <Space>
              <FolderOpenOutlined />
              <span>模板配置</span>
            </Space>
          }
          style={{ marginBottom: 16 }}
        >
          <Form
            form={form}
            layout="vertical"
            onFinish={handleSave}
            initialValues={config || undefined}
          >
            <Alert
              message="POC 模板目录设置"
              description="请选择包含 Nuclei 模板文件的目录，系统将递归扫描所有 .yaml 和 .yml 文件"
              type="info"
              showIcon
              style={{ marginBottom: 24 }}
            />

            <Form.Item
              label={
                <Space>
                  <Text strong>POC 模板目录</Text>
                  <Tooltip title="选择存放 Nuclei YAML 模板文件的根目录">
                    <InfoCircleOutlined style={{ color: '#8c8c8c' }} />
                  </Tooltip>
                </Space>
              }
              name="poc_directory"
              rules={[{ required: true, message: '请输入或选择 POC 目录' }]}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Input
                  placeholder="例如: /Users/xinux/tools/nuclei-templates"
                  style={{ width: 'calc(100% - 140px)' }}
                  size="large"
                />
                <Button
                  icon={<FolderOpenOutlined />}
                  onClick={handleSelectDirectory}
                  size="large"
                  style={{ width: 140 }}
                >
                  浏览目录
                </Button>
              </Space.Compact>
            </Form.Item>

            <Form.Item>
              <Button
                type="primary"
                icon={<ReloadOutlined />}
                onClick={handleImportTemplates}
                loading={importing}
                size="large"
                block
                style={{ height: 48 }}
              >
                {importing ? '正在导入模板...' : '从目录导入 POC 模板'}
              </Button>
              <Paragraph type="secondary" style={{ marginTop: 8, marginBottom: 0, fontSize: 12 }}>
                点击后将扫描指定目录下的所有 YAML 文件并导入到数据库
              </Paragraph>
            </Form.Item>
          </Form>
        </Card>

        {/* 系统配置卡片 */}
        <Card 
          title={
            <Space>
              <SaveOutlined />
              <span>系统配置</span>
            </Space>
          }
          style={{ marginBottom: 16 }}
        >
          <Form
            form={form}
            layout="vertical"
            onFinish={handleSave}
          >
            <Row gutter={16}>
              <Col xs={24} lg={12}>
                <Form.Item
                  label={<Text strong>扫描结果目录</Text>}
                  name="results_dir"
                  rules={[{ required: true, message: '请输入结果目录' }]}
                >
                  <Input placeholder="~/.wepoc/results" size="large" />
                </Form.Item>
              </Col>
              <Col xs={24} lg={12}>
                <Form.Item
                  label={<Text strong>数据库路径</Text>}
                  name="database_path"
                  rules={[{ required: true, message: '请输入数据库路径' }]}
                >
                  <Input placeholder="~/.wepoc/wepoc.db" size="large" />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              label={
                <Space>
                  <Text strong>Nuclei 可执行文件路径</Text>
                  <Tooltip title="设置 Nuclei 可执行文件的完整路径，或点击选择目录按钮自动查找">
                    <InfoCircleOutlined style={{ color: '#8c8c8c' }} />
                  </Tooltip>
                </Space>
              }
              name="nuclei_path"
              rules={[{ required: true, message: '请输入或选择 Nuclei 路径' }]}
            >
              <Space.Compact style={{ width: '100%' }}>
                <Input 
                  placeholder="/usr/local/bin/nuclei" 
                  size="large"
                  style={{ width: 'calc(100% - 300px)' }}
                  value={currentNucleiPath}
                  onChange={(e) => setCurrentNucleiPath(e.target.value)}
                />
                <Button
                  icon={<FileOutlined />}
                  onClick={handleSelectNucleiDirectory}
                  size="large"
                  style={{ width: 100 }}
                >
                  选择目录
                </Button>
                <Button
                  icon={<PlayCircleOutlined />}
                  onClick={handleTestNucleiPath}
                  loading={testingNuclei}
                  size="large"
                  style={{ width: 100 }}
                >
                  测试
                </Button>
                <Button
                  type="primary"
                  onClick={handleSetNucleiPath}
                  size="large"
                  style={{ width: 100 }}
                >
                  保存
                </Button>
              </Space.Compact>
            </Form.Item>

           

            {/* Nuclei 状态显示 */}
            {nucleiStatus && (
              <Alert
                message={
                  nucleiStatus.valid ? (
                    <Space direction="vertical" size="small" style={{ width: '100%' }}>
                      <Space>
                        <CheckCircleOutlined style={{ color: '#52c41a' }} />
                        <Text strong style={{ color: '#52c41a' }}>Nuclei 路径有效</Text>
                        {nucleiStatus.version && (
                          <Tag color="green">版本: {nucleiStatus.version}</Tag>
                        )}
                      </Space>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        当前路径: {currentNucleiPath}
                      </Text>
                    </Space>
                  ) : (
                    <Space direction="vertical" size="small" style={{ width: '100%' }}>
                      <Space>
                        <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                        <Text strong style={{ color: '#ff4d4f' }}>Nuclei 路径无效</Text>
                      </Space>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        请检查路径是否正确，或点击"浏览"按钮选择正确的可执行文件
                      </Text>
                    </Space>
                  )
                }
                type={nucleiStatus.valid ? 'success' : 'error'}
                style={{ marginBottom: 16 }}
                showIcon={false}
              />
            )}

            {/* 当前 nuclei 路径显示 */}
            {currentNucleiPath && !nucleiStatus && (
              <Alert
                message={
                  <Space>
                    <SettingOutlined style={{ color: '#1890ff' }} />
                    <Text>当前 Nuclei 路径: {currentNucleiPath}</Text>
                    <Tag color="blue">未验证</Tag>
                  </Space>
                }
                type="info"
                style={{ marginBottom: 16 }}
                showIcon={false}
                action={
                  <Button 
                    size="small" 
                    onClick={handleTestNucleiPath}
                    loading={testingNuclei}
                  >
                    验证路径
                  </Button>
                }
              />
            )}

            <Row gutter={16}>
              <Col xs={24} sm={12}>
                <Form.Item
                  label={
                    <Space>
                      <Text strong>最大并发任务数</Text>
                      <Tooltip title="同时运行的最大扫描任务数量">
                        <InfoCircleOutlined style={{ color: '#8c8c8c' }} />
                      </Tooltip>
                    </Space>
                  }
                  name="max_concurrency"
                  rules={[{ required: true, message: '请输入最大并发数' }]}
                >
                  <InputNumber 
                    min={1} 
                    max={10} 
                    style={{ width: '100%' }} 
                    size="large"
                    placeholder="1-10"
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12}>
                <Form.Item
                  label={
                    <Space>
                      <Text strong>超时时间 (秒)</Text>
                      <Tooltip title="单个扫描任务的最大执行时间">
                        <InfoCircleOutlined style={{ color: '#8c8c8c' }} />
                      </Tooltip>
                    </Space>
                  }
                  name="timeout"
                  rules={[{ required: true, message: '请输入超时时间' }]}
                >
                  <InputNumber 
                    min={1} 
                    max={300} 
                    style={{ width: '100%' }} 
                    size="large"
                    placeholder="1-300"
                  />
                </Form.Item>
              </Col>
            </Row>

            <Divider />

            <Form.Item style={{ marginBottom: 0 }}>
              <Space size="middle">
                <Button 
                  type="primary" 
                  htmlType="submit" 
                  icon={<SaveOutlined />} 
                  loading={loading}
                  size="large"
                >
                  保存设置
                </Button>
                <Button 
                  onClick={() => form.resetFields()} 
                  size="large"
                >
                  重置为默认值
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Card>

        {/* GitHub 信息卡片 */}
        <Card title="项目信息" style={{ marginTop: 16 }}>
          <div style={{ textAlign: 'center', padding: '20px 0' }}>
            <h2 style={{ margin: '0 0 8px 0', color: '#1890ff' }}>wepoc</h2>
            <p style={{ margin: '0 0 16px 0', color: '#666' }}>
              wepoc - Nuclei 漏洞扫描器图形界面工具
            </p>
            <div style={{ marginBottom: 16 }}>
              Github：https://github.com/cyber0s/wepoc
            </div>
            <div style={{ marginBottom: 16 }}>
              <Tag color="blue" style={{ marginRight: 8 }}>Go</Tag>
              <Tag color="green" style={{ marginRight: 8 }}>React</Tag>
              <Tag color="purple" style={{ marginRight: 8 }}>Wails</Tag>
              <Tag color="orange">Nuclei</Tag>
            </div>
            
            <div style={{ fontSize: '12px', color: '#999' }}>
              <p style={{ margin: '4px 0' }}>
                <strong>版本:</strong> 1.0.0
              </p>
              <p style={{ margin: '4px 0' }}>
                <strong>许可证:</strong> GPL-3.0
              </p>
              <p style={{ margin: '4px 0' }}>
                <strong>作者:</strong> cyber0s
              </p>
            </div>
          </div>
        </Card>
        
      </div>
    </div>
  );
};

export default Settings;