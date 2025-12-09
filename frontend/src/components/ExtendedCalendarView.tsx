import { useState } from 'react';
import './ExtendedCalendarView.css';
import { CalendarSidebar } from './CalendarSidebar';
import { MainCalendar } from './MainCalendar';
import { CalendarSource, CalendarEvent } from '../types';

export function ExtendedCalendarView() {
    const [currentDate, setCurrentDate] = useState(new Date());

    // Mock Data
    const [sources, setSources] = useState<CalendarSource[]>([
        { id: '1', email: 'personal@gmail.com', color: '#ff5f57', enabled: true, type: 'gmail' },
        { id: '2', email: 'work@corp.com', color: '#febc2e', enabled: true, type: 'outlook' },
        { id: '3', email: 'family@icloud.com', color: '#28c840', enabled: false, type: 'icloud' },
    ]);

    // Generate some dummy events relative to current date
    const [events] = useState<CalendarEvent[]>([
        { id: '101', title: 'Team Sync', start: new Date(new Date().setHours(10, 0)), end: new Date(new Date().setHours(11, 0)), sourceId: '1', description: 'Weekly sync with engineering.' },
        { id: '102', title: 'Project Due', start: new Date(new Date().setDate(new Date().getDate() + 3)), end: new Date(new Date().setDate(new Date().getDate() + 3)), sourceId: '2', description: 'Submit Q3 deliverables.', isAllDay: true },
        { id: '103', title: 'Family Dinner', start: new Date(new Date().setDate(new Date().getDate() + 5)), end: new Date(new Date().setDate(new Date().getDate() + 5)), sourceId: '3', description: 'Mom\'s Birthday.' },
    ]);

    const toggleSource = (id: string) => {
        setSources(prev => prev.map(s => s.id === id ? { ...s, enabled: !s.enabled } : s));
    };

    const enabledSourceIds = sources.filter(s => s.enabled).map(s => s.id);
    const filteredEvents = events.filter(evt => enabledSourceIds.includes(evt.sourceId));

    return (
        <div className="extended-grid">
            <div className="extended-sidebar">
                <CalendarSidebar
                    currentDate={currentDate}
                    sources={sources}
                    onToggleSource={toggleSource}
                    onDateChange={setCurrentDate}
                />
            </div>
            <div className="extended-main">
                <MainCalendar
                    currentDate={currentDate}
                    events={filteredEvents}
                    sources={sources}
                />
            </div>
        </div>
    );
}
