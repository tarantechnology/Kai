import {
  CalendarDays,
  CheckSquare2,
  ChevronRight,
  Clock3,
  Plus,
  Search,
  ShieldCheck,
  Square,
  Trash2,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { GlassPanel } from "../../components/GlassPanel";
import { Sidebar } from "../../components/Sidebar";
import type {
  AssignmentItem,
  ConnectedAccount,
  EventItem,
  NoteItem,
  SyncQueueItem,
  TaskItem,
  ViewMode,
} from "../../lib/types";

interface DashboardProps {
  activeView: ViewMode;
  tasks: TaskItem[];
  events: EventItem[];
  assignments: AssignmentItem[];
  accounts: ConnectedAccount[];
  syncQueue: SyncQueueItem[];
  quickNoteDraft: string;
  onQuickNoteDraftChange: (value: string) => void;
  notes: NoteItem[];
  selectedNoteId: string | null;
  selectedNoteBody: string;
  onSelectNote: (noteId: string) => void;
  onCreateNote: () => void;
  onDeleteSelectedNote: () => void;
  onUpdateSelectedNote: (value: string) => void;
  sidebarCollapsed: boolean;
  authName?: string;
  authEmail?: string;
  onToggleSidebar: () => void;
  onStartGoogleAuth: () => void;
  onLogout: () => void;
  onSelectView: (view: ViewMode) => void;
}

const hours = ["8 AM", "9 AM", "10 AM", "11 AM", "12 PM", "1 PM", "2 PM", "3 PM", "4 PM", "5 PM", "6 PM"];
const weekDays = [
  { key: "Mon", label: "Mon", dateLabel: "25" },
  { key: "Tue", label: "Tue", dateLabel: "26" },
  { key: "Wed", label: "Wed", dateLabel: "27" },
  { key: "Thu", label: "Thu", dateLabel: "28" },
  { key: "Fri", label: "Fri", dateLabel: "29" },
  { key: "Sat", label: "Sat", dateLabel: "30" },
  { key: "Sun", label: "Sun", dateLabel: "31" },
] as const;
const CALENDAR_HOUR_HEIGHT = 56;
const CALENDAR_START_MINUTES = 8 * 60;
const WEEK_EVENT_PREVIEW_HEIGHT = 34;
const DAY_EVENT_PREVIEW_HEIGHT = 40;

const accentClass = {
  gold: "event-chip accent-gold",
  blue: "event-chip accent-blue",
  teal: "event-chip accent-teal",
};

const parseMinutes = (label: string) => {
  const match = label.match(/^(\d{1,2}):(\d{2})\s(AM|PM)$/);

  if (!match) {
    return CALENDAR_START_MINUTES;
  }

  let hour = Number(match[1]) % 12;
  const minutes = Number(match[2]);

  if (match[3] === "PM") {
    hour += 12;
  }

  return hour * 60 + minutes;
};

const formatUpdatedAt = (value: string) =>
  new Date(value).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });

const TodayView = ({
  tasks,
  events,
  quickNoteDraft,
  onQuickNoteDraftChange,
}: Pick<DashboardProps, "tasks" | "events" | "quickNoteDraft" | "onQuickNoteDraftChange">) => {
  const notesRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      notesRef.current?.focus();
      notesRef.current?.setSelectionRange(quickNoteDraft.length, quickNoteDraft.length);
    }, 60);

    return () => window.clearTimeout(timer);
  }, [quickNoteDraft.length]);

  return (
    <div className="view-grid today-layout">
      <section className="hero-block">
        <div>
          <h1>Good afternoon, Derek</h1>
          <p className="eyebrow">Thursday, March 27</p>
        </div>
        <div className="avatar">DW</div>
      </section>

      <GlassPanel className="content-card today-task-card">
        <div className="card-header">
          <h2>Reminders</h2>
          <span>{tasks.filter((task) => task.lane === "today").length} today</span>
        </div>
        <div className="today-task-list">
          {tasks
            .filter((task) => task.lane === "today")
            .map((task) => (
              <div key={task.id} className={`task-row today-task-row ${task.status === "done" ? "is-done" : ""}`}>
                {task.status === "done" ? <CheckSquare2 size={20} /> : <Square size={20} />}
                <span>{task.title}</span>
                <span>{task.dueLabel ?? task.dueAt ?? ""}</span>
              </div>
            ))}
        </div>
      </GlassPanel>

      <GlassPanel className="content-card notes-card">
        <textarea
          ref={notesRef}
          className="notes-pad"
          value={quickNoteDraft}
          onChange={(event) => onQuickNoteDraftChange(event.target.value)}
          placeholder="Capture a thought, reminder, or class note..."
        />
      </GlassPanel>

      <GlassPanel className="content-card today-assignment-card">
        <div className="card-header assignment-card-header">
          <div className="assignment-card-title">
            <CalendarDays size={18} />
            <h2>Calendar</h2>
          </div>
          <span>Today</span>
        </div>
        {events
          .filter((event) => event.day === "Thu")
          .map((event) => (
            <div key={event.id} className="assignment-row featured-assignment-row calendar-event-row">
              <div>
                <strong>{event.title}</strong>
                <p>
                  {event.startLabel} - {event.endLabel}
                </p>
              </div>
              <span>{event.source === "google" ? "Google" : "Kai"}</span>
            </div>
          ))}
      </GlassPanel>
    </div>
  );
};

