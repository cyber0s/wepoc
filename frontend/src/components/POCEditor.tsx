import React, { useState, useEffect } from 'react';
import { Modal, Input, InputNumber, Button, Form, Space, message, Spin, Alert, Collapse } from 'antd';
import { PlayCircleOutlined, SaveOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { GetPOCTemplateContent, TestSinglePOC, SavePOCTemplate } from '../../wailsjs/go/main/App';
import './POCEditor.css';

const { TextArea } = Input;
const { Panel } = Collapse;

interface POCEditorProps {
  visible: boolean;
  onClose: () => void;
  templateId: string;
  templatePath: string;
  defaultTarget?: string;
}

interface TestParams {
  template_content: string;
  target: string;
  concurrency: number;
  rate_limit: number;
  interactsh_url: string;
  interactsh_token: string;
  proxy_url: string;
}

interface TestResult {
  success: boolean;
  message: string;
  results_count: number;
  results: any[];
  raw_output: string;
  stderr?: string;
  error?: string;
  error_detail?: string;
  warning?: string;
}

const POCEditor: React.FC<POCEditorProps> = ({
  visible,
  onClose,
  templateId,
  templatePath,
  defaultTarget = ''
}) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [templateContent, setTemplateContent] = useState('');
  const [originalContent, setOriginalContent] = useState('');
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [hasChanges, setHasChanges] = useState(false);

  // Load POC template content
  useEffect(() => {
    if (visible && templatePath) {
      loadTemplate();
    }
  }, [visible, templatePath]);

  const loadTemplate = async () => {
    setLoading(true);
    try {
      const content = await GetPOCTemplateContent(templatePath);
      setTemplateContent(content);
      setOriginalContent(content);
      setHasChanges(false);
      setTestResult(null);
    } catch (error: any) {
      message.error(`加载模板失败: ${error.message || error}`);
    } finally {
      setLoading(false);
    }
  };

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newContent = e.target.value;
    setTemplateContent(newContent);
    setHasChanges(newContent !== originalContent);
  };

  const handleTest = async () => {
    try {
      await form.validateFields();
      const values = form.getFieldsValue();

      if (!templateContent.trim()) {
        message.warning('模板内容不能为空');
        return;
      }

      if (!values.target) {
        message.warning('请输入目标URL');
        return;
      }

      setTesting(true);
      setTestResult(null);

      const params: TestParams = {
        template_content: templateContent,
        target: values.target,
        concurrency: values.concurrency || 25,
        rate_limit: values.rate_limit || 150,
        interactsh_url: values.interactsh_url || '',
        interactsh_token: values.interactsh_token || '',
        proxy_url: values.proxy_url || '',
      };

      const result = await TestSinglePOC(params);
      setTestResult(result as TestResult);

      if (result.results_count > 0) {
        message.success(result.message);
      } else {
        message.info(result.message);
      }
    } catch (error: any) {
      message.error(`测试失败: ${error.message || error}`);
      setTestResult({
        success: false,
        message: error.message || error,
        results_count: 0,
        results: [],
        raw_output: ''
      });
    } finally {
      setTesting(false);
    }
  };

  const handleSave = async () => {
    if (!hasChanges) {
      message.info('没有修改，无需保存');
      return;
    }

    Modal.confirm({
      title: '确认保存',
      icon: <ExclamationCircleOutlined />,
      content: '保存将覆盖原模板文件，系统会自动创建备份。确定要保存吗？',
      okText: '确定保存',
      cancelText: '取消',
      onOk: async () => {
        setSaving(true);
        try {
          await SavePOCTemplate(templatePath, templateContent);
          message.success('保存成功！已自动创建备份文件');
          setOriginalContent(templateContent);
          setHasChanges(false);
        } catch (error: any) {
          message.error(`保存失败: ${error.message || error}`);
        } finally {
          setSaving(false);
        }
      }
    });
  };

  const handleClose = () => {
    if (hasChanges) {
      Modal.confirm({
        title: '未保存的修改',
        icon: <ExclamationCircleOutlined />,
        content: '您有未保存的修改，确定要关闭吗？',
        okText: '确定关闭',
        okType: 'danger',
        cancelText: '取消',
        onOk: () => {
          onClose();
        }
      });
    } else {
      onClose();
    }
  };

  return (
    <Modal
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span>POC 编辑器 - {templateId}</span>
          {hasChanges && <span style={{ color: '#ff4d4f', fontSize: 12, marginLeft: 16 }}>● 未保存</span>}
        </div>
      }
      open={visible}
      onCancel={handleClose}
      width="90%"
      style={{ top: 20 }}
      footer={null}
      maskClosable={false}
    >
      <Spin spinning={loading} tip="加载中...">
        <div className="poc-editor-container">
          <div className="poc-editor-left">
            <div className="poc-editor-header">
              <span>模板内容 (YAML)</span>
              <Space>
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  onClick={handleSave}
                  loading={saving}
                  disabled={!hasChanges}
                >
                  保存
                </Button>
              </Space>
            </div>
            <TextArea
              value={templateContent}
              onChange={handleContentChange}
              placeholder="在此编辑 POC 模板内容..."
              className="poc-editor-textarea"
              spellCheck={false}
            />
          </div>

          <div className="poc-editor-right">
            <div className="poc-editor-settings">
              <h3>扫描参数设置</h3>
              <Form
                form={form}
                layout="vertical"
                initialValues={{
                  target: defaultTarget,
                  concurrency: 25,
                  rate_limit: 150,
                  interactsh_url: '',
                  interactsh_token: '',
                  proxy_url: ''
                }}
              >
                <Form.Item
                  label="目标URL"
                  name="target"
                  rules={[{ required: true, message: '请输入目标URL' }]}
                >
                  <Input placeholder="https://example.com" />
                </Form.Item>

                <Form.Item
                  label="并发数 (Concurrency)"
                  name="concurrency"
                  tooltip="同时运行的模板数量"
                >
                  <InputNumber min={1} max={100} style={{ width: '100%' }} />
                </Form.Item>

                <Form.Item
                  label="扫描速率 (Rate Limit)"
                  name="rate_limit"
                  tooltip="每秒最大请求数"
                >
                  <InputNumber min={1} max={500} style={{ width: '100%' }} />
                </Form.Item>

                <Form.Item
                  label="Interactsh URL"
                  name="interactsh_url"
                  tooltip="自定义 Interactsh 服务器地址"
                >
                  <Input placeholder="https://interactsh.com" />
                </Form.Item>

                <Form.Item
                  label="Interactsh Token"
                  name="interactsh_token"
                  tooltip="Interactsh 服务器认证令牌"
                >
                  <Input.Password placeholder="输入令牌" />
                </Form.Item>

                <Form.Item
                  label="网络代理"
                  name="proxy_url"
                  tooltip="HTTP/HTTPS/SOCKS5 代理地址"
                >
                  <Input placeholder="http://127.0.0.1:8080" />
                </Form.Item>

                <Form.Item>
                  <Button
                    type="primary"
                    icon={<PlayCircleOutlined />}
                    onClick={handleTest}
                    loading={testing}
                    block
                    size="large"
                  >
                    测试 POC
                  </Button>
                </Form.Item>
              </Form>
            </div>

            {testResult && (
              <div className="poc-test-result">
                <Alert
                  message={testResult.message}
                  type={testResult.success === false ? 'error' : (testResult.results_count > 0 ? 'success' : 'info')}
                  description={testResult.error_detail || testResult.warning}
                  showIcon
                  style={{ marginBottom: 12 }}
                />

                {testResult.results_count > 0 && (
                  <Collapse defaultActiveKey={['1']}>
                    <Panel header={`漏洞结果 (${testResult.results_count})`} key="1">
                      <div style={{ maxHeight: 300, overflow: 'auto' }}>
                        {testResult.results.map((result, idx) => (
                          <div key={idx} className="vulnerability-item">
                            <div><strong>模板:</strong> {result['template-id'] || result.templateID}</div>
                            <div><strong>主机:</strong> {result.host}</div>
                            {result['matched-at'] && (
                              <div><strong>匹配位置:</strong> {result['matched-at']}</div>
                            )}
                            {result.info && (
                              <>
                                <div><strong>严重程度:</strong> {result.info.severity}</div>
                                <div><strong>描述:</strong> {result.info.name}</div>
                              </>
                            )}
                          </div>
                        ))}
                      </div>
                    </Panel>
                    {testResult.raw_output && (
                      <Panel header="原始输出" key="2">
                        <pre className="raw-output">{testResult.raw_output}</pre>
                      </Panel>
                    )}
                  </Collapse>
                )}

                {testResult.results_count === 0 && (testResult.raw_output || testResult.stderr) && (
                  <Collapse>
                    {testResult.raw_output && (
                      <Panel header="标准输出 (stdout)" key="1">
                        <pre className="raw-output">{testResult.raw_output}</pre>
                      </Panel>
                    )}
                    {testResult.stderr && (
                      <Panel header="错误输出 (stderr)" key="2">
                        <pre className="raw-output">{testResult.stderr}</pre>
                      </Panel>
                    )}
                  </Collapse>
                )}
              </div>
            )}
          </div>
        </div>
      </Spin>
    </Modal>
  );
};

export default POCEditor;
