export interface Template {
  id: number;
  template_id: string;
  name: string;
  severity: string;
  tags: string;
  author: string;
  file_path: string;
  created_at: string;
}

export interface ScanTask {
  id: number;
  name: string;
  status: 'pending' | 'running' | 'paused' | 'stopped' | 'completed' | 'failed';
  pocs: string; // JSON string
  targets: string; // JSON string
  total_requests: number;
  completed_requests: number;
  found_vulns: number;
  output_file: string;
  start_time: string;
  end_time?: string;
  created_at: string;
  error?: string;
}

export interface TaskProgress {
  task_id: number;
  total_requests: number;
  completed_requests: number;
  found_vulns: number;
  percentage: number;
  current_template: string;
  current_target?: string;
  status: string;
}

export interface TaskEvent {
  task_id: number;
  event_type: 'progress' | 'log' | 'completed' | 'error';
  data: any;
  timestamp: string;
}

export interface NucleiResult {
  'template-id': string;
  'template-path': string;
  info: {
    name: string;
    author: string[];
    tags: string[];
    description?: string;
    severity: string;
  };
  type: string;
  host: string;
  'matched-at': string;
  'extracted-results'?: string[];
  request?: string;
  response?: string;
  timestamp: string;
  'curl-command'?: string;
}

export interface NucleiAdvancedConfig {
  // 线程配置
  concurrency: number;
  bulk_size: number;
  rate_limit: number;
  rate_limit_minute: number;
  
  // 代理配置
  proxy_enabled: boolean;
  proxy_url: string;
  proxy_list: string[];
  proxy_internal: boolean;
  
  // DNS/OAST配置
  interactsh_enabled: boolean;
  interactsh_server: string;
  interactsh_token: string;
  interactsh_disable: boolean;
  
  // 其他选项
  retries: number;
  max_host_error: number;
  disable_update_check: boolean;
  follow_redirects: boolean;
  max_redirects: number;
}

export interface Config {
  poc_directory: string;
  results_dir: string;
  database_path: string;
  nuclei_path: string;
  max_concurrency: number;
  timeout: number;
  nuclei_config: NucleiAdvancedConfig;
}

export interface ScanLog {
  task_id: number;
  timestamp: string;
  level: 'INFO' | 'WARNING' | 'ERROR' | 'VULN';
  template_id: string;
  target: string;
  message: string;
  is_vuln_found: boolean;
}

// New types for JSON-based task management

export interface ScanEvent {
  task_id: number;
  event_type: 'progress' | 'log' | 'vuln_found' | 'completed' | 'error' | 'http';
  data: any;
  timestamp: string;
}

export interface ScanProgress {
  task_id: number;
  total_requests: number;
  completed_requests: number;
  found_vulns: number;
  percentage: number;
  status: string;
  current_template: string;
  current_target: string;
  total_templates: number;
  completed_templates: number;
  scanned_templates: number;
  failed_templates: number;
  filtered_templates?: number; // 被Nuclei过滤的模板数量
  skipped_templates?: number;  // 被跳过的模板数量
  current_index: number;
  selected_templates: string[];
  scanned_template_ids: string[];
  failed_template_ids: string[];
  filtered_template_ids?: string[]; // 被过滤的模板ID列表
  skipped_template_ids?: string[];  // 被跳过的模板ID列表
}

export interface ScanLogEntry {
  timestamp: string;
  level: 'INFO' | 'WARN' | 'ERROR' | 'DEBUG';
  template_id?: string;
  target?: string;
  message: string;
  request?: string;
  response?: string;
  is_vuln_found: boolean;
}

export interface TaskConfig {
  id: number;
  name: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  pocs: string[];
  targets: string[];
  total_requests: number;
  completed_requests: number;
  found_vulns: number;
  start_time: string;
  end_time?: string;
  output_file: string;
  log_file: string;
  created_at: string;
  updated_at: string;
}

export interface TaskResult {
  task_id: number;
  task_name: string;
  status: string;
  start_time: string;
  end_time: string;
  duration: string;
  targets: string[];
  templates: string[];
  template_count: number;
  target_count: number;
  total_requests: number;
  completed_requests: number;
  found_vulns: number;
  success_rate: number;
  vulnerabilities: NucleiResult[];
  summary: Record<string, any>;
  created_at: string;

  // 新增：详细统计信息
  scanned_templates?: number;       // 实际扫描的模板数量
  filtered_templates?: number;      // 被Nuclei过滤的模板数量
  skipped_templates?: number;       // 被跳过的模板数量
  failed_templates?: number;        // 扫描失败的模板数量
  filtered_template_ids?: string[]; // 被过滤的模板ID列表
  skipped_template_ids?: string[];  // 被跳过的模板ID列表
  failed_template_ids?: string[];   // 失败的模板ID列表
  scanned_template_ids?: string[];  // 已扫描的模板ID列表
  http_requests?: number;           // 实际HTTP请求数量
}

// HTTP请求日志 - 用于前端表格展示每个HTTP请求
export interface HTTPRequestLog {
  id: number;                   // 请求序号
  task_id: number;              // 所属任务ID
  timestamp: string;            // 请求时间
  template_id: string;          // POC模板ID
  template_name: string;        // POC模板名称
  severity: string;             // 严重程度
  target: string;               // 目标URL
  method: string;               // HTTP方法（GET/POST等）
  status_code: number;          // HTTP状态码
  is_vuln_found: boolean;       // 是否发现漏洞
  request: string;              // 完整请求包
  response: string;             // 完整响应包
  duration_ms: number;          // 请求耗时（毫秒）
}

