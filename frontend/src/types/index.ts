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

export interface Config {
  poc_directory: string;
  results_dir: string;
  database_path: string;
  nuclei_path: string;
  max_concurrency: number;
  timeout: number;
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
  event_type: 'progress' | 'log' | 'vuln_found' | 'completed' | 'error';
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
}

