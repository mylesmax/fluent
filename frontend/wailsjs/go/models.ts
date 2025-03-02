export namespace db {
	
	export class Profile {
	    id: number;
	    name: string;
	    emoji: string;
	    glowColor: string;
	    currentDrops: number;
	    classDBPath: string;
	    classUUID: string;
	    isAddNew: boolean;
	    active: boolean;
	    nameHistory: string[];
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.emoji = source["emoji"];
	        this.glowColor = source["glowColor"];
	        this.currentDrops = source["currentDrops"];
	        this.classDBPath = source["classDBPath"];
	        this.classUUID = source["classUUID"];
	        this.isAddNew = source["isAddNew"];
	        this.active = source["active"];
	        this.nameHistory = source["nameHistory"];
	    }
	}
	export class SessionData {
	    id: string;
	    classUUID: string;
	    uploadPrompt: string;
	    chatHistory: number[];
	    // Go type: time
	    startTimestamp: any;
	    // Go type: time
	    endTimestamp?: any;
	
	    static createFrom(source: any = {}) {
	        return new SessionData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.classUUID = source["classUUID"];
	        this.uploadPrompt = source["uploadPrompt"];
	        this.chatHistory = source["chatHistory"];
	        this.startTimestamp = this.convertValues(source["startTimestamp"], null);
	        this.endTimestamp = this.convertValues(source["endTimestamp"], null);
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

