import './HubView.css';
import { DailyBriefCard } from './DailyBriefCard';
import { StickyNotesCard } from './StickyNotesCard';
import { ChatbotCard } from './ChatbotCard';

export function HubView() {
    return (
        <div className="hub-grid">
            <div className="hub-column left-column">
                <DailyBriefCard />
            </div>

            <div className="hub-column right-column">
                <StickyNotesCard />
                <ChatbotCard />
            </div>
        </div>
    );
}
