export interface CalendarSource {
    id: string;
    email: string;
    color: string; // CSS color string (hex)
    enabled: boolean;
    type: 'gmail' | 'outlook' | 'icloud' | 'local';
}

export interface CalendarEvent {
    id: string;
    title: string;
    description?: string;
    start: Date;
    end: Date;
    sourceId: string;
    isAllDay?: boolean;
}
