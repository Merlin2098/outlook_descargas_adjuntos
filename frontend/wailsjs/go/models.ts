export namespace config {
	
	export class Config {
	    lastDestDir: string;
	    organize: string;
	    skipInline: boolean;
	    windowWidth: number;
	    windowHeight: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lastDestDir = source["lastDestDir"];
	        this.organize = source["organize"];
	        this.skipInline = source["skipInline"];
	        this.windowWidth = source["windowWidth"];
	        this.windowHeight = source["windowHeight"];
	    }
	}

}

export namespace domain {
	
	export class FolderRef {
	    storeId: string;
	    entryId: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new FolderRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.storeId = source["storeId"];
	        this.entryId = source["entryId"];
	        this.path = source["path"];
	    }
	}
	export class Folder {
	    ref: FolderRef;
	    name: string;
	    itemCount: number;
	    hasChildren: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Folder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], FolderRef);
	        this.name = source["name"];
	        this.itemCount = source["itemCount"];
	        this.hasChildren = source["hasChildren"];
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
	
	export class Summary {
	    scanned: number;
	    outOfRange: number;
	    inRange: number;
	    mailsWithDownloads: number;
	    saved: number;
	    skipped: number;
	    errors: number;
	    bytes: number;
	    byExt: Record<string, number>;
	    reasons: Record<string, number>;
	    errorReasons: Record<string, number>;
	    bySubfolder: Record<string, number>;
	    // Go type: time
	    started: any;
	    // Go type: time
	    finished: any;
	    cancelled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scanned = source["scanned"];
	        this.outOfRange = source["outOfRange"];
	        this.inRange = source["inRange"];
	        this.mailsWithDownloads = source["mailsWithDownloads"];
	        this.saved = source["saved"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	        this.bytes = source["bytes"];
	        this.byExt = source["byExt"];
	        this.reasons = source["reasons"];
	        this.errorReasons = source["errorReasons"];
	        this.bySubfolder = source["bySubfolder"];
	        this.started = this.convertValues(source["started"], null);
	        this.finished = this.convertValues(source["finished"], null);
	        this.cancelled = source["cancelled"];
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

export namespace extract {
	
	export class Activity {
	    // Go type: time
	    at: any;
	    kind: string;
	    name: string;
	    subfolder: string;
	    reason: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new Activity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.at = this.convertValues(source["at"], null);
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.subfolder = source["subfolder"];
	        this.reason = source["reason"];
	        this.size = source["size"];
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
	export class Progress {
	    phase: string;
	    scanned: number;
	    inRange: number;
	    mailsDone: number;
	    mailsTotal: number;
	    saved: number;
	    skipped: number;
	    errors: number;
	    bytes: number;
	    elapsedMs: number;
	
	    static createFrom(source: any = {}) {
	        return new Progress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.scanned = source["scanned"];
	        this.inRange = source["inRange"];
	        this.mailsDone = source["mailsDone"];
	        this.mailsTotal = source["mailsTotal"];
	        this.saved = source["saved"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	        this.bytes = source["bytes"];
	        this.elapsedMs = source["elapsedMs"];
	    }
	}

}

export namespace jobs {
	
	export class Terminal {
	    jobId: string;
	    summary: domain.Summary;
	    reportPath: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Terminal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.summary = this.convertValues(source["summary"], domain.Summary);
	        this.reportPath = source["reportPath"];
	        this.message = source["message"];
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
	export class Status {
	    jobId: string;
	    running: boolean;
	    destDir: string;
	    progress: extract.Progress;
	    recent: extract.Activity[];
	    last?: Terminal;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.running = source["running"];
	        this.destDir = source["destDir"];
	        this.progress = this.convertValues(source["progress"], extract.Progress);
	        this.recent = this.convertValues(source["recent"], extract.Activity);
	        this.last = this.convertValues(source["last"], Terminal);
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

export namespace main {
	
	export class JobRequestDTO {
	    folder: domain.FolderRef;
	    from: string;
	    to: string;
	    subjectContains: string;
	    senderContains: string;
	    extensions: string[];
	    skipInline: boolean;
	    includeSubfolders: boolean;
	    destDir: string;
	    organize: string;
	
	    static createFrom(source: any = {}) {
	        return new JobRequestDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = this.convertValues(source["folder"], domain.FolderRef);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.subjectContains = source["subjectContains"];
	        this.senderContains = source["senderContains"];
	        this.extensions = source["extensions"];
	        this.skipInline = source["skipInline"];
	        this.includeSubfolders = source["includeSubfolders"];
	        this.destDir = source["destDir"];
	        this.organize = source["organize"];
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
	export class SourceInfo {
	    name: string;
	    demo: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.demo = source["demo"];
	        this.version = source["version"];
	    }
	}

}

