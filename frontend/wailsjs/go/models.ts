export namespace db {
	
	export class ExplorerAIHistoryEntry {
	    id: number;
	    class_uuid: string;
	    session_id: string;
	    prompt: string;
	    response: string;
	    tokens: number;
	    cost: number;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new ExplorerAIHistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.class_uuid = source["class_uuid"];
	        this.session_id = source["session_id"];
	        this.prompt = source["prompt"];
	        this.response = source["response"];
	        this.tokens = source["tokens"];
	        this.cost = source["cost"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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
	export class AIHistoryEntries {
	    uploads: ExplorerAIHistoryEntry[];
	    chats: ExplorerAIHistoryEntry[];
	    parser: ExplorerAIHistoryEntry[];
	    extractor: ExplorerAIHistoryEntry[];
	
	    static createFrom(source: any = {}) {
	        return new AIHistoryEntries(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uploads = this.convertValues(source["uploads"], ExplorerAIHistoryEntry);
	        this.chats = this.convertValues(source["chats"], ExplorerAIHistoryEntry);
	        this.parser = this.convertValues(source["parser"], ExplorerAIHistoryEntry);
	        this.extractor = this.convertValues(source["extractor"], ExplorerAIHistoryEntry);
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
	export class ClassInfo {
	    uuid: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    last_modified: any;
	    name_history: string[];
	
	    static createFrom(source: any = {}) {
	        return new ClassInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.last_modified = this.convertValues(source["last_modified"], null);
	        this.name_history = source["name_history"];
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
	export class SessionInfo {
	    id: string;
	    upload_prompt: string;
	    topic: string;
	    chat_history: number[];
	    // Go type: time
	    start_timestamp: any;
	    // Go type: time
	    end_timestamp?: any;
	    status: string;
	    droplet_count: number;
	    total_factoids?: number;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.upload_prompt = source["upload_prompt"];
	        this.topic = source["topic"];
	        this.chat_history = source["chat_history"];
	        this.start_timestamp = this.convertValues(source["start_timestamp"], null);
	        this.end_timestamp = this.convertValues(source["end_timestamp"], null);
	        this.status = source["status"];
	        this.droplet_count = source["droplet_count"];
	        this.total_factoids = source["total_factoids"];
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
	export class DatabaseExplorerData {
	    classInfo: ClassInfo;
	    sessions: SessionInfo[];
	    aiHistory: AIHistoryEntries;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseExplorerData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.classInfo = this.convertValues(source["classInfo"], ClassInfo);
	        this.sessions = this.convertValues(source["sessions"], SessionInfo);
	        this.aiHistory = this.convertValues(source["aiHistory"], AIHistoryEntries);
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
	
	export class FactoidData {
	    id: string;
	    class_uuid: string;
	    question: string;
	    answer: string;
	    type: string;
	    verbatim: string;
	    context: string;
	    requires_clarification: boolean;
	    alternative_subjects_count: number;
	    difficulty: number;
	    examples: string[];
	    // Go type: time
	    last_review: any;
	    // Go type: time
	    next_review: any;
	    stability: number;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new FactoidData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.class_uuid = source["class_uuid"];
	        this.question = source["question"];
	        this.answer = source["answer"];
	        this.type = source["type"];
	        this.verbatim = source["verbatim"];
	        this.context = source["context"];
	        this.requires_clarification = source["requires_clarification"];
	        this.alternative_subjects_count = source["alternative_subjects_count"];
	        this.difficulty = source["difficulty"];
	        this.examples = source["examples"];
	        this.last_review = this.convertValues(source["last_review"], null);
	        this.next_review = this.convertValues(source["next_review"], null);
	        this.stability = source["stability"];
	        this.created_at = this.convertValues(source["created_at"], null);
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

