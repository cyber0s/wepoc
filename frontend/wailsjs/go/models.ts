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
	export class ProxyTestResult {
	    url: string;
	    available: boolean;
	    response_time: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.available = source["available"];
	        this.response_time = source["response_time"];
	        this.error = source["error"];
	    }
	}
	export class ProxyTestResults {
	    results: ProxyTestResult[];
	    // Go type: struct { Total int "json:\"total\""; Available int "json:\"available\""; Failed int "json:\"failed\"" }
	    summary: any;
	
	    static createFrom(source: any = {}) {
	        return new ProxyTestResults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], ProxyTestResult);
	        this.summary = this.convertValues(source["summary"], Object);
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
	export class TestSinglePOCParams {
	    template_content: string;
	    target: string;
	    concurrency: number;
	    rate_limit: number;
	    interactsh_url: string;
	    interactsh_token: string;
	    proxy_url: string;
	
	    static createFrom(source: any = {}) {
	        return new TestSinglePOCParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.template_content = source["template_content"];
	        this.target = source["target"];
	        this.concurrency = source["concurrency"];
	        this.rate_limit = source["rate_limit"];
	        this.interactsh_url = source["interactsh_url"];
	        this.interactsh_token = source["interactsh_token"];
	        this.proxy_url = source["proxy_url"];
	    }
	}

}

export namespace models {
	
	export class NucleiAdvancedConfig {
	    concurrency: number;
	    bulk_size: number;
	    rate_limit: number;
	    rate_limit_minute: number;
	    proxy_enabled: boolean;
	    proxy_url: string;
	    proxy_list: string[];
	    proxy_internal: boolean;
	    interactsh_enabled: boolean;
	    interactsh_server: string;
	    interactsh_token: string;
	    interactsh_disable: boolean;
	    retries: number;
	    max_host_error: number;
	    disable_update_check: boolean;
	    follow_redirects: boolean;
	    max_redirects: number;
	
	    static createFrom(source: any = {}) {
	        return new NucleiAdvancedConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.concurrency = source["concurrency"];
	        this.bulk_size = source["bulk_size"];
	        this.rate_limit = source["rate_limit"];
	        this.rate_limit_minute = source["rate_limit_minute"];
	        this.proxy_enabled = source["proxy_enabled"];
	        this.proxy_url = source["proxy_url"];
	        this.proxy_list = source["proxy_list"];
	        this.proxy_internal = source["proxy_internal"];
	        this.interactsh_enabled = source["interactsh_enabled"];
	        this.interactsh_server = source["interactsh_server"];
	        this.interactsh_token = source["interactsh_token"];
	        this.interactsh_disable = source["interactsh_disable"];
	        this.retries = source["retries"];
	        this.max_host_error = source["max_host_error"];
	        this.disable_update_check = source["disable_update_check"];
	        this.follow_redirects = source["follow_redirects"];
	        this.max_redirects = source["max_redirects"];
	    }
	}
	export class Config {
	    poc_directory: string;
	    results_dir: string;
	    database_path: string;
	    nuclei_path: string;
	    max_concurrency: number;
	    timeout: number;
	    nuclei_config: NucleiAdvancedConfig;
	
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
	        this.nuclei_config = this.convertValues(source["nuclei_config"], NucleiAdvancedConfig);
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
	
	export class HTTPRequestLog {
	    id: number;
	    task_id: number;
	    // Go type: time
	    timestamp: any;
	    template_id: string;
	    template_name: string;
	    severity: string;
	    target: string;
	    method: string;
	    status_code: number;
	    is_vuln_found: boolean;
	    request: string;
	    response: string;
	    duration_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new HTTPRequestLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.task_id = source["task_id"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.template_id = source["template_id"];
	        this.template_name = source["template_name"];
	        this.severity = source["severity"];
	        this.target = source["target"];
	        this.method = source["method"];
	        this.status_code = source["status_code"];
	        this.is_vuln_found = source["is_vuln_found"];
	        this.request = source["request"];
	        this.response = source["response"];
	        this.duration_ms = source["duration_ms"];
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
	    scanned_templates: number;
	    filtered_templates: number;
	    skipped_templates: number;
	    failed_templates: number;
	    filtered_template_ids: string[];
	    skipped_template_ids: string[];
	    failed_template_ids: string[];
	    scanned_template_ids: string[];
	    http_requests: number;
	
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
	        this.scanned_templates = source["scanned_templates"];
	        this.filtered_templates = source["filtered_templates"];
	        this.skipped_templates = source["skipped_templates"];
	        this.failed_templates = source["failed_templates"];
	        this.filtered_template_ids = source["filtered_template_ids"];
	        this.skipped_template_ids = source["skipped_template_ids"];
	        this.failed_template_ids = source["failed_template_ids"];
	        this.scanned_template_ids = source["scanned_template_ids"];
	        this.http_requests = source["http_requests"];
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

