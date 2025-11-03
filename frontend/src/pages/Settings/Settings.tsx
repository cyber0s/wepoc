import { useState, useEffect } from 'react';
import { Card, Typography, Form, Input, InputNumber, Button, message, Space, Alert, Progress, Row, Col, Statistic, Divider, Tooltip, Tag, Switch } from 'antd';
import { SaveOutlined, FolderOpenOutlined, ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, WarningOutlined, InfoCircleOutlined, PlayCircleOutlined, FileOutlined, SettingOutlined, GithubOutlined, StarOutlined } from '@ant-design/icons';
import { Config } from '../../types';
import { api } from '../../services/api';

const { Title, Text, Paragraph } = Typography;

const Settings = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [config, setConfig] = useState<Config | null>(null);
  const [testingNuclei, setTestingNuclei] = useState(false);
  const [nucleiStatus, setNucleiStatus] = useState<{ valid: boolean; version: string } | null>(null);
  const [currentNucleiPath, setCurrentNucleiPath] = useState<string>('');
  
  // 代理测试状态
  const [testingProxies, setTestingProxies] = useState(false);
  const [proxyTestResults, setProxyTestResults] = useState<any>(null);
  
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
      
      // 确保代理配置有默认值，代理默认关闭
      const configWithDefaults = {
        ...cfg,
        nuclei_config: {
          ...cfg.nuclei_config,
          proxy_enabled: cfg.nuclei_config?.proxy_enabled || false,
          proxy_internal: cfg.nuclei_config?.proxy_internal || false,
          proxy_url: cfg.nuclei_config?.proxy_url || '',
          proxy_list: cfg.nuclei_config?.proxy_list || []
        }
      };
      
      form.setFieldsValue(configWithDefaults);
      
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

  const handleTestProxies = async () => {
    const proxyUrl = form.getFieldValue(['nuclei_config', 'proxy_url']);
    const proxyListText = form.getFieldValue(['nuclei_config', 'proxy_list']);
    
    // 构建代理列表
    let proxyList: string[] = [];
    
    if (proxyUrl && proxyUrl.trim()) {
      proxyList.push(proxyUrl.trim());
    }
    
    if (proxyListText && proxyListText.trim()) {
      const listProxies = proxyListText.split('\n')
        .map((line: string) => line.trim())
        .filter((line: string) => line.length > 0);
      proxyList = proxyList.concat(listProxies);
    }
    
    if (proxyList.length === 0) {
      message.warning('请先配置代理服务器');
      return;
    }

    setTestingProxies(true);
    setProxyTestResults(null);
    
    try {
      const results = await api.testProxies(proxyList);
      setProxyTestResults(results);
      
      if (results && results.summary) {
        const { available, failed, total } = results.summary;
        if (available > 0) {
          message.success(`代理测试完成：${available}/${total} 个代理可用`);
        } else {
          message.error(`代理测试失败：所有 ${total} 个代理都不可用`);
        }
      }
    } catch (error: any) {
      message.error(`代理测试失败: ${error.message}`);
      setProxyTestResults(null);
    } finally {
      setTestingProxies(false);
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
    <div style={{ padding: 0, backgroundColor: '#f0f2f5', minHeight: '100vh' }}>
      {/* 顶部导航栏 */}
      <div style={{
        position: 'sticky',
        top: 0,
        zIndex: 100,
        marginBottom: 16,
        padding: '12px 20px',
        backgroundColor: 'white',
        borderBottom: '1px solid #e8e8e8',
        boxShadow: '0 1px 4px rgba(0,0,0,0.05)',
      }}>
        <Space align="center">
          <InfoCircleOutlined style={{ fontSize: 18, color: '#1890ff' }} />
          <Title level={4} style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>系统设置</Title>
        </Space>
        <Text type="secondary" style={{ fontSize: 12 }}>配置 POC 模板路径、导入规则和系统参数</Text>
      </div>
      
      <div style={{ 
        padding: '0 24px', 
        maxWidth: 1400, 
        margin: '0 auto',
        height: 'calc(100vh - 80px)',
        overflowY: 'auto'
      }}>
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

        {/* 主要布局：左侧设置区域 + 右侧项目信息面板 */}
        <Row gutter={[24, 16]}>
          {/* 左侧主要设置区域 */}
          <Col xs={24} lg={18}>
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

            {/* 代理配置部分 */}
            <Title level={5} style={{ marginBottom: 16 }}>
              <Space>
                <SettingOutlined />
                <span>代理配置</span>
              </Space>
            </Title>

            <Row gutter={16}>
              <Col xs={24} sm={8}>
                <Form.Item
                  label={<Text strong>启用代理</Text>}
                  name={['nuclei_config', 'proxy_enabled']}
                  valuePropName="checked"
                >
                  <Switch 
                    checkedChildren="开启" 
                    unCheckedChildren="关闭"
                    size="default"
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={8}>
                <Form.Item
                  label={<Text strong>代理内部请求</Text>}
                  name={['nuclei_config', 'proxy_internal']}
                  valuePropName="checked"
                >
                  <Switch 
                    checkedChildren="是" 
                    unCheckedChildren="否"
                    size="default"
                  />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              label={
                <Space>
                  <Text strong>代理服务器 URL</Text>
                  <Tooltip title="支持 HTTP/HTTPS/SOCKS5 代理，格式：http://proxy:port 或 socks5://proxy:port">
                    <InfoCircleOutlined style={{ color: '#8c8c8c' }} />
                  </Tooltip>
                </Space>
              }
              name={['nuclei_config', 'proxy_url']}
            >
              <Input 
                placeholder="http://127.0.0.1:8080 或 socks5://127.0.0.1:1080" 
                size="large"
              />
            </Form.Item>

            <Form.Item
              label={
                <Space>
                  <Text strong>代理服务器列表</Text>
                  <Tooltip title="多个代理地址，每行一个。系统会自动轮换使用">
                    <InfoCircleOutlined style={{ color: '#8c8c8c' }} />
                  </Tooltip>
                </Space>
              }
              name={['nuclei_config', 'proxy_list']}
            >
              <Input.TextArea 
                placeholder="http://127.0.0.1:8080&#10;http://127.0.0.1:8081&#10;socks5://127.0.0.1:1080"
                rows={4}
                size="large"
              />
            </Form.Item>

            {/* 代理测试按钮 */}
            <Form.Item>
              <Button
                type="default"
                icon={<PlayCircleOutlined />}
                onClick={handleTestProxies}
                loading={testingProxies}
                size="large"
              >
                {testingProxies ? '正在测试代理...' : '测试代理连接'}
              </Button>
            </Form.Item>

            {/* 代理测试结果显示 */}
            {proxyTestResults && (
              <Alert
                message={
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <Space>
                      <Text strong>代理测试结果</Text>
                      <Tag color={proxyTestResults.summary.available > 0 ? 'green' : 'red'}>
                        {proxyTestResults.summary.available}/{proxyTestResults.summary.total} 可用
                      </Tag>
                    </Space>
                    <div style={{ maxHeight: 200, overflowY: 'auto' }}>
                      {proxyTestResults.results.map((result: any, index: number) => (
                        <div key={index} style={{ marginBottom: 8, fontSize: 12 }}>
                          <Space>
                            {result.available ? (
                              <CheckCircleOutlined style={{ color: '#52c41a' }} />
                            ) : (
                              <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                            )}
                            <Text code style={{ fontSize: 11 }}>{result.url}</Text>
                            {result.available && (
                              <Tag color="blue" style={{ fontSize: 10 }}>
                                {result.response_time}ms
                              </Tag>
                            )}
                            {result.error && (
                              <Text type="secondary" style={{ fontSize: 10 }}>
                                {result.error}
                              </Text>
                            )}
                          </Space>
                        </div>
                      ))}
                    </div>
                  </Space>
                }
                type={proxyTestResults.summary.available > 0 ? 'success' : 'error'}
                style={{ marginBottom: 16 }}
                showIcon={false}
              />
            )}

            <Divider />

            {/* DNS外带 (Interactsh) 配置部分 */}
            <Title level={5} style={{ marginBottom: 16 }}>
              <Space>
                <SettingOutlined />
                <span>DNS 外带 (Interactsh) 配置</span>
              </Space>
            </Title>

            <Alert
              message="DNS外带检测功能说明"
              description="Interactsh 是 Nuclei 的带外数据通信服务，用于检测盲注等无法直接获得响应的漏洞。默认使用 ProjectDiscovery 的公共服务器，也可以配置私有服务器。"
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <Row gutter={16}>
              <Col xs={24} sm={8}>
                <Form.Item
                  label={
                    <Space>
                      <Text strong>启用 Interactsh</Text>
                      <Tooltip title="开启 DNS 外带检测功能，用于检测盲注漏洞">
                        <InfoCircleOutlined style={{ color: '#8c8c8c' }} />
                      </Tooltip>
                    </Space>
                  }
                  name={['nuclei_config', 'interactsh_enabled']}
                  valuePropName="checked"
                >
                  <Switch
                    checkedChildren="开启"
                    unCheckedChildren="关闭"
                    size="default"
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={8}>
                <Form.Item
                  label={
                    <Space>
                      <Text strong>完全禁用 Interactsh</Text>
                      <Tooltip title="完全禁用 Interactsh 功能，不使用任何外带服务（包括默认服务）">
                        <InfoCircleOutlined style={{ color: '#8c8c8c' }} />
                      </Tooltip>
                    </Space>
                  }
                  name={['nuclei_config', 'interactsh_disable']}
                  valuePropName="checked"
                >
                  <Switch
                    checkedChildren="禁用"
                    unCheckedChildren="允许"
                    size="default"
                  />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              label={
                <Space>
                  <Text strong>自定义 Interactsh 服务器</Text>
                  <Tooltip title="留空则使用 ProjectDiscovery 的默认服务器。如需隐私保护，可部署私有服务器">
                    <InfoCircleOutlined style={{ color: '#8c8c8c' }} />
                  </Tooltip>
                </Space>
              }
              name={['nuclei_config', 'interactsh_server']}
            >
              <Input
                placeholder="https://interact.projectdiscovery.io 或自建服务器地址"
                size="large"
              />
            </Form.Item>

            <Form.Item
              label={
                <Space>
                  <Text strong>Interactsh 认证 Token</Text>
                  <Tooltip title="如果使用需要认证的私有 Interactsh 服务器，请填入认证 Token">
                    <InfoCircleOutlined style={{ color: '#8c8c8c' }} />
                  </Tooltip>
                </Space>
              }
              name={['nuclei_config', 'interactsh_token']}
            >
              <Input.Password
                placeholder="认证 Token（可选）"
                size="large"
              />
            </Form.Item>

            <Alert
              message={
                <Space direction="vertical" size="small" style={{ width: '100%' }}>
                  <Text strong>使用建议</Text>
                  <ul style={{ margin: 0, paddingLeft: 20, fontSize: 12 }}>
                    <li><Text type="secondary">默认情况下，Nuclei 会自动使用 ProjectDiscovery 的公共 Interactsh 服务</Text></li>
                    <li><Text type="secondary">开启"启用 Interactsh"并配置自定义服务器，可使用私有部署</Text></li>
                    <li><Text type="secondary">如需完全禁用（某些环境禁止外联），勾选"完全禁用 Interactsh"</Text></li>
                    <li><Text type="secondary">私有服务器部署教程：<a href="https://github.com/projectdiscovery/interactsh" target="_blank" rel="noopener noreferrer">github.com/projectdiscovery/interactsh</a></Text></li>
                  </ul>
                </Space>
              }
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
            />

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
            </Col>
          
          {/* 右侧项目信息面板 */}
          <Col xs={24} lg={6}>
            <div style={{ position: 'sticky', top: 80 }}>
              <Card 
                title={
                  <Space>
                    <GithubOutlined />
                    <span>项目信息</span>
                  </Space>
                }
                style={{ borderRadius: 8 }}
              >
                <div style={{ textAlign: 'center', padding: '16px 0' }}>
                  <div style={{ marginBottom: 16 }}>
                    <h3 style={{ margin: '0 0 8px 0', color: '#1890ff', fontSize: 18 }}>WePOC</h3>
                    <Text type="secondary" style={{ fontSize: 13 }}>
                      Nuclei 漏洞扫描器图形界面工具
                    </Text>
                  </div>
                  
                  <div style={{ marginBottom: 16 }}>
                    <Space wrap size="small">
                      <Tag color="blue" icon={<StarOutlined />}>Go</Tag>
                      <Tag color="green" icon={<StarOutlined />}>React</Tag>
                      <Tag color="purple" icon={<StarOutlined />}>Wails</Tag>
                      <Tag color="orange" icon={<StarOutlined />}>Nuclei</Tag>
                    </Space>
                  </div>
                  
                  <Divider style={{ margin: '16px 0' }} />
                  
                  <div style={{ textAlign: 'left' }}>
                    <div style={{ marginBottom: 8, fontSize: 12 }}>
                      <Text strong>版本：</Text>
                <Text type="secondary">1.3</Text>
                    </div>
                    <div style={{ marginBottom: 8, fontSize: 12 }}>
                      <Text strong>许可证：</Text>
                      <Text type="secondary">GPL-3.0</Text>
                    </div>
                    <div style={{ marginBottom: 8, fontSize: 12 }}>
                      <Text strong>作者：</Text>
                      <Text type="secondary">cyber0s</Text>
                    </div>
                    <div style={{ fontSize: 12 }}>
                      <Text strong>Github：</Text>
                      <br />
                      <Text 
                        type="secondary" 
                        style={{ 
                          fontSize: 11, 
                          wordBreak: 'break-all',
                          cursor: 'pointer',
                          color: '#1890ff'
                        }}
                        onClick={() => window.open('https://github.com/cyber0s/wepoc', '_blank')}
                      >
                        github.com/cyber0s/wepoc
                      </Text>
                    </div>
                  </div>
                </div>
              </Card>
            </div>
          </Col>
        </Row>
        
      </div>
    </div>
  );
};

export default Settings;