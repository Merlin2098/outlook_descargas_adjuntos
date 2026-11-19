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
	
	    static createFrom(source: any = {}) {
	        return new SourceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.demo = source["demo"];
	    }
	}

}

