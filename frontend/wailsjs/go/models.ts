export namespace main {
	
	export class NodeVersion {
	    Version: string;
	    IsCurrent: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NodeVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Version = source["Version"];
	        this.IsCurrent = source["IsCurrent"];
	    }
	}
	export class NodeVersionInfo {
	    Version: string;
	    Status: string;
	
	    static createFrom(source: any = {}) {
	        return new NodeVersionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Version = source["Version"];
	        this.Status = source["Status"];
	    }
	}

}

