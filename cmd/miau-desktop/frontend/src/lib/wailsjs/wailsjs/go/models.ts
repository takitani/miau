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

}

