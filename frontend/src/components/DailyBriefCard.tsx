import './DailyBriefCard.css';

export function DailyBriefCard() {
    const currentDate = new Date().toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' });

    // Dummy data for now
    const events = [
        { time: '10:00 AM', title: 'Team Sync', color: '#ff5f57' },
        { time: '1:00 PM', title: 'Deep Work', color: '#febc2e' },
        { time: '3:30 PM', title: 'Client Call', color: '#28c840' }
    ];

    return (
        <div className="card daily-brief-card">
            <div className="daily-header">
                <h2>{currentDate}</h2>
                <span className="subtitle">Here is your schedule for today</span>
            </div>

            <div className="events-list">
                {events.map((evt, idx) => (
                    <div key={idx} className="event-item">
                        <div className="event-time">{evt.time}</div>
                        <div className="event-marker" style={{ backgroundColor: evt.color }}></div>
                        <div className="event-title">{evt.title}</div>
                    </div>
                ))}
                {events.length === 0 && <p className="no-events">No events scheduled.</p>}
            </div>
        </div>
    );
}
