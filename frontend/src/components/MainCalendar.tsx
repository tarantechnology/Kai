import { useState } from 'react';
import './MainCalendar.css';
import { CalendarEvent, CalendarSource } from '../types';
import { generateMonthGrid, isSameDate, isCurrentMonth, formatMonthYear } from '../utils/calendar';
import { format } from 'date-fns';

interface MainCalendarProps {
    currentDate: Date;
    events: CalendarEvent[];
    sources: CalendarSource[];
}

export function MainCalendar({ currentDate, events, sources }: MainCalendarProps) {
    const days = generateMonthGrid(currentDate);
    const [hoveredEvent, setHoveredEvent] = useState<{ event: CalendarEvent, x: number, y: number } | null>(null);

    const getSourceColor = (sourceId: string) => {
        return sources.find(s => s.id === sourceId)?.color || '#fff';
    };

    const handleMouseEnter = (e: React.MouseEvent, evt: CalendarEvent) => {
        const rect = (e.target as HTMLElement).getBoundingClientRect();
        setHoveredEvent({
            event: evt,
            x: rect.left,
            y: rect.top - 10 // Slightly above
        });
    };

    const handleMouseLeave = () => {
        setHoveredEvent(null);
    };

    return (
        <div className="main-calendar">
            <div className="calendar-header">
                <h2>{formatMonthYear(currentDate)}</h2>
                <div className="weekday-header">
                    {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map(d => (
                        <div key={d} className="weekday-label">{d}</div>
                    ))}
                </div>
            </div>

            <div className="calendar-grid">
                {days.map((day, idx) => {
                    const dayEvents = events.filter(e => isSameDate(e.start, day));
                    const isCurrent = isCurrentMonth(day, currentDate);

                    return (
                        <div key={idx} className={`calendar-cell ${!isCurrent ? 'dimmed' : ''}`}>
                            <span className="day-number">{format(day, 'd')}</span>

                            <div className="events-container">
                                {dayEvents.map(evt => (
                                    <div
                                        key={evt.id}
                                        className="event-pill"
                                        style={{
                                            backgroundColor: getSourceColor(evt.sourceId) + '33', // 20% opacity 
                                            borderLeft: `3px solid ${getSourceColor(evt.sourceId)}`
                                        }}
                                        onMouseEnter={(e) => handleMouseEnter(e, evt)}
                                        onMouseLeave={handleMouseLeave}
                                    >
                                        {evt.title}
                                    </div>
                                ))}
                            </div>
                        </div>
                    );
                })}
            </div>

            {/* Custom Hover Tooltip */}
            {hoveredEvent && (
                <div
                    className="hover-tooltip"
                    style={{
                        top: hoveredEvent.y,
                        left: hoveredEvent.x,
                        borderColor: getSourceColor(hoveredEvent.event.sourceId)
                    }}
                >
                    <h4>{hoveredEvent.event.title}</h4>
                    <p className="tooltip-time">{format(hoveredEvent.event.start, 'h:mm a')} - {format(hoveredEvent.event.end, 'h:mm a')}</p>
                    {hoveredEvent.event.description && <p className="tooltip-desc">{hoveredEvent.event.description}</p>}
                </div>
            )}
        </div>
    );
}
