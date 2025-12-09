import './EventDetailsModal.css';

interface EventDetailsModalProps {
    event: { title: string; time?: string; description?: string } | null;
    onClose: () => void;
}

export function EventDetailsModal({ event, onClose }: EventDetailsModalProps) {
    if (!event) return null;

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal-content" onClick={(e) => e.stopPropagation()}>
                <div className="modal-header">
                    <h2>{event.title}</h2>
                    <button className="close-btn" onClick={onClose}>&times;</button>
                </div>
                <div className="modal-body">
                    <p className="event-time">{event.time || 'All Day'}</p>
                    <p className="event-desc">{event.description || 'No additional details.'}</p>
                </div>
                <div className="modal-footer">
                    <button className="delete-btn">Delete</button>
                    <button className="edit-btn">Edit</button>
                </div>
            </div>
        </div>
    );
}
