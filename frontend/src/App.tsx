import { useState } from 'react';
import './App.css';
import { HubView } from './components/HubView';
import { ExtendedCalendarView } from './components/ExtendedCalendarView';
import { TabSelector } from './components/TabSelector';

function App() {
  const [activeTab, setActiveTab] = useState<'hub' | 'extended'>('hub');

  return (
    <div className="app-container">
      <div className="view-container">
        {activeTab === 'hub' ? (
          <HubView />
        ) : (
          <ExtendedCalendarView />
        )}
      </div>

      <TabSelector activeTab={activeTab} onTabChange={setActiveTab} />
    </div>
  );
}

export default App;

