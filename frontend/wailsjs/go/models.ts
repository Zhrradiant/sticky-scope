export namespace model {
	
	export class FileChange {
	    path: string;
	    status: string;
	    added: number;
	    removed: number;
	    binary: boolean;
	    oldSize: number;
	    newSize: number;
	
	    static createFrom(source: any = {}) {
	        return new FileChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	        this.added = source["added"];
	        this.removed = source["removed"];
	        this.binary = source["binary"];
	        this.oldSize = source["oldSize"];
	        this.newSize = source["newSize"];
	    }
	}
	export class ChangeSet {
	    projectId: string;
	    files: FileChange[];
	    totalAdded: number;
	    totalRemoved: number;
	    totalFiles: number;
	    truncated: boolean;
	    generatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ChangeSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.files = this.convertValues(source["files"], FileChange);
	        this.totalAdded = source["totalAdded"];
	        this.totalRemoved = source["totalRemoved"];
	        this.totalFiles = source["totalFiles"];
	        this.truncated = source["truncated"];
	        this.generatedAt = source["generatedAt"];
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
	export class DiffLine {
	    kind: string;
	    content: string;
	    oldLine: number;
	    newLine: number;
	
	    static createFrom(source: any = {}) {
	        return new DiffLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.content = source["content"];
	        this.oldLine = source["oldLine"];
	        this.newLine = source["newLine"];
	    }
	}
	
	export class Hunk {
	    oldStart: number;
	    oldLines: number;
	    newStart: number;
	    newLines: number;
	    header: string;
	    lines: DiffLine[];
	
	    static createFrom(source: any = {}) {
	        return new Hunk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oldStart = source["oldStart"];
	        this.oldLines = source["oldLines"];
	        this.newStart = source["newStart"];
	        this.newLines = source["newLines"];
	        this.header = source["header"];
	        this.lines = this.convertValues(source["lines"], DiffLine);
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
	export class FileDiff {
	    path: string;
	    status: string;
	    binary: boolean;
	    truncated: boolean;
	    added: number;
	    removed: number;
	    oldSize: number;
	    newSize: number;
	    message: string;
	    hunks: Hunk[];
	
	    static createFrom(source: any = {}) {
	        return new FileDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	        this.binary = source["binary"];
	        this.truncated = source["truncated"];
	        this.added = source["added"];
	        this.removed = source["removed"];
	        this.oldSize = source["oldSize"];
	        this.newSize = source["newSize"];
	        this.message = source["message"];
	        this.hunks = this.convertValues(source["hunks"], Hunk);
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
	
	export class ProjectInfo {
	    id: string;
	    name: string;
	    path: string;
	    createdAt: string;
	    available: boolean;
	    ignore: string[];
	    useGitignore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProjectInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.createdAt = source["createdAt"];
	        this.available = source["available"];
	        this.ignore = source["ignore"];
	        this.useGitignore = source["useGitignore"];
	    }
	}
	export class SettingsInfo {
	    language: string;
	    defaultPatterns: string[];
	
	    static createFrom(source: any = {}) {
	        return new SettingsInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.language = source["language"];
	        this.defaultPatterns = source["defaultPatterns"];
	    }
	}

}

