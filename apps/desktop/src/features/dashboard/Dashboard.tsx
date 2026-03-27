import { CalendarDays, CheckSquare2, ChevronRight, ShieldCheck, Square } from "lucide-react";
import { useEffect, useRef } from "react";
import { GlassPanel } from "../../components/GlassPanel";
import { Sidebar } from "../../components/Sidebar";
import type { AssignmentItem, ConnectedAccount, EventItem, SyncQueueItem, TaskItem, ViewMode } from "../../lib/types";

interface DashboardProps {
  activeView: ViewMode;
  tasks: TaskItem[];
  events: EventItem[];
  assignments: AssignmentItem[];
  accounts: ConnectedAccount[];
  syncQueue: SyncQueueItem[];
  notesDraft: string;
  onNotesDraftChange: (value: string) => void;
  onSelectView: (view: ViewMode) => void;
}

const hours = ["9:00 AM", "10:00 AM", "11:00 AM", "12:00 PM", "1:00 PM", "2:00 PM", "3:00 PM"];
const CALENDAR_ROW_HEIGHT = 54;
const CALENDAR_TOP_OFFSET = 20;

const accentClass = {
  gold: "event-chip accent-gold",
  blue: "event-chip accent-blue",
  teal: "event-chip accent-teal",
};

const TodayView = ({
  tasks,
  events,
  notesDraft,
  onNotesDraftChange,
}: Pick<DashboardProps, "tasks" | "events" | "notesDraft" | "onNotesDraftChange">) => {
  const notesRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      notesRef.current?.focus();
      notesRef.current?.setSelectionRange(notesDraft.length, notesDraft.length);
    }, 60);

    return () => window.clearTimeout(timer);
  }, [notesDraft.length]);

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
          <span>Today</span>
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
          value={notesDraft}
          onChange={(event) => onNotesDraftChange(event.target.value)}
          placeholder="Write a quick thought, task, or reminder..."
        />
      </GlassPanel>

      <GlassPanel className="content-card assignment-card today-assignment-card">
        <div className="card-header assignment-card-header">
          <div className="assignment-card-title">
            <CalendarDays size={18} />
            <h2>Calendar</h2>
          </div>
          <ChevronRight size={18} />
        </div>
        {events.map((event) => (
          <div key={event.id} className="assignment-row featured-assignment-row calendar-event-row">
            <div>
              <strong>{event.title}</strong>
              <p>{event.startLabel} - {event.endLabel}</p>
            </div>
            <span>{event.source === "google" ? "Google" : "Kai"}</span>
          </div>
        ))}
      </GlassPanel>
    </div>
  );
};

const CalendarView = ({ events, assignments }: Pick<DashboardProps, "events" | "assignments">) => (
  <div className="calendar-layout">
    <div className="calendar-toolbar">
      <div>
        <h1>Apr 25 - May 1</h1>
        <p>Week view</p>
      </div>
      <div className="toolbar-pill">Week · Today · April 27</div>
    </div>

    <GlassPanel className="content-card calendar-card">
      <div className="calendar-grid">
        <div className="calendar-times">
          {hours.map((hour) => (
            <span key={hour}>{hour}</span>
          ))}
        </div>
        <div className="calendar-track">
          {hours.map((hour) => (
            <div key={hour} className="calendar-row" />
          ))}
          <div className="time-now" />
          {events.map((event) => (
            <div
              key={event.id}
              className={accentClass[event.accent]}
              style={{
                top: `${hours.indexOf(event.startLabel) * CALENDAR_ROW_HEIGHT + CALENDAR_TOP_OFFSET}px`,
                left: `${event.track * 22 + 14}%`,
              }}
            >
              {event.title}
            </div>
          ))}
        </div>
      </div>
      <div className="assignment-strip">
        <div className="card-header">
          <h2>Assignments</h2>
          <ChevronRight size={18} />
        </div>
        {assignments.map((assignment) => (
          <div key={assignment.id} className="assignment-row compact">
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
);

const TasksView = ({ tasks, syncQueue }: Pick<DashboardProps, "tasks" | "syncQueue">) => (
  <div className="split-view">
    <section>
      <h1>Tasks</h1>
      <div className="list-section">
        <h2>Today</h2>
        <GlassPanel className="content-card">
          <div className="task-list">
            {tasks
              .filter((task) => task.lane === "today")
              .map((task) => (
                <div key={task.id} className={`task-row ${task.status === "done" ? "is-done" : ""}`}>
                  <CheckSquare2 size={18} />
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
      <h1>Sync</h1>
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

const SettingsView = ({ accounts, syncQueue }: Pick<DashboardProps, "accounts" | "syncQueue">) => (
  <div className="split-view">
    <section>
      <h1>Settings</h1>
      <GlassPanel className="content-card">
        <div className="settings-row profile">
          <div className="avatar">DW</div>
          <div>
            <strong>Derek Williams</strong>
            <p>derik@example.com</p>
          </div>
          <ChevronRight size={18} />
        </div>
        {accounts.map((account) => (
          <div key={account.id} className="settings-row">
            <div>
              <strong>{account.label}</strong>
              <p>{account.email ?? "Connected"}</p>
            </div>
            <span className="status-pill">{account.status}</span>
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
            <p>Local-first storage, keychain-backed tokens, action log</p>
          </div>
          <ShieldCheck size={18} />
        </div>
      </GlassPanel>
    </section>

    <section>
      <h1>Sync Status</h1>
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
  notesDraft,
  onNotesDraftChange,
  onSelectView,
}: DashboardProps) => (
  <GlassPanel className="dashboard-shell">
    <Sidebar activeView={activeView} onSelect={onSelectView} />
    <main className="dashboard-content">
      <div className="dashboard-topbar">
        <div className="toolbar-pill">Kai</div>
      </div>
      {activeView === "today" && (
        <TodayView
          tasks={tasks}
          events={events}
          notesDraft={notesDraft}
          onNotesDraftChange={onNotesDraftChange}
        />
      )}
      {activeView === "calendar" && <CalendarView events={events} assignments={assignments} />}
      {activeView === "tasks" && <TasksView tasks={tasks} syncQueue={syncQueue} />}
      {activeView === "settings" && <SettingsView accounts={accounts} syncQueue={syncQueue} />}
    </main>
  </GlassPanel>
);
