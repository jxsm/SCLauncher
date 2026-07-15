export namespace main {
	
	export class Furniture {
	    id: string;
	    name: string;
	    fileName: string;
	
	    static createFrom(source: any = {}) {
	        return new Furniture(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.fileName = source["fileName"];
	    }
	}
	export class Texture {
	    id: string;
	    name: string;
	    fileName: string;
	
	    static createFrom(source: any = {}) {
	        return new Texture(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.fileName = source["fileName"];
	    }
	}

}

export namespace mod {
	
	export class Dependency {
	    packageName: string;
	    versionRange: string;
	    displayName?: string;
	
	    static createFrom(source: any = {}) {
	        return new Dependency(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.packageName = source["packageName"];
	        this.versionRange = source["versionRange"];
	        this.displayName = source["displayName"];
	    }
	}
	export class ModInfo {
	    name: string;
	    version: string;
	    apiVersion: string;
	    packageName: string;
	    description: string;
	    scVersion: string;
	    loadOrder: number;
	    nonPersistentMod: boolean;
	    gameplayImpactLevel: string;
	    link: string;
	    author: string;
	    dependencies: Dependency[];
	
	    static createFrom(source: any = {}) {
	        return new ModInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.apiVersion = source["apiVersion"];
	        this.packageName = source["packageName"];
	        this.description = source["description"];
	        this.scVersion = source["scVersion"];
	        this.loadOrder = source["loadOrder"];
	        this.nonPersistentMod = source["nonPersistentMod"];
	        this.gameplayImpactLevel = source["gameplayImpactLevel"];
	        this.link = source["link"];
	        this.author = source["author"];
	        this.dependencies = this.convertValues(source["dependencies"], Dependency);
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
	export class Mod {
	    id: string;
	    name: string;
	    fileName: string;
	    versionId: string;
	    enabled: boolean;
	    size: number;
	    installDate: string;
	    modInfo?: ModInfo;
	
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
	        this.modInfo = this.convertValues(source["modInfo"], ModInfo);
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

export namespace savegame {
	
	export class SaveGame {
	    id: string;
	    name: string;
	    gameVersion: string;
	    gameMode: string;
	    // Go type: time
	    lastModified: any;
	    isAutoSave: boolean;
	    projectPath: string;
	    worldPath: string;
	    isImported: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SaveGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.gameVersion = source["gameVersion"];
	        this.gameMode = source["gameMode"];
	        this.lastModified = this.convertValues(source["lastModified"], null);
	        this.isAutoSave = source["isAutoSave"];
	        this.projectPath = source["projectPath"];
	        this.worldPath = source["worldPath"];
	        this.isImported = source["isImported"];
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
	export class SaveRequiredMod {
	    packageName: string;
	    version: string;
	    name: string;
	    author: string;
	    link: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveRequiredMod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.packageName = source["packageName"];
	        this.version = source["version"];
	        this.name = source["name"];
	        this.author = source["author"];
	        this.link = source["link"];
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
	    isPrimary: boolean;
	    isCustomName: boolean;
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
	        this.isPrimary = source["isPrimary"];
	        this.isCustomName = source["isCustomName"];
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

