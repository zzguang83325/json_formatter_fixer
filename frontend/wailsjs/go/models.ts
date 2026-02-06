export namespace main {
	
	export class JSONResponse {
	    success: boolean;
	    data: string;
	    error: string;
	    repaired: boolean;
	
	    static createFrom(source: any = {}) {
	        return new JSONResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = source["data"];
	        this.error = source["error"];
	        this.repaired = source["repaired"];
	    }
	}
	export class PathInfo {
	    offset: number;
	    length: number;
	
	    static createFrom(source: any = {}) {
	        return new PathInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.offset = source["offset"];
	        this.length = source["length"];
	    }
	}

}

