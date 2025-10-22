export namespace main {
	
	export class NucleiTestResult {
	    valid: boolean;
	    version: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new NucleiTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.version = source["version"];
	        this.error = source["error"];
	    }
	}

}

export namespace models {
	
	export class Config {
	    poc_directory: string;
	    results_dir: string;
	    database_path: string;
	    nuclei_path: string;
	    max_concurrency: number;
	    timeout: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.poc_directory = source["poc_directory"];
	        this.results_dir = source["results_dir"];
	        this.database_path = source["database_path"];
	        this.nuclei_path = source["nuclei_path"];
	        this.max_concurrency = source["max_concurrency"];
	        this.timeout = source["timeout"];
	    }
	}
	export class NucleiInfo {
	    name: string;
	    author: string[];
	    tags: string[];
	    description?: string;
	    reference?: any;
	    severity: string;
	    metadata?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new NucleiInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.author = source["author"];
	        this.tags = source["tags"];
	        this.description = source["description"];
	        this.reference = source["reference"];
	        this.severity = source["severity"];
	        this.metadata = source["metadata"];
	    }
	}
	export class NucleiResult {
	    "template-id": string;
	    "template-path": string;
	    info: NucleiInfo;
	    type: string;
	    host: string;
	    "matched-at": string;
	    "extracted-results"?: string[];
	    request?: string;
	    response?: string;
	    // Go type: time
	    timestamp: any;
	    "curl-command"?: string;
	    metadata?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new NucleiResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this["template-id"] = source["template-id"];
	        this["template-path"] = source["template-path"];
	        this.info = this.convertValues(source["info"], NucleiInfo);
	        this.type = source["type"];
	        this.host = source["host"];
	        this["matched-at"] = source["matched-at"];
	        this["extracted-results"] = source["extracted-results"];
	        this.request = source["request"];
	        this.response = source["response"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this["curl-command"] = source["curl-command"];
	        this.metadata = source["metadata"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScanLog {
	    task_id: number;
	    // Go type: time
	    timestamp: any;
	    level: string;
	    template_id: string;
	    target: string;
	    message: string;
	    is_vuln_found: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScanLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task_id = source["task_id"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.level = source["level"];
	        this.template_id = source["template_id"];
	        this.target = source["target"];
	        this.message = source["message"];
	        this.is_vuln_found = source["is_vuln_found"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScanTask {
	    id: number;
	    name: string;
	    status: string;
	    pocs: string;
	    targets: string;
	    total_requests: number;
	    completed_requests: number;
	    found_vulns: number;
	    output_file: string;
	    // Go type: time
	    start_time: any;
	    // Go type: time
	    end_time?: any;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new ScanTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.pocs = source["pocs"];
	        this.targets = source["targets"];
	        this.total_requests = source["total_requests"];
	        this.completed_requests = source["completed_requests"];
	        this.found_vulns = source["found_vulns"];
	        this.output_file = source["output_file"];
	        this.start_time = this.convertValues(source["start_time"], null);
	        this.end_time = this.convertValues(source["end_time"], null);
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TaskProgress {
	    task_id: number;
	    total_requests: number;
	    completed_requests: number;
	    found_vulns: number;
	    percentage: number;
	    current_template: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task_id = source["task_id"];
	        this.total_requests = source["total_requests"];
	        this.completed_requests = source["completed_requests"];
	        this.found_vulns = source["found_vulns"];
	        this.percentage = source["percentage"];
	        this.current_template = source["current_template"];
	        this.status = source["status"];
	    }
	}
	export class Template {
	    id: number;
	    template_id: string;
	    name: string;
	    severity: string;
	    tags: string;
	    author: string;
	    file_path: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Template(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.template_id = source["template_id"];
	        this.name = source["name"];
	        this.severity = source["severity"];
	        this.tags = source["tags"];
	        this.author = source["author"];
	        this.file_path = source["file_path"];
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace scanner {
	
	export class ImportResult {
	    total_found: number;
	    validated: number;
	    failed: number;
	    already_exists: number;
	    errors: string[];
	    valid_templates?: models.Template[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_found = source["total_found"];
	        this.validated = source["validated"];
	        this.failed = source["failed"];
	        this.already_exists = source["already_exists"];
	        this.errors = source["errors"];
	        this.valid_templates = this.convertValues(source["valid_templates"], models.Template);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScanLogEntry {
	    // Go type: time
	    timestamp: any;
	    level: string;
	    template_id?: string;
	    target?: string;
	    message: string;
	    request?: string;
	    response?: string;
	    is_vuln_found: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScanLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.level = source["level"];
	        this.template_id = source["template_id"];
	        this.target = source["target"];
	        this.message = source["message"];
	        this.request = source["request"];
	        this.response = source["response"];
	        this.is_vuln_found = source["is_vuln_found"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TaskConfig {
	    id: number;
	    name: string;
	    status: string;
	    pocs: string[];
	    targets: string[];
	    total_requests: number;
	    completed_requests: number;
	    found_vulns: number;
	    // Go type: time
	    start_time: any;
	    // Go type: time
	    end_time?: any;
	    output_file: string;
	    log_file: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new TaskConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.pocs = source["pocs"];
	        this.targets = source["targets"];
	        this.total_requests = source["total_requests"];
	        this.completed_requests = source["completed_requests"];
	        this.found_vulns = source["found_vulns"];
	        this.start_time = this.convertValues(source["start_time"], null);
	        this.end_time = this.convertValues(source["end_time"], null);
	        this.output_file = source["output_file"];
	        this.log_file = source["log_file"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TaskResult {
	    task_id: number;
	    task_name: string;
	    status: string;
	    // Go type: time
	    start_time: any;
	    // Go type: time
	    end_time: any;
	    duration: string;
	    targets: string[];
	    templates: string[];
	    template_count: number;
	    target_count: number;
	    total_requests: number;
	    completed_requests: number;
	    found_vulns: number;
	    success_rate: number;
	    vulnerabilities: models.NucleiResult[];
	    summary: Record<string, any>;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new TaskResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task_id = source["task_id"];
	        this.task_name = source["task_name"];
	        this.status = source["status"];
	        this.start_time = this.convertValues(source["start_time"], null);
	        this.end_time = this.convertValues(source["end_time"], null);
	        this.duration = source["duration"];
	        this.targets = source["targets"];
	        this.templates = source["templates"];
	        this.template_count = source["template_count"];
	        this.target_count = source["target_count"];
	        this.total_requests = source["total_requests"];
	        this.completed_requests = source["completed_requests"];
	        this.found_vulns = source["found_vulns"];
	        this.success_rate = source["success_rate"];
	        this.vulnerabilities = this.convertValues(source["vulnerabilities"], models.NucleiResult);
	        this.summary = source["summary"];
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