const CalendarView = ({ events, assignments }: Pick<DashboardProps, "events" | "assignments">) => {
  const [calendarMode, setCalendarMode] = useState<"week" | "day">("week");
  const [selectedDay, setSelectedDay] = useState<(typeof weekDays)[number]["key"]>("Thu");
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);

  const eventLayouts = useMemo(
    () =>
      events.map((event) => {
        const startMinutes = parseMinutes(event.startLabel);
        const endMinutes = parseMinutes(event.endLabel);
        const duration = Math.max(30, endMinutes - startMinutes);
        const dayIndex = weekDays.findIndex((day) => day.key === event.day);

        return {
          ...event,
          dayIndex: Math.max(dayIndex, 0),
          top: ((startMinutes - CALENDAR_START_MINUTES) / 60) * CALENDAR_HOUR_HEIGHT,
          height: (duration / 60) * CALENDAR_HOUR_HEIGHT,
        };
      }),
    [events],
  );

  const dayEvents = eventLayouts.filter((event) => event.day === selectedDay);
  const selectedEvent =
    eventLayouts.find((event) => event.id === selectedEventId) ??
    dayEvents[0] ??
    eventLayouts.find((event) => event.day === "Thu") ??
    null;

  return (
    <div className="calendar-layout">
      <div className="calendar-toolbar calendar-toolbar-native">
        <div>
          <h1>Calendar</h1>
          <p>Week of March 25 - 31</p>
        </div>
        <div className="toolbar-group">
          <div className="toolbar-pill segmented-pill">
            <button type="button" className={calendarMode === "week" ? "is-active" : ""} onClick={() => setCalendarMode("week")}>
              Week
            </button>
            <button type="button" className={calendarMode === "day" ? "is-active" : ""} onClick={() => setCalendarMode("day")}>
              Day
            </button>
          </div>
          <button type="button" className="toolbar-pill toolbar-button" onClick={() => setSelectedDay("Thu")}>
            Today
          </button>
        </div>
      </div>

      <div className="calendar-main">
        <GlassPanel className="content-card calendar-card">
          {calendarMode === "week" ? (
            <div className="week-grid">
              <div className="week-grid-header week-grid-corner" />
              {weekDays.map((day) => (
                <button
                  key={day.key}
                  type="button"
                  className={`week-grid-header week-grid-day-button ${selectedDay === day.key ? "is-selected" : ""} ${day.key === "Thu" ? "is-today" : ""}`}
                  onClick={() => setSelectedDay(day.key)}
                >
                  <span>{day.label}</span>
                  <strong>{day.dateLabel}</strong>
                </button>
              ))}

              <div className="calendar-hours-column">
                {hours.map((hour) => (
                  <span key={hour}>{hour}</span>
                ))}
              </div>

              {weekDays.map((day) => (
                <div key={day.key} className={`calendar-day-column ${day.key === "Thu" ? "is-today" : ""} ${selectedDay === day.key ? "is-selected" : ""}`}>
                  {hours.map((hour) => (
                    <div key={hour} className="calendar-hour-cell" />
                  ))}
                  {eventLayouts
                    .filter((event) => event.day === day.key)
                    .map((event) => (
                    <button
                      key={event.id}
                      type="button"
                      className={accentClass[event.accent]}
                      onClick={() => setSelectedEventId(event.id)}
                      style={{
                        top: `${event.top + 6}px`,
                        height: `${WEEK_EVENT_PREVIEW_HEIGHT}px`,
                        left: `${8 + event.track * 4}px`,
                        right: `${8 + Math.max(0, 2 - event.track) * 4}px`,
                      }}
                    >
                      <strong>{event.title}</strong>
                    </button>
                  ))}
                </div>
              ))}
            </div>
          ) : (
            <div className="day-grid">
              <div className="day-grid-header">
                <strong>{weekDays.find((day) => day.key === selectedDay)?.label}</strong>
                <span>{weekDays.find((day) => day.key === selectedDay)?.dateLabel}</span>
              </div>
              <div className="day-grid-body">
                <div className="calendar-hours-column day-hours-column">
                  {hours.map((hour) => (
                    <span key={hour}>{hour}</span>
                  ))}
                </div>
                <div className="day-events-column">
                  {hours.map((hour) => (
                    <div key={hour} className="calendar-hour-cell day-hour-cell" />
                  ))}
                  {dayEvents.map((event) => (
                    <button
                      key={event.id}
                      type="button"
                      className={accentClass[event.accent]}
                      onClick={() => setSelectedEventId(event.id)}
                      style={{
                        top: `${event.top + 8}px`,
                        height: `${DAY_EVENT_PREVIEW_HEIGHT}px`,
                        left: "14px",
                        right: "14px",
                      }}
                    >
                      <strong>{event.title}</strong>
                    </button>
                  ))}
                </div>
              </div>
            </div>
          )}
        </GlassPanel>

        <div className="calendar-rail">
          <GlassPanel className="content-card calendar-sidebar-card calendar-detail-card">
            {selectedEvent ? (
              <div className="calendar-popover">
                <div className="calendar-popover-header">
                  <div className={`calendar-popover-dot accent-dot-${selectedEvent.accent}`} />
                  <div>
                    <h2>{selectedEvent.title}</h2>
                    <p>
                      {selectedEvent.day} {selectedEvent.dateLabel}
                    </p>
                  </div>
                </div>
                <div className="calendar-popover-meta">
                  <Clock3 size={15} />
                  <span>
                    {selectedEvent.startLabel} - {selectedEvent.endLabel}
                  </span>
                </div>
                <div className="calendar-popover-meta">
                  <CalendarDays size={15} />
                  <span>{selectedEvent.source === "google" ? "Google Calendar" : "Kai Calendar"}</span>
                </div>
                <p className="calendar-popover-description">
                  {selectedEvent.title} is scheduled on {selectedEvent.day} and remains visible in your weekly calendar.
                </p>
              </div>
            ) : null}
          </GlassPanel>

          <GlassPanel className="content-card calendar-sidebar-card calendar-upcoming-card">
            <div className="card-header">
              <h2>Upcoming</h2>
              <span>{assignments.length + events.length} items</span>
            </div>
            <div className="calendar-sidebar-list">
              {events.slice(0, 4).map((event) => (
                <div key={event.id} className="calendar-sidebar-row">
                  <div>
                    <strong>{event.title}</strong>
                    <p>
                      {event.day} {event.startLabel} - {event.endLabel}
                    </p>
                  </div>
                  <span>{event.source}</span>
                </div>
              ))}
              {assignments.map((assignment) => (
                <div key={assignment.id} className="calendar-sidebar-row">
                  <div>
                    <strong>{assignment.title}</strong>
                    <p>{assignment.subtitle}</p>
                  </div>
                  <span>{assignment.dueLabel}</span>
                </div>
              ))}
            </div>
          </GlassPanel>
        </div>
      </div>
    </div>
  );
};

