import { CalendarDays, CheckSquare, Settings, Square, SunMedium } from "lucide-react";
import type { ViewMode } from "../lib/types";

interface SidebarProps {
  activeView: ViewMode;
  onSelect: (view: ViewMode) => void;
}

const navItems = [
  { id: "today" as const, label: "Today", icon: SunMedium },
  { id: "calendar" as const, label: "Calendar", icon: CalendarDays },
  { id: "tasks" as const, label: "Tasks", icon: CheckSquare },
  { id: "settings" as const, label: "Settings", icon: Settings },
];

export const Sidebar = ({ activeView, onSelect }: SidebarProps) => (
  <aside className="sidebar">
    <div className="brand">
      <span className="brand-mark" />
      <span>Kai</span>
    </div>
    <nav className="nav-list" aria-label="Primary">
      {navItems.map(({ id, label, icon: Icon }) => (
        <button
          key={id}
          className={`nav-item ${activeView === id ? "is-active" : ""}`}
          onClick={() => onSelect(id)}
          type="button"
        >
          <Icon size={18} />
          <span>{label}</span>
        </button>
      ))}
    </nav>
    <button className="sidebar-footer" type="button">
      <Square size={16} />
      <span>Add new task</span>
    </button>
  </aside>
);

