// 导入Wails生成的绑定
import * as App from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

// 使用any类型来绕过TypeScript检查
const AppAny = App as any;

// 检查是否存在 Wails 运行时
const isWailsReady = () => {
  try {
    return typeof window !== 'undefined' && !!(window as any).go && !!(window as any).runtime;
  } catch {
    return false;
  }
};

// 直接导出所有API方法（在无 Wails 环境下提供安全兜底）
export const api = {
  // 配置相关
  getConfig: async () => {
    if (!isWailsReady()) return {} as any;
    return AppAny.GetConfig();
  },
  saveConfig: async (cfg: any) => {
    if (!isWailsReady()) return;
    return AppAny.SaveConfig(cfg);
  },
  reloadConfig: async () => {
    if (!isWailsReady()) return;
    return AppAny.ReloadConfig();
  },

  // 模板相关
  getAllTemplates: async () => {
    if (!isWailsReady()) return [];
    return AppAny.GetAllTemplates();
  },
  searchTemplates: async (kw: string, severity: string) => {
    if (!isWailsReady()) return [];
    return AppAny.SearchTemplates(kw, severity);
  },
  importTemplates: async (dir: string) => {
    if (!isWailsReady()) return { templates: [], errors: [], successful: 0 } as any;
    return AppAny.ImportTemplates(dir);
  },
  preValidateTemplates: async (dir: string) => {
    if (!isWailsReady()) return { templates: [], errors: [], successful: 0 } as any;
    return AppAny.PreValidateTemplates(dir);
  },
  confirmAndImportTemplates: async (templates: any[]) => {
    if (!isWailsReady()) return { successful: 0, errors: [] } as any;
    return AppAny.ConfirmAndImportTemplates(templates);
  },
  deleteTemplate: async (id: string) => {
    if (!isWailsReady()) return;
    return AppAny.DeleteTemplate(id);
  },
  clearAllTemplates: async () => {
    if (!isWailsReady()) return;
    return AppAny.ClearAllTemplates();
  },

  // 扫描任务相关
  createScanTask: async (name: string, pocsJson: string, targetsJson: string) => {
    if (!isWailsReady()) return null;
    return AppAny.CreateScanTask(name, pocsJson, targetsJson);
  },
  startScanTask: async (id: number) => {
    if (!isWailsReady()) return;
    return AppAny.StartScanTask(id);
  },
  stopScanTask: async (id: number) => {
    if (!isWailsReady()) return;
    return AppAny.StopScanTask(id);
  },
  pauseScanTask: async (id: number) => {
    if (!isWailsReady()) return;
    return AppAny.PauseScanTask(id);
  },
  rescanTask: async (id: number) => {
    if (!isWailsReady()) return;
    return AppAny.RescanTask(id);
  },
  getAllScanTasks: async () => {
    if (!isWailsReady()) return [];
    return AppAny.GetAllScanTasks();
  },
  getRunningScanTasks: async () => {
    if (!isWailsReady()) return [];
    return AppAny.GetRunningScanTasks();
  },
  updateScanTask: async (id: number, name: string, pocsJson: string, targetsJson: string) => {
    if (!isWailsReady()) return null;
    return AppAny.UpdateScanTask(id, name, pocsJson, targetsJson);
  },
  deleteScanTask: async (id: number) => {
    if (!isWailsReady()) return;
    return AppAny.DeleteScanTask(id);
  },

  // 扫描结果相关
  getScanTaskResult: async (id: number) => {
    if (!isWailsReady()) return null as any;
    return AppAny.GetScanTaskResult(id);
  },
  getAllScanResults: async () => {
    if (!isWailsReady()) return [];
    return AppAny.GetAllScanResults();
  },
  getScanResults: async (id: number) => {
    if (!isWailsReady()) return [];
    return AppAny.GetScanResults(id);
  },
  getTaskProgress: async (id: number) => {
    if (!isWailsReady()) return null as any;
    return AppAny.GetTaskProgress(id);
  },
  getTaskLogs: async (id: number) => {
    if (!isWailsReady()) return [];
    return AppAny.GetTaskLogs(id);
  },
  getTaskLogsFromFile: async (id: number) => {
    if (!isWailsReady()) return [];
    return AppAny.GetTaskLogsFromFile(id);
  },
  getTaskLogSummary: async (id: number) => {
    if (!isWailsReady()) return {} as any;
    return AppAny.GetTaskLogSummary(id);
  },

  // HTTP请求日志相关
  getTaskHTTPLogs: async (taskId: number) => {
    if (!isWailsReady()) return [];
    return AppAny.GetTaskHTTPLogs(taskId);
  },

  // POC模板相关
  getPOCTemplateContent: async (templatePath: string) => {
    if (!isWailsReady()) return '';
    return AppAny.GetPOCTemplateContent(templatePath);
  },

  // 导出相关
  exportTaskResultAsJSON: async (taskId: number) => {
    if (!isWailsReady()) return '';
    return AppAny.ExportTaskResultAsJSON(taskId);
  },
  getScanResult: async (id: number) => {
    if (!isWailsReady()) return {} as any;
    return AppAny.GetScanResult(id);
  },
  deleteScanResult: async (file: string) => {
    if (!isWailsReady()) return;
    return AppAny.DeleteScanResult(file);
  },
  listResultFiles: async () => {
    if (!isWailsReady()) return [];
    return AppAny.ListResultFiles();
  },

  // 工具相关
  selectDirectory: async () => {
    if (!isWailsReady()) return '';
    return AppAny.SelectDirectory();
  },
  selectNucleiDirectory: async () => {
    if (!isWailsReady()) return '';
    return AppAny.SelectNucleiDirectory();
  },
  checkNucleiInstalled: async () => {
    if (!isWailsReady()) return false;
    return AppAny.CheckNucleiInstalled();
  },
  validateNucleiPath: async () => {
    if (!isWailsReady()) return;
    return AppAny.ValidateNucleiPath();
  },
  testNucleiPath: async (path: string) => {
    if (!isWailsReady()) return { valid: false, message: 'Not in Wails runtime' } as any;
    return AppAny.TestNucleiPath(path);
  },
  setNucleiPath: async (path: string) => {
    if (!isWailsReady()) return;
    return AppAny.SetNucleiPath(path);
  },
  
  // 代理测试相关
  testProxies: async (proxies: string[]) => {
    if (!isWailsReady()) return { results: [] } as any;
    return AppAny.TestProxies(proxies);
  },

  // 应用信息
  getAppInfo: async () => {
    if (!isWailsReady()) return {} as any;
    return AppAny.GetAppInfo();
  },

  // 事件监听
  onTemplateImportProgress: (callback: (data: any) => void) => {
    // 检查Wails运行时是否已初始化
    try {
      const hasRuntime = typeof window !== 'undefined' && (window as any).runtime;
      if (!hasRuntime || typeof EventsOn === 'undefined') {
        console.warn('Wails runtime not initialized yet for template import progress');
        return () => {}; // 返回空的取消订阅函数
      }
      return EventsOn('template-import-progress', callback);
    } catch (err) {
      console.warn('EventsOn guard caught error:', err);
      return () => {};
    }
  },
  onScanEvent: (callback: (data: any) => void) => {
    // 检查Wails运行时是否已初始化
    try {
      const hasRuntime = typeof window !== 'undefined' && (window as any).runtime;
      if (!hasRuntime || typeof EventsOn === 'undefined') {
        console.warn('Wails runtime not initialized yet, will not subscribe.');
        return () => {}; // 返回空的取消订阅函数
      }
      return EventsOn('scan-event', callback);
    } catch (err) {
      console.warn('EventsOn guard caught error:', err);
      return () => {};
    }
  },
};
