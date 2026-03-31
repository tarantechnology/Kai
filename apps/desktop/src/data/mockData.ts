import type {
  AssignmentItem,
  ConnectedAccount,
  EventItem,
  KaiState,
  SyncQueueItem,
  TaskItem,
} from "../lib/types";

const tasks: TaskItem[] = [
  {
    id: "task-1",
    title: "Finish project report",
    dueLabel: "25m left",
    dueAt: "3:00 PM",
    source: "kai",
    status: "done",
    lane: "today",
  },
  {
    id: "task-2",
    title: "Pick up dry cleaning",
    source: "kai",
    status: "todo",
    lane: "today",
  },
  {
    id: "task-3",
    title: "Call back Jake",
    source: "kai",
    status: "todo",
    lane: "today",
  },
  {
    id: "task-4",
    title: "Workout",
    dueAt: "6:00 PM",
    source: "kai",
    status: "todo",
    lane: "today",
  },
  {
    id: "task-5",
    title: "Dentist",
    dueLabel: "Fri 9 AM",
    source: "canvas",
    status: "todo",
    lane: "upcoming",
  },
  {
    id: "task-6",
    title: "Buy gift for mom",
    source: "kai",
    status: "todo",
    lane: "upcoming",
  },
];

const events: EventItem[] = [
  {
    id: "event-1",
    title: "Team Standup",
    startLabel: "10:00 AM",
    endLabel: "10:30 AM",
    day: "Mon",
    dateLabel: "25",
    track: 0,
    accent: "gold",
    source: "google",
  },
  {
    id: "event-2",
    title: "Call with Emily",
    startLabel: "11:00 AM",
    endLabel: "11:30 AM",
    day: "Thu",
    dateLabel: "28",
    track: 2,
    accent: "blue",
    source: "google",
  },
  {
    id: "event-3",
    title: "Lunch with Dan",
    startLabel: "12:00 PM",
    endLabel: "12:45 PM",
    day: "Tue",
    dateLabel: "26",
    track: 0,
    accent: "teal",
    source: "google",
  },
  {
    id: "event-4",
    title: "Team Project Meeting",
    startLabel: "1:00 PM",
    endLabel: "2:00 PM",
    day: "Thu",
    dateLabel: "28",
    track: 1,
    accent: "blue",
    source: "google",
  },
  {
    id: "event-5",
    title: "Workout",
    startLabel: "2:00 PM",
    endLabel: "2:45 PM",
    day: "Fri",
    dateLabel: "29",
    track: 0,
    accent: "teal",
    source: "kai",
  },
];

const assignments: AssignmentItem[] = [
  {
    id: "assignment-1",
    title: "Homework 4",
    subtitle: "Canvas",
    dueLabel: "Tomorrow",
    source: "canvas",
  },
];

const accounts: ConnectedAccount[] = [
  {
    id: "account-google",
    provider: "google",
    label: "Google",
    status: "disconnected",
  },
  {
    id: "account-canvas",
    provider: "canvas",
    label: "Canvas",
    status: "disconnected",
  },
];

const syncQueue: SyncQueueItem[] = [
  {
    id: "sync-1",
    type: "sync_google_calendar",
    status: "pending",
    description: "Syncing with Google Calendar...",
  },
];

export const initialKaiState: KaiState = {
  activeView: "today",
  paletteQuery: "",
  paletteResult: {
    mode: "idle",
    title: "Quick command",
    detail: "Create reminders, block time, or sync providers.",
  },
  tasks,
  events,
  assignments,
  accounts,
  syncQueue,
};