const TasksView = ({ tasks, syncQueue }: Pick<DashboardProps, "tasks" | "syncQueue">) => (
  <div className="split-view">
    <section>
      <div className="section-heading">
        <h1>Tasks</h1>
        <div className="toolbar-pill">Today</div>
      </div>
      <div className="list-section">
        <h2>Today</h2>
        <GlassPanel className="content-card">
          <div className="task-list">
            {tasks
              .filter((task) => task.lane === "today")
              .map((task) => (
                <div key={task.id} className={`task-row ${task.status === "done" ? "is-done" : ""}`}>
                  {task.status === "done" ? <CheckSquare2 size={18} /> : <Square size={18} />}
                  <div>
                    <span>{task.title}</span>
                    {task.dueAt && <p>{task.dueAt}</p>}
                  </div>
                  <span>{task.dueLabel ?? task.dueAt ?? ""}</span>
                </div>
              ))}
          </div>
        </GlassPanel>
      </div>
      <div className="list-section">
        <h2>Upcoming</h2>
        <GlassPanel className="content-card">
          <div className="task-list">
            {tasks
              .filter((task) => task.lane === "upcoming")
              .map((task) => (
                <div key={task.id} className="task-row">
                  <CalendarDays size={18} />
                  <span>{task.title}</span>
                  <span>{task.dueLabel ?? ""}</span>
                </div>
              ))}
          </div>
        </GlassPanel>
      </div>
    </section>

    <section>
      <div className="section-heading">
        <h1>Activity</h1>
        <div className="toolbar-pill">{syncQueue.length} sync items</div>
      </div>
      <GlassPanel className="content-card">
        {syncQueue.map((item) => (
          <div key={item.id} className="settings-row">
            <div>
              <strong>{item.type}</strong>
              <p>{item.description}</p>
            </div>
            <span className="status-pill">{item.status}</span>
          </div>
        ))}
      </GlassPanel>
    </section>
  </div>
);

