import { useState } from 'react';
import './StickyNotesCard.css';

export function StickyNotesCard() {
    const [notes, setNotes] = useState<string>('Remember to buy milk...');

    return (
        <div className="card sticky-notes-card">
            <div className="sticky-header">
                <h2>Sticky Notes</h2>
            </div>
            <textarea
                className="sticky-textarea"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Type a quick note..."
            />
        </div>
    );
}
