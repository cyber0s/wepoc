// Wails runtime import
import * as App from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { Template, ScanTask, TaskProgress, Config, NucleiResult } from '../types';

// Configuration
export const getConfig = (): Promise<Config> => {
  return App.GetConfig();
};

export const saveConfig = (config: Config): Promise<void> => {
  return App.SaveConfig(config);
};

export const reloadConfig = (): Promise<void> => {
  return App.ReloadConfig();
};

// Template Management
export const selectDirectory = (): Promise<string> => {
  return App.SelectDirectory();
};

export const selectNucleiDirectory = (): Promise<string> => {
  return App.SelectNucleiDirectory();
};

export const preValidateTemplates = (dirPath: string): Promise<{
  totalFound: number;
  validated: number;
  failed: number;
  alreadyExists: number;
  errors: string[];
  validTemplates: any[];
}> => {
  return App.PreValidateTemplates(dirPath).then((result: any) => {
    if (result && typeof result === 'object') {
      return {
        totalFound: result.total_found || 0,
        validated: result.validated || 0,
        failed: result.failed || 0,
        alreadyExists: result.already_exists || 0,
        errors: result.errors || [],
        validTemplates: result.valid_templates || []
      };
    } else {
      return {
        totalFound: 0,
        validated: 0,
        failed: 0,
        alreadyExists: 0,
        errors: ['Unknown response format'],
        validTemplates: []
      };
    }
  });
};

export const confirmAndImportTemplates = (validTemplates: any[]): Promise<{
  totalFound: number;
  validated: number;
  failed: number;
  alreadyExists: number;
  errors: string[];
}> => {
  return App.ConfirmAndImportTemplates(validTemplates).then((result: any) => {
    if (result && typeof result === 'object') {
      return {
        totalFound: result.total_found || 0,
        validated: result.validated || 0,
        failed: result.failed || 0,
        alreadyExists: result.already_exists || 0,
        errors: result.errors || []
      };
    } else {
      return {
        totalFound: 0,
        validated: 0,
        failed: 0,
        alreadyExists: 0,
        errors: ['Unknown response format']
      };
    }
  });
};

export const importTemplates = (dirPath: string): Promise<{
  totalFound: number;
  validated: number;
  failed: number;
  alreadyExists: number;
  errors: string[];
}> => {
  return App.ImportTemplates(dirPath).then((result: any) => {
    if (result && typeof result === 'object') {
      return {
        totalFound: result.total_found || 0,
        validated: result.validated || 0,
        failed: result.failed || 0,
        alreadyExists: result.already_exists || 0,
        errors: result.errors || []
      };
    } else {
      return {
        totalFound: 0,
        validated: 0,
        failed: 0,
        alreadyExists: 0,
        errors: ['Unknown response format']
      };
    }
  });
};

export const getAllTemplates = (): Promise<Template[]> => {
  return App.GetAllTemplates();
};

export const deleteTemplate = (templateID: string): Promise<void> => {
  return App.DeleteTemplate(templateID);
};

export const searchTemplates = (keyword: string, severity: string): Promise<Template[]> => {
  return App.SearchTemplates(keyword, severity);
};

export const clearAllTemplates = (): Promise<void> => {
  return App.ClearAllTemplates();
};


// Scan Tasks (JSON-based)
export const createScanTask = (pocs: string[], targets: string[], taskName?: string): Promise<any> => {
  return App.CreateScanTask(JSON.stringify(pocs), JSON.stringify(targets), taskName || '');
};

export const startScanTask = (taskId: number): Promise<void> => {
  return App.StartScanTask(taskId);
};

export const rescanTask = (taskId: number): Promise<void> => {
  return App.RescanTask(taskId);
};

export const pauseScanTask = (taskId: number): Promise<void> => {
  return App.PauseScanTask(taskId);
};

export const stopScanTask = (taskId: number): Promise<void> => {
  return App.StopScanTask(taskId);
};

export const getAllScanTasks = (): Promise<any[]> => {
  return App.GetAllScanTasks();
};

export const deleteScanTask = (taskId: number): Promise<void> => {
  return App.DeleteScanTask(taskId);
};

export const getScanTaskResult = (taskId: number): Promise<any> => {
  return App.GetScanTaskResult(taskId);
};

export const getAllScanResults = (): Promise<any[]> => {
  return App.GetAllScanResults();
};

export const getRunningScanTasks = (): Promise<any[]> => {
  return App.GetRunningScanTasks();
};

export const getTaskProgress = (taskId: number): Promise<TaskProgress> => {
  return App.GetTaskProgress(taskId);
};

export const getTaskLogs = (taskId: number): Promise<any[]> => {
  return App.GetTaskLogs(taskId);
};

// Get task logs from JSON file (new method)
export const getTaskLogsFromFile = (taskId: number): Promise<any[]> => {
  return App.GetTaskLogsFromFile(taskId);
};

export const getTaskLogSummary = (taskId: number): Promise<any> => {
  return App.GetTaskLogSummary(taskId);
};

export const getScanResult = (taskId: number): Promise<any> => {
  return App.GetScanResult(taskId);
};

export const deleteScanResult = (filepath: string): Promise<void> => {
  return App.DeleteScanResult(filepath);
};

// Results
export const getScanResults = (taskId: number): Promise<NucleiResult[]> => {
  return App.GetScanResults(taskId);
};

export const listResultFiles = (): Promise<any[]> => {
  return App.ListResultFiles();
};

// Utilities
export const checkNucleiInstalled = (): Promise<{ installed: boolean; version: string }> => {
  return App.CheckNucleiInstalled().then((result: any) => {
    // 后端返回的是 [bool, string, error] 数组
    if (Array.isArray(result) && result.length >= 2) {
      return {
        installed: result[0],
        version: result[1] || ''
      };
    }
    // 如果返回格式不是数组，可能是错误
    throw new Error('Invalid response format from CheckNucleiInstalled');
  });
};

export const testNucleiPath = (userPath: string): Promise<{ valid: boolean; version: string }> => {
  return App.TestNucleiPath(userPath).then((result: any) => {
    // 后端现在返回的是 NucleiTestResult 结构体
    if (result && typeof result === 'object') {
      return {
        valid: result.valid || false,
        version: result.version || ''
      };
    }
    // 如果返回格式不是对象，可能是错误
    throw new Error('Invalid response format from TestNucleiPath');
  });
};

export const setNucleiPath = (newPath: string): Promise<void> => {
  return App.SetNucleiPath(newPath);
};

export const validateNucleiPath = (): Promise<void> => {
  return App.ValidateNucleiPath();
};

export const getAppInfo = (): Promise<Record<string, string>> => {
  return App.GetAppInfo();
};

// Event Listeners
export const onTaskEvent = (callback: (event: any) => void) => {
  EventsOn('task-event', callback);
  // EventsOn doesn't return a cleanup function, so we return a no-op function
  return () => {};
};

// Listen to scan events (new event system)
export const onScanEvent = (callback: (event: any) => void) => {
  EventsOn('scan-event', callback);
  // EventsOn doesn't return a cleanup function, so we return a no-op function
  return () => {};
};

// Listen to template import progress events
export const onTemplateImportProgress = (callback: (event: any) => void) => {
  EventsOn('template-import-progress', callback);
  // EventsOn doesn't return a cleanup function, so we return a no-op function
  return () => {};
};
