export type ViewMode = "today" | "calendar" | "tasks" | "notes" | "settings";
export type DesktopSurface = "palette" | "dashboard";
export type PaletteMode =
  | "idle"
  | "loading"
  | "success"
  | "task_list"
  | "schedule"
  | "sync"
  | "error";

export type TaskStatus = "todo" | "done";
export type SourceType = "kai" | "google" | "canvas";

export interface TaskItem {
  id: string;
  title: string;
  dueLabel?: string;
  dueAt?: string;
  source: SourceType;
  status: TaskStatus;
  lane: "today" | "upcoming";
}

export interface EventItem {
  id: string;
  title: string;
  startLabel: string;
  endLabel: string;
  day: "Mon" | "Tue" | "Wed" | "Thu" | "Fri" | "Sat" | "Sun";
  dateLabel: string;
  track: number;
  accent: "gold" | "blue" | "teal";
  source: SourceType;
}

export interface NoteItem {
  id: string;
  title: string;
  body: string;
  updatedAt: string;
}

export interface AssignmentItem {
  id: string;
  title: string;
  subtitle: string;
  dueLabel: string;
  source: "canvas";
}

export interface ConnectedAccount {
  id: string;
  provider: "google" | "canvas";
  label: string;
  email?: string;
  status: "connected" | "syncing" | "disconnected" | "error";
}

export type KaiCommand =
  | {
      type: "create_reminder";
      title: string;
      datetimeLabel: string;
      sourceText: string;
      confidence: number;
    }
  | {
      type: "create_event";
      title: string;
      startLabel: string;
      endLabel: string;
      sourceText: string;
      confidence: number;
    }
  | {
      type: "show_tasks";
      range: "today" | "tomorrow" | "week";
      sourceText: string;
      confidence: number;
    }
  | {
      type: "show_calendar";
      range: "day" | "tomorrow" | "week";
      sourceText: string;
      confidence: number;
    }
  | {
      type: "sync_canvas";
      range: "today" | "week" | "all";
      sourceText: string;
      confidence: number;
    }
  | {
      type: "sync_google_calendar";
      range?: "today" | "week" | "all";
      sourceText: string;
      confidence: number;
    };

export interface LocalParserResponse {
  command: KaiCommand | null;
  clarification?: string | null;
  backend?: string;
  model?: string;
  confidence: number;
}

export interface PaletteResult {
  mode: PaletteMode;
  title?: string;
  detail?: string;
  tasks?: TaskItem[];
  events?: EventItem[];
}

export interface SyncQueueItem {
  id: string;
  type: string;
  status: "pending" | "synced";
  description: string;
}

export interface KaiState {
  activeView: ViewMode;
  paletteQuery: string;
  paletteResult: PaletteResult;
  tasks: TaskItem[];
  events: EventItem[];
  assignments: AssignmentItem[];
  accounts: ConnectedAccount[];
  syncQueue: SyncQueueItem[];
}
