import './CalendarSidebar.css';
import { CalendarSource } from '../types';
import { format } from 'date-fns';

interface CalendarSidebarProps {
    currentDate: Date;
    sources: CalendarSource[];
    onToggleSource: (id: string) => void;
    onDateChange: (date: Date) => void;
}

export function CalendarSidebar({ currentDate, sources, onToggleSource, onDateChange }: CalendarSidebarProps) {

    const handleMonthChange = (offset: number) => {
        const newDate = new Date(currentDate);
        newDate.setMonth(newDate.getMonth() + offset);
        onDateChange(newDate);
    };

    return (
        <div className="calendar-sidebar">
            {/* Mini Calendar Widget */}
            <div className="mini-calendar card">
                <div className="mini-header">
                    <button onClick={() => handleMonthChange(-1)}>{'<'}</button>
                    <span>{format(currentDate, 'MMMM yyyy')}</span>
                    <button onClick={() => handleMonthChange(1)}>{'>'}</button>
                </div>
                <div className="mini-grid">
                    {['S', 'M', 'T', 'W', 'T', 'F', 'S'].map(d => <div key={d} className="day-label">{d}</div>)}
                    {/* Simplified Mini Grid (Just numbers for now to save complexity, full grid would reuse logic) */}
                    {Array.from({ length: 30 }).map((_, i) => (
                        <div key={i} className={`mini-day ${i === currentDate.getDate() - 1 ? 'today' : ''}`}>{i + 1}</div>
                    ))}
                </div>
            </div>

            {/* Account Toggles */}
            <div className="account-list">
                <h3 className="section-title">Accounts</h3>
                {sources.map((src) => (
                    <div key={src.id} className="account-item" onClick={() => onToggleSource(src.id)}>
                        <div
                            className="account-toggle"
                            style={{
                                backgroundColor: src.enabled ? src.color : 'transparent',
                                borderColor: src.color
                            }}
                        ></div>
                        <span className="account-email" style={{ opacity: src.enabled ? 1 : 0.5 }}>{src.email}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}
