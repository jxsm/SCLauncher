export namespace mod {
	
	export class Mod {
	    id: string;
	    name: string;
	    fileName: string;
	    versionId: string;
	    enabled: boolean;
	    size: number;
	    installDate: string;
	
	    static createFrom(source: any = {}) {
	        return new Mod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.fileName = source["fileName"];
	        this.versionId = source["versionId"];
	        this.enabled = source["enabled"];
	        this.size = source["size"];
	        this.installDate = source["installDate"];
	    }
	}

}

export namespace skin {
	
	export class Skin {
	    fileName: string;
	    size: number;
	    importDate: string;
	
	    static createFrom(source: any = {}) {
	        return new Skin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.size = source["size"];
	        this.importDate = source["importDate"];
	    }
	}

}

export namespace version {
	
	export class Version {
	    id: string;
	    versionType: string;
	    gameVersion: string;
	    subVersion: string;
	    name: string;
	    size: number;
	    downloadUrl: string;
	    checksum: string;
	    fileFormat: string;
	    illustrate: string;
	    // Go type: time
	    releaseDate: any;
	    installed: boolean;
	    localPath?: string;
	    pathExists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Version(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.versionType = source["versionType"];
	        this.gameVersion = source["gameVersion"];
	        this.subVersion = source["subVersion"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.downloadUrl = source["downloadUrl"];
	        this.checksum = source["checksum"];
	        this.fileFormat = source["fileFormat"];
	        this.illustrate = source["illustrate"];
	        this.releaseDate = this.convertValues(source["releaseDate"], null);
	        this.installed = source["installed"];
	        this.localPath = source["localPath"];
	        this.pathExists = source["pathExists"];
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

