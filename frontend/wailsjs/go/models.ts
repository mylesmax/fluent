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

}

