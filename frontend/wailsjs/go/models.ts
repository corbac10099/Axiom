export namespace filesystem {
	
	export class FileEntry {
	    path: string;
	    name: string;
	    is_dir: boolean;
	    size: number;
	    mod_time: number;
	
	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.is_dir = source["is_dir"];
	        this.size = source["size"];
	        this.mod_time = source["mod_time"];
	    }
	}

}

