import { useEffect } from 'react';
import './TabSelector.css';

interface TabSelectorProps {
    activeTab: 'hub' | 'extended';
    onTabChange: (tab: 'hub' | 'extended') => void;
}

export function TabSelector({ activeTab, onTabChange }: TabSelectorProps) {

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // Cmd+1 for Hub, Cmd+2 for Extended
            if ((e.metaKey || e.ctrlKey) && e.key === '1') {
                onTabChange('hub');
            }
            if ((e.metaKey || e.ctrlKey) && e.key === '2') {
                onTabChange('extended');
            }
        };

        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [onTabChange]);

    return (
        <div className="tab-selector-container">
            <div className="tab-selector">
                <button
                    className={`tab-btn ${activeTab === 'hub' ? 'active' : ''}`}
                    onClick={() => onTabChange('hub')}
                >
                    <span className="tab-label">Hub</span>
                    <span className="shortcut-hint">⌘1</span>
                </button>
                <button
                    className={`tab-btn ${activeTab === 'extended' ? 'active' : ''}`}
                    onClick={() => onTabChange('extended')}
                >
                    <span className="tab-label">Calendar</span>
                    <span className="shortcut-hint">⌘2</span>
                </button>

                {/* Sliding Indicator */}
                <div className={`tab-indicator ${activeTab}`} />
            </div>
        </div>
    );
}