const NotesView = ({
  notes,
  selectedNoteId,
  selectedNoteBody,
  onSelectNote,
  onCreateNote,
  onDeleteSelectedNote,
  onUpdateSelectedNote,
}: Pick<
  DashboardProps,
  | "notes"
  | "selectedNoteId"
  | "selectedNoteBody"
  | "onSelectNote"
  | "onCreateNote"
  | "onDeleteSelectedNote"
  | "onUpdateSelectedNote"
>) => {
  const notesEditorRef = useRef<HTMLTextAreaElement>(null);
  const [query, setQuery] = useState("");

  const filteredNotes = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();

    if (!normalizedQuery) {
      return notes;
    }

    return notes.filter((note) =>
      `${note.title}\n${note.body}`.toLowerCase().includes(normalizedQuery),
    );
  }, [notes, query]);
  const activeNote = notes.find((note) => note.id === selectedNoteId) ?? null;

  useEffect(() => {
    const timer = window.setTimeout(() => {
      notesEditorRef.current?.focus();
    }, 60);

    return () => window.clearTimeout(timer);
  }, [selectedNoteId]);

  return (
    <div className="notes-layout">
      <GlassPanel className="content-card notes-list-card">
        <div className="card-header notes-list-header">
          <div className="notes-toolbar">
            <label className="notes-search">
              <Search size={14} />
              <input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search"
              />
            </label>
          </div>
          <div className="notes-toolbar-actions">
            <button type="button" className="ghost-icon-button" onClick={onCreateNote} aria-label="Create note">
              <Plus size={16} />
            </button>
            <button type="button" className="ghost-icon-button" onClick={onDeleteSelectedNote} aria-label="Delete note">
              <Trash2 size={16} />
            </button>
          </div>
        </div>
        <div className="notes-list">
          {filteredNotes.map((note) => (
            <button
              key={note.id}
              type="button"
              className={`note-list-item ${selectedNoteId === note.id ? "is-active" : ""}`}
              onClick={() => onSelectNote(note.id)}
            >
              <strong>{note.title}</strong>
              <p>{note.body.trim() || "Empty note"}</p>
              <span>{formatUpdatedAt(note.updatedAt)}</span>
            </button>
          ))}
        </div>
      </GlassPanel>

      <GlassPanel className="content-card notes-editor-card">
        <div className="card-header notes-editor-header">
          <div>
            <h2>{activeNote?.title ?? "Untitled Note"}</h2>
            <span>{activeNote?.updatedAt ? `Updated ${formatUpdatedAt(activeNote.updatedAt)}` : "New note"}</span>
          </div>
        </div>
        <textarea
          ref={notesEditorRef}
          className="notes-editor"
          value={selectedNoteBody}
          onChange={(event) => onUpdateSelectedNote(event.target.value)}
          placeholder="Start writing..."
        />
      </GlassPanel>
    </div>
  );
};

