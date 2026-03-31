import { CalendarDays, CheckSquare, ChevronLeft, FileText, Settings, Square, SunMedium } from "lucide-react";
import type { ViewMode } from "../lib/types";

interface SidebarProps {
  activeView: ViewMode;
  collapsed: boolean;
  onSelect: (view: ViewMode) => void;
  onToggleCollapse: () => void;
}

const navItems = [
  { id: "today" as const, label: "Today", icon: SunMedium },
  { id: "calendar" as const, label: "Calendar", icon: CalendarDays },
  { id: "tasks" as const, label: "Tasks", icon: CheckSquare },
  { id: "notes" as const, label: "Notes", icon: FileText },
  { id: "settings" as const, label: "Settings", icon: Settings },
];

export const Sidebar = ({ activeView, collapsed, onSelect, onToggleCollapse }: SidebarProps) => (
  <aside className={`sidebar ${collapsed ? "is-collapsed" : ""}`}>
    <div className="brand-row">
      <div className="brand">
        <span className="brand-mark" />
        {!collapsed && <span>Kai</span>}
      </div>
      <button type="button" className="ghost-icon-button sidebar-toggle" onClick={onToggleCollapse} aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}>
        <ChevronLeft size={16} className={collapsed ? "is-collapsed" : ""} />
      </button>
    </div>
    {!collapsed && (
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
    )}
    {collapsed && (
      <nav className="nav-list nav-list-collapsed" aria-label="Primary">
        {navItems.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            className={`nav-item nav-item-collapsed ${activeView === id ? "is-active" : ""}`}
            onClick={() => onSelect(id)}
            type="button"
            aria-label={label}
            title={label}
          >
            <Icon size={18} />
          </button>
        ))}
      </nav>
    )}
    <button className={`sidebar-footer ${collapsed ? "sidebar-footer-collapsed" : ""}`} type="button" aria-label="Add new task">
      <Square size={16} />
      {!collapsed && <span>Add new task</span>}
    </button>
  </aside>
);
