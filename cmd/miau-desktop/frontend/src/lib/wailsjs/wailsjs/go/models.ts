export namespace desktop {
	
	export class AccountDTO {
	    email: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new AccountDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.name = source["name"];
	    }
	}
	export class AnalyticsOverviewDTO {
	    totalEmails: number;
	    unreadEmails: number;
	    starredEmails: number;
	    archivedEmails: number;
	    sentEmails: number;
	    draftCount: number;
	    storageUsedMb: number;
	
	    static createFrom(source: any = {}) {
	        return new AnalyticsOverviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalEmails = source["totalEmails"];
	        this.unreadEmails = source["unreadEmails"];
	        this.starredEmails = source["starredEmails"];
	        this.archivedEmails = source["archivedEmails"];
	        this.sentEmails = source["sentEmails"];
	        this.draftCount = source["draftCount"];
	        this.storageUsedMb = source["storageUsedMb"];
	    }
	}
	export class ResponseTimeStatsDTO {
	    avgResponseMinutes: number;
	    medianMinutes: number;
	    responseRate: number;
	
	    static createFrom(source: any = {}) {
	        return new ResponseTimeStatsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.avgResponseMinutes = source["avgResponseMinutes"];
	        this.medianMinutes = source["medianMinutes"];
	        this.responseRate = source["responseRate"];
	    }
	}
	export class WeekdayStatsDTO {
	    weekday: number;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new WeekdayStatsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.weekday = source["weekday"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class HourlyStatsDTO {
	    hour: number;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new HourlyStatsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hour = source["hour"];
	        this.count = source["count"];
	    }
	}
	export class DailyStatsDTO {
	    date: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new DailyStatsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.count = source["count"];
	    }
	}
	export class EmailTrendsDTO {
	    daily: DailyStatsDTO[];
	    hourly: HourlyStatsDTO[];
	    weekday: WeekdayStatsDTO[];
	
	    static createFrom(source: any = {}) {
	        return new EmailTrendsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.daily = this.convertValues(source["daily"], DailyStatsDTO);
	        this.hourly = this.convertValues(source["hourly"], HourlyStatsDTO);
	        this.weekday = this.convertValues(source["weekday"], WeekdayStatsDTO);
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
	export class SenderStatsDTO {
	    email: string;
	    name: string;
	    count: number;
	    unreadCount: number;
	    percentage: number;
	
	    static createFrom(source: any = {}) {
	        return new SenderStatsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.name = source["name"];
	        this.count = source["count"];
	        this.unreadCount = source["unreadCount"];
	        this.percentage = source["percentage"];
	    }
	}
	export class AnalyticsResultDTO {
	    overview: AnalyticsOverviewDTO;
	    topSenders: SenderStatsDTO[];
	    trends: EmailTrendsDTO;
	    responseTime: ResponseTimeStatsDTO;
	    period: string;
	    // Go type: time
	    generatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AnalyticsResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.overview = this.convertValues(source["overview"], AnalyticsOverviewDTO);
	        this.topSenders = this.convertValues(source["topSenders"], SenderStatsDTO);
	        this.trends = this.convertValues(source["trends"], EmailTrendsDTO);
	        this.responseTime = this.convertValues(source["responseTime"], ResponseTimeStatsDTO);
	        this.period = source["period"];
	        this.generatedAt = this.convertValues(source["generatedAt"], null);
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
	export class AttachmentDTO {
	    id: number;
	    filename: string;
	    contentType: string;
	    contentId?: string;
	    size: number;
	    data?: string;
	    isInline: boolean;
	    partNumber?: string;
	
	    static createFrom(source: any = {}) {
	        return new AttachmentDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.filename = source["filename"];
	        this.contentType = source["contentType"];
	        this.contentId = source["contentId"];
	        this.size = source["size"];
	        this.data = source["data"];
	        this.isInline = source["isInline"];
	        this.partNumber = source["partNumber"];
	    }
	}
	export class AvailableFolderDTO {
	    name: string;
	    isSelected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AvailableFolderDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isSelected = source["isSelected"];
	    }
	}
	export class BasecampAccountDTO {
	    id: number;
	    name: string;
	    href: string;
	
	    static createFrom(source: any = {}) {
	        return new BasecampAccountDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.href = source["href"];
	    }
	}
	export class BasecampConfigDTO {
	    enabled: boolean;
	    clientId?: string;
	    clientSecret?: string;
	    accountId?: string;
	    connected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BasecampConfigDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.clientId = source["clientId"];
	        this.clientSecret = source["clientSecret"];
	        this.accountId = source["accountId"];
	        this.connected = source["connected"];
	    }
	}
	export class BasecampPersonDTO {
	    id: number;
	    name: string;
	    emailAddress: string;
	    title?: string;
	    avatarUrl?: string;
	    admin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BasecampPersonDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.emailAddress = source["emailAddress"];
	        this.title = source["title"];
	        this.avatarUrl = source["avatarUrl"];
	        this.admin = source["admin"];
	    }
	}
	export class BasecampMessageDTO {
	    id: number;
	    projectId: number;
	    subject: string;
	    content: string;
	    creator?: BasecampPersonDTO;
	    commentsCount: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new BasecampMessageDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.subject = source["subject"];
	        this.content = source["content"];
	        this.creator = this.convertValues(source["creator"], BasecampPersonDTO);
	        this.commentsCount = source["commentsCount"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	
	export class BasecampProjectDTO {
	    id: number;
	    name: string;
	    description?: string;
	    status: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new BasecampProjectDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class BasecampTodoDTO {
	    id: number;
	    todoListId: number;
	    projectId: number;
	    content: string;
	    description?: string;
	    dueOn?: string;
	    completed: boolean;
	    // Go type: time
	    completedAt?: any;
	    creator?: BasecampPersonDTO;
	    assignees?: BasecampPersonDTO[];
	    commentsCount: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new BasecampTodoDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.todoListId = source["todoListId"];
	        this.projectId = source["projectId"];
	        this.content = source["content"];
	        this.description = source["description"];
	        this.dueOn = source["dueOn"];
	        this.completed = source["completed"];
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.creator = this.convertValues(source["creator"], BasecampPersonDTO);
	        this.assignees = this.convertValues(source["assignees"], BasecampPersonDTO);
	        this.commentsCount = source["commentsCount"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class BasecampTodoInputDTO {
	    id?: number;
	    todoListId: number;
	    projectId: number;
	    content: string;
	    description?: string;
	    // Go type: time
	    dueDate?: any;
	    assigneeIds?: number[];
	
	    static createFrom(source: any = {}) {
	        return new BasecampTodoInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.todoListId = source["todoListId"];
	        this.projectId = source["projectId"];
	        this.content = source["content"];
	        this.description = source["description"];
	        this.dueDate = this.convertValues(source["dueDate"], null);
	        this.assigneeIds = source["assigneeIds"];
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
	export class BasecampTodoListDTO {
	    id: number;
	    projectId: number;
	    title: string;
	    description?: string;
	    completed: boolean;
	    completedRatio?: string;
	    todosCount: number;
	    completedCount: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new BasecampTodoListDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.completed = source["completed"];
	        this.completedRatio = source["completedRatio"];
	        this.todosCount = source["todosCount"];
	        this.completedCount = source["completedCount"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class CalendarEventCountsDTO {
	    upcoming: number;
	    completed: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new CalendarEventCountsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.upcoming = source["upcoming"];
	        this.completed = source["completed"];
	        this.total = source["total"];
	    }
	}
	export class CalendarEventDTO {
	    id: number;
	    accountId: number;
	    title: string;
	    description?: string;
	    eventType: string;
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime?: any;
	    allDay: boolean;
	    color?: string;
	    taskId?: number;
	    emailId?: number;
	    isCompleted: boolean;
	    source: string;
	    googleEventId?: string;
	    googleCalendarId?: string;
	    // Go type: time
	    lastSyncedAt?: any;
	    syncStatus: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new CalendarEventDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.accountId = source["accountId"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.eventType = source["eventType"];
	        this.startTime = this.convertValues(source["startTime"], null);
	        this.endTime = this.convertValues(source["endTime"], null);
	        this.allDay = source["allDay"];
	        this.color = source["color"];
	        this.taskId = source["taskId"];
	        this.emailId = source["emailId"];
	        this.isCompleted = source["isCompleted"];
	        this.source = source["source"];
	        this.googleEventId = source["googleEventId"];
	        this.googleCalendarId = source["googleCalendarId"];
	        this.lastSyncedAt = this.convertValues(source["lastSyncedAt"], null);
	        this.syncStatus = source["syncStatus"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class CalendarEventInputDTO {
	    id?: number;
	    title: string;
	    description?: string;
	    eventType?: string;
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime?: any;
	    allDay: boolean;
	    color?: string;
	    taskId?: number;
	    emailId?: number;
	    isCompleted: boolean;
	    source?: string;
	
	    static createFrom(source: any = {}) {
	        return new CalendarEventInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.eventType = source["eventType"];
	        this.startTime = this.convertValues(source["startTime"], null);
	        this.endTime = this.convertValues(source["endTime"], null);
	        this.allDay = source["allDay"];
	        this.color = source["color"];
	        this.taskId = source["taskId"];
	        this.emailId = source["emailId"];
	        this.isCompleted = source["isCompleted"];
	        this.source = source["source"];
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
	export class ConnectionStatus {
	    connected: boolean;
	    // Go type: time
	    lastSync: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.lastSync = this.convertValues(source["lastSync"], null);
	        this.error = source["error"];
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
	export class ContactPhoneDTO {
	    phone: string;
	    type?: string;
	    isPrimary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContactPhoneDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phone = source["phone"];
	        this.type = source["type"];
	        this.isPrimary = source["isPrimary"];
	    }
	}
	export class ContactEmailDTO {
	    email: string;
	    type?: string;
	    isPrimary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContactEmailDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.type = source["type"];
	        this.isPrimary = source["isPrimary"];
	    }
	}
	export class ContactDTO {
	    id: number;
	    displayName: string;
	    givenName?: string;
	    familyName?: string;
	    photoUrl?: string;
	    photoPath?: string;
	    isStarred: boolean;
	    interactionCount: number;
	    emails: ContactEmailDTO[];
	    phones?: ContactPhoneDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ContactDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.givenName = source["givenName"];
	        this.familyName = source["familyName"];
	        this.photoUrl = source["photoUrl"];
	        this.photoPath = source["photoPath"];
	        this.isStarred = source["isStarred"];
	        this.interactionCount = source["interactionCount"];
	        this.emails = this.convertValues(source["emails"], ContactEmailDTO);
	        this.phones = this.convertValues(source["phones"], ContactPhoneDTO);
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
	
	
	export class ContactSyncStatusDTO {
	    totalContacts: number;
	    // Go type: time
	    lastSync?: any;
	    status: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ContactSyncStatusDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalContacts = source["totalContacts"];
	        this.lastSync = this.convertValues(source["lastSync"], null);
	        this.status = source["status"];
	        this.error = source["error"];
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
	
	export class DraftDTO {
	    id?: number;
	    to: string[];
	    cc: string[];
	    bcc: string[];
	    subject: string;
	    bodyHtml: string;
	    bodyText: string;
	    replyToId?: number;
	
	    static createFrom(source: any = {}) {
	        return new DraftDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.to = source["to"];
	        this.cc = source["cc"];
	        this.bcc = source["bcc"];
	        this.subject = source["subject"];
	        this.bodyHtml = source["bodyHtml"];
	        this.bodyText = source["bodyText"];
	        this.replyToId = source["replyToId"];
	    }
	}
	export class EmailDTO {
	    id: number;
	    uid: number;
	    subject: string;
	    fromName: string;
	    fromEmail: string;
	    // Go type: time
	    date: any;
	    isRead: boolean;
	    isStarred: boolean;
	    hasAttachments: boolean;
	    snippet: string;
	    threadId?: string;
	    threadCount?: number;
	
	    static createFrom(source: any = {}) {
	        return new EmailDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.uid = source["uid"];
	        this.subject = source["subject"];
	        this.fromName = source["fromName"];
	        this.fromEmail = source["fromEmail"];
	        this.date = this.convertValues(source["date"], null);
	        this.isRead = source["isRead"];
	        this.isStarred = source["isStarred"];
	        this.hasAttachments = source["hasAttachments"];
	        this.snippet = source["snippet"];
	        this.threadId = source["threadId"];
	        this.threadCount = source["threadCount"];
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
	export class EmailDetailDTO {
	    id: number;
	    uid: number;
	    subject: string;
	    fromName: string;
	    fromEmail: string;
	    // Go type: time
	    date: any;
	    isRead: boolean;
	    isStarred: boolean;
	    hasAttachments: boolean;
	    snippet: string;
	    threadId?: string;
	    threadCount?: number;
	    toAddresses: string;
	    ccAddresses: string;
	    bodyText: string;
	    bodyHtml: string;
	    attachments: AttachmentDTO[];
	
	    static createFrom(source: any = {}) {
	        return new EmailDetailDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.uid = source["uid"];
	        this.subject = source["subject"];
	        this.fromName = source["fromName"];
	        this.fromEmail = source["fromEmail"];
	        this.date = this.convertValues(source["date"], null);
	        this.isRead = source["isRead"];
	        this.isStarred = source["isStarred"];
	        this.hasAttachments = source["hasAttachments"];
	        this.snippet = source["snippet"];
	        this.threadId = source["threadId"];
	        this.threadCount = source["threadCount"];
	        this.toAddresses = source["toAddresses"];
	        this.ccAddresses = source["ccAddresses"];
	        this.bodyText = source["bodyText"];
	        this.bodyHtml = source["bodyHtml"];
	        this.attachments = this.convertValues(source["attachments"], AttachmentDTO);
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
	
	export class FolderDTO {
	    id: number;
	    name: string;
	    totalMessages: number;
	    unreadMessages: number;
	
	    static createFrom(source: any = {}) {
	        return new FolderDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.totalMessages = source["totalMessages"];
	        this.unreadMessages = source["unreadMessages"];
	    }
	}
	export class GoogleCalendarDTO {
	    id: string;
	    summary: string;
	    description?: string;
	    primary: boolean;
	    backgroundColor?: string;
	    accessRole: string;
	
	    static createFrom(source: any = {}) {
	        return new GoogleCalendarDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.summary = source["summary"];
	        this.description = source["description"];
	        this.primary = source["primary"];
	        this.backgroundColor = source["backgroundColor"];
	        this.accessRole = source["accessRole"];
	    }
	}
	export class GoogleEventDTO {
	    id: string;
	    calendarId: string;
	    summary: string;
	    description?: string;
	    location?: string;
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime: any;
	    allDay: boolean;
	    status: string;
	    htmlLink?: string;
	    colorId?: string;
	
	    static createFrom(source: any = {}) {
	        return new GoogleEventDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.calendarId = source["calendarId"];
	        this.summary = source["summary"];
	        this.description = source["description"];
	        this.location = source["location"];
	        this.startTime = this.convertValues(source["startTime"], null);
	        this.endTime = this.convertValues(source["endTime"], null);
	        this.allDay = source["allDay"];
	        this.status = source["status"];
	        this.htmlLink = source["htmlLink"];
	        this.colorId = source["colorId"];
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
	
	
	export class SearchResultDTO {
	    emails: EmailDTO[];
	    totalCount: number;
	    query: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.emails = this.convertValues(source["emails"], EmailDTO);
	        this.totalCount = source["totalCount"];
	        this.query = source["query"];
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
	export class SendRequest {
	    to: string[];
	    cc: string[];
	    bcc: string[];
	    subject: string;
	    body: string;
	    isHtml: boolean;
	    replyTo?: number;
	
	    static createFrom(source: any = {}) {
	        return new SendRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.to = source["to"];
	        this.cc = source["cc"];
	        this.bcc = source["bcc"];
	        this.subject = source["subject"];
	        this.body = source["body"];
	        this.isHtml = source["isHtml"];
	        this.replyTo = source["replyTo"];
	    }
	}
	export class SendResult {
	    success: boolean;
	    messageId: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SendResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.messageId = source["messageId"];
	        this.error = source["error"];
	    }
	}
	
	export class SettingsDTO {
	    syncFolders: string[];
	    uiTheme: string;
	    uiShowPreview: boolean;
	    uiPageSize: number;
	    composeFormat: string;
	    composeSendDelay: number;
	    syncInterval: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.syncFolders = source["syncFolders"];
	        this.uiTheme = source["uiTheme"];
	        this.uiShowPreview = source["uiShowPreview"];
	        this.uiPageSize = source["uiPageSize"];
	        this.composeFormat = source["composeFormat"];
	        this.composeSendDelay = source["composeSendDelay"];
	        this.syncInterval = source["syncInterval"];
	    }
	}
	export class SummaryResult {
	    emailId: number;
	    style: string;
	    content: string;
	    keyPoints: string[];
	    cached: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SummaryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.emailId = source["emailId"];
	        this.style = source["style"];
	        this.content = source["content"];
	        this.keyPoints = source["keyPoints"];
	        this.cached = source["cached"];
	    }
	}
	export class SyncResultDTO {
	    newEmails: number;
	    deletedEmails: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.newEmails = source["newEmails"];
	        this.deletedEmails = source["deletedEmails"];
	    }
	}
	export class TaskCountsDTO {
	    pending: number;
	    completed: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskCountsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pending = source["pending"];
	        this.completed = source["completed"];
	        this.total = source["total"];
	    }
	}
	export class TaskDTO {
	    id: number;
	    accountId: number;
	    title: string;
	    description?: string;
	    isCompleted: boolean;
	    priority: number;
	    // Go type: time
	    dueDate?: any;
	    emailId?: number;
	    source: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new TaskDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.accountId = source["accountId"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.isCompleted = source["isCompleted"];
	        this.priority = source["priority"];
	        this.dueDate = this.convertValues(source["dueDate"], null);
	        this.emailId = source["emailId"];
	        this.source = source["source"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class TaskInputDTO {
	    id?: number;
	    title: string;
	    description?: string;
	    isCompleted: boolean;
	    priority: number;
	    // Go type: time
	    dueDate?: any;
	    emailId?: number;
	    source?: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.isCompleted = source["isCompleted"];
	        this.priority = source["priority"];
	        this.dueDate = this.convertValues(source["dueDate"], null);
	        this.emailId = source["emailId"];
	        this.source = source["source"];
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
	export class ThreadEmailDTO {
	    id: number;
	    uid: number;
	    messageId: string;
	    subject: string;
	    fromName: string;
	    fromEmail: string;
	    toAddresses: string;
	    // Go type: time
	    date: any;
	    isRead: boolean;
	    isStarred: boolean;
	    isReplied: boolean;
	    hasAttachments: boolean;
	    snippet: string;
	    bodyText: string;
	    bodyHtml: string;
	
	    static createFrom(source: any = {}) {
	        return new ThreadEmailDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.uid = source["uid"];
	        this.messageId = source["messageId"];
	        this.subject = source["subject"];
	        this.fromName = source["fromName"];
	        this.fromEmail = source["fromEmail"];
	        this.toAddresses = source["toAddresses"];
	        this.date = this.convertValues(source["date"], null);
	        this.isRead = source["isRead"];
	        this.isStarred = source["isStarred"];
	        this.isReplied = source["isReplied"];
	        this.hasAttachments = source["hasAttachments"];
	        this.snippet = source["snippet"];
	        this.bodyText = source["bodyText"];
	        this.bodyHtml = source["bodyHtml"];
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
	export class ThreadDTO {
	    threadId: string;
	    subject: string;
	    participants: string[];
	    messageCount: number;
	    messages: ThreadEmailDTO[];
	    isRead: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ThreadDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.threadId = source["threadId"];
	        this.subject = source["subject"];
	        this.participants = source["participants"];
	        this.messageCount = source["messageCount"];
	        this.messages = this.convertValues(source["messages"], ThreadEmailDTO);
	        this.isRead = source["isRead"];
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
	
	export class ThreadSummaryDTO {
	    threadId: string;
	    subject: string;
	    lastSender: string;
	    lastSenderEmail: string;
	    // Go type: time
	    lastDate: any;
	    messageCount: number;
	    unreadCount: number;
	    hasAttachments: boolean;
	    participants: string[];
	
	    static createFrom(source: any = {}) {
	        return new ThreadSummaryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.threadId = source["threadId"];
	        this.subject = source["subject"];
	        this.lastSender = source["lastSender"];
	        this.lastSenderEmail = source["lastSenderEmail"];
	        this.lastDate = this.convertValues(source["lastDate"], null);
	        this.messageCount = source["messageCount"];
	        this.unreadCount = source["unreadCount"];
	        this.hasAttachments = source["hasAttachments"];
	        this.participants = source["participants"];
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
	export class ThreadSummaryResult {
	    threadId: string;
	    participants: string[];
	    timeline: string;
	    keyDecisions: string[];
	    actionItems: string[];
	    cached: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ThreadSummaryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.threadId = source["threadId"];
	        this.participants = source["participants"];
	        this.timeline = source["timeline"];
	        this.keyDecisions = source["keyDecisions"];
	        this.actionItems = source["actionItems"];
	        this.cached = source["cached"];
	    }
	}
	export class UndoResult {
	    success: boolean;
	    description: string;
	    canUndo: boolean;
	    canRedo: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UndoResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.description = source["description"];
	        this.canUndo = source["canUndo"];
	        this.canRedo = source["canRedo"];
	    }
	}

}