const SettingsView = ({
  accounts,
  syncQueue,
  authName,
  authEmail,
  onStartGoogleAuth,
  onLogout,
}: Pick<DashboardProps, "accounts" | "syncQueue" | "authName" | "authEmail" | "onStartGoogleAuth" | "onLogout">) => (
  <div className="split-view">
    <section>
      <div className="section-heading">
        <h1>Settings</h1>
        <div className="toolbar-pill">Local-first</div>
      </div>
      <GlassPanel className="content-card">
        <div className="settings-row profile">
          <div className="avatar">DW</div>
          <div>
            <strong>{authName ?? "Kai user"}</strong>
            <p>{authEmail ?? "Signed in"}</p>
          </div>
          <button type="button" className="toolbar-pill toolbar-button" onClick={onLogout}>
            Sign out
          </button>
        </div>
        {accounts.map((account) => (
          <div key={account.id} className="settings-row">
            <div>
              <strong>{account.label}</strong>
              <p>{account.email ?? (account.status === "disconnected" ? "Not connected" : "Connected")}</p>
            </div>
            {account.provider === "google" && account.status !== "connected" ? (
              <button type="button" className="toolbar-pill toolbar-button" onClick={onStartGoogleAuth}>
                Connect
              </button>
            ) : (
              <span className="status-pill">{account.status}</span>
            )}
          </div>
        ))}
        <div className="settings-row">
          <div>
            <strong>Notification Settings</strong>
            <p>System reminders and quiet hours</p>
          </div>
          <ChevronRight size={18} />
        </div>
        <div className="settings-row">
          <div>
            <strong>Data & Privacy</strong>
            <p>Local storage, keychain-backed tokens, action history</p>
          </div>
          <ShieldCheck size={18} />
        </div>
      </GlassPanel>
    </section>

    <section>
      <div className="section-heading">
        <h1>Sync Status</h1>
        <div className="toolbar-pill">Background</div>
      </div>
      <GlassPanel className="content-card">
        {syncQueue.map((item) => (
          <div key={item.id} className="settings-row">
            <div>
              <strong>{item.type}</strong>
              <p>{item.description}</p>
            </div>
            <span className="status-pill">{item.status}</span>
          </div>
        ))}
      </GlassPanel>
    </section>
  </div>
);

export const Dashboard = ({
  activeView,
  tasks,
  events,
  assignments,
  accounts,
  syncQueue,
  quickNoteDraft,
  onQuickNoteDraftChange,
  notes,
  selectedNoteId,
  selectedNoteBody,
  onSelectNote,
  onCreateNote,
  onDeleteSelectedNote,
  onUpdateSelectedNote,
  sidebarCollapsed,
  authName,
  authEmail,
  onToggleSidebar,
  onStartGoogleAuth,
  onLogout,
  onSelectView,
}: DashboardProps) => (
  <GlassPanel className={`dashboard-shell ${sidebarCollapsed ? "sidebar-collapsed" : ""}`}>
    <Sidebar activeView={activeView} collapsed={sidebarCollapsed} onSelect={onSelectView} onToggleCollapse={onToggleSidebar} />
    <main className="dashboard-content">
      <div className="dashboard-topbar">
        <div className="toolbar-pill">Kai</div>
      </div>
      {activeView === "today" && (
        <TodayView
          tasks={tasks}
          events={events}
          quickNoteDraft={quickNoteDraft}
          onQuickNoteDraftChange={onQuickNoteDraftChange}
        />
      )}
      {activeView === "calendar" && <CalendarView events={events} assignments={assignments} />}
      {activeView === "tasks" && <TasksView tasks={tasks} syncQueue={syncQueue} />}
      {activeView === "notes" && (
        <NotesView
          notes={notes}
          selectedNoteId={selectedNoteId}
          selectedNoteBody={selectedNoteBody}
          onSelectNote={onSelectNote}
          onCreateNote={onCreateNote}
          onDeleteSelectedNote={onDeleteSelectedNote}
          onUpdateSelectedNote={onUpdateSelectedNote}
        />
      )}
      {activeView === "settings" && (
        <SettingsView
          accounts={accounts}
          syncQueue={syncQueue}
          authName={authName}
          authEmail={authEmail}
          onStartGoogleAuth={onStartGoogleAuth}
          onLogout={onLogout}
        />
      )}
    </main>
  </GlassPanel>
);
