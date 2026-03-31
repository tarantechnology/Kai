import { isTauriRuntime } from "./desktop";
import type { EventItem, KaiCommand, KaiState, LocalParserResponse, PaletteResult, TaskItem } from "./types";

const EVENT_TIMES = [
  "9:00 AM",
  "10:00 AM",
  "11:00 AM",
  "12:00 PM",
  "1:00 PM",
  "2:00 PM",
  "3:00 PM",
  "4:00 PM",
];

const titleCase = (value: string) =>
  value
    .split(" ")
    .filter(Boolean)
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(" ");

type CommandRecord = Record<string, unknown>;

const isRecord = (value: unknown): value is CommandRecord => typeof value === "object" && value !== null;
const isString = (value: unknown): value is string => typeof value === "string" && value.trim().length > 0;
const isNumber = (value: unknown): value is number => typeof value === "number" && Number.isFinite(value);

const normalizeConfidence = (value: unknown) => {
  // parser confidence is normalized here so the ui can rely on a stable 0..1 range.
  if (!isNumber(value)) {
    return 0.5;
  }

  return Math.max(0, Math.min(1, value));
};

const parseNaturalTimeLabel = (sourceText: string) => {
  // this is a tiny deterministic repair layer for simple reminder times like "at 7".
  const match = sourceText.match(/\bat\s+(\d{1,2})(?::(\d{2}))?\s*(am|pm)?\b/i);

  if (!match) {
    return null;
  }

  const hour = Number(match[1]);
  const minutes = (match[2] ?? "00").padStart(2, "0");
  const explicitMeridiem = match[3]?.toUpperCase();

  if (hour < 1 || hour > 12) {
    return null;
  }

  const meridiem = explicitMeridiem ?? (hour <= 7 ? "PM" : "AM");
  return `${hour}:${minutes} ${meridiem}`;
};

const extractReminderTitle = (sourceText: string) => {
  const normalized = sourceText
    .replace(/^\s*remind me\s+(to\s+)?/i, "")
    .replace(/\bat\s+\d{1,2}(?::\d{2})?\s*(am|pm)?\b/i, "")
    .replace(/\s+/g, " ")
    .trim();

  return normalized ? normalized.charAt(0).toUpperCase() + normalized.slice(1) : null;
};

const repairReminderCommand = (value: CommandRecord, sourceText: string, confidence: number): KaiCommand | null => {
  // the model can miss obvious reminder slots, so this patch-up keeps simple commands reliable.
  const title = isString(value.title) ? value.title : extractReminderTitle(sourceText);
  const datetimeLabel = isString(value.datetimeLabel) ? value.datetimeLabel : parseNaturalTimeLabel(sourceText);

  if (!title || !datetimeLabel) {
    return null;
  }

  return {
    type: "create_reminder",
    title,
    datetimeLabel,
    sourceText,
    confidence,
  };
};

const repairFromSourceText = (sourceText: string, confidence: number): KaiCommand | null => {
  if (/^\s*remind me\b/i.test(sourceText)) {
    return repairReminderCommand({ type: "create_reminder" }, sourceText, Math.max(confidence, 0.82));
  }

  return null;
};

const validateKaiCommand = (value: unknown, sourceText: string): KaiCommand | null => {
  // this keeps execution safe by converting only schema-valid payloads into app commands.
  if (!isRecord(value) || !isString(value.type)) {
    return null;
  }

  const confidence = normalizeConfidence(value.confidence);

  switch (value.type) {
    case "create_reminder":
      return repairReminderCommand(value, sourceText, confidence);
    case "create_event":
      if (!isString(value.title) || !isString(value.startLabel) || !isString(value.endLabel)) {
        return null;
      }

      return {
        type: "create_event",
        title: value.title,
        startLabel: value.startLabel,
        endLabel: value.endLabel,
        sourceText,
        confidence,
      };
    case "show_tasks":
      if (value.range !== "today" && value.range !== "tomorrow" && value.range !== "week") {
        return null;
      }

      return {
        type: "show_tasks",
        range: value.range,
        sourceText,
        confidence,
      };
    case "show_calendar":
      if (value.range !== "day" && value.range !== "tomorrow" && value.range !== "week") {
        return null;
      }

      return {
        type: "show_calendar",
        range: value.range,
        sourceText,
        confidence,
      };
    case "sync_canvas":
      if (value.range !== "today" && value.range !== "week" && value.range !== "all") {
        return null;
      }

      return {
        type: "sync_canvas",
        range: value.range,
        sourceText,
        confidence,
      };
    case "sync_google_calendar":
      if (
        value.range !== undefined &&
        value.range !== "today" &&
        value.range !== "week" &&
        value.range !== "all"
      ) {
        return null;
      }

      return {
        type: "sync_google_calendar",
        range: value.range,
        sourceText,
        confidence,
      };
    default:
      return null;
  }
};

export const parseCommand = async (sourceText: string): Promise<LocalParserResponse> => {
  if (!sourceText.trim()) {
    return {
      command: null,
      clarification: "Type a command for Kai to interpret.",
      confidence: 0,
    };
  }

  if (!isTauriRuntime()) {
    return {
      command: null,
      clarification: "Local parsing is only available inside the Tauri desktop runtime.",
      confidence: 0,
    };
  }

  const { invoke } = await import("@tauri-apps/api/core");
  const now = new Date().toLocaleString();

  // the frontend does not call ollama directly; it goes through tauri so native backends stay swappable.
  const response = (await invoke("parse_command_with_ollama", {
    input: sourceText,
    now,
  })) as {
    command: unknown;
    clarification?: string | null;
    backend?: string;
    model?: string;
    confidence: number;
  };

  const command = validateKaiCommand(response.command, sourceText);
  const repairedCommand = command ?? repairFromSourceText(sourceText, normalizeConfidence(response.confidence));

  return {
    command: repairedCommand,
    clarification:
      repairedCommand === null ? response.clarification ?? "Kai needs a little more detail to act on that." : null,
    backend: response.backend,
    model: response.model,
    confidence: normalizeConfidence(response.confidence),
  };
};

const buildEvent = (title: string, startLabel: string, endLabel: string): EventItem => ({
  id: `event-${crypto.randomUUID()}`,
  title,
  startLabel,
  endLabel,
  day: "Thu",
  dateLabel: "28",
  track: 3,
  accent: "blue",
  source: "kai",
});

const buildTask = (title: string, dueLabel: string): TaskItem => ({
  id: `task-${crypto.randomUUID()}`,
  title,
  dueLabel,
  dueAt: dueLabel,
  source: "kai",
  status: "todo",
  lane: "today",
});

const sortEvents = (events: EventItem[]) =>
  [...events].sort(
    (left, right) => EVENT_TIMES.indexOf(left.startLabel) - EVENT_TIMES.indexOf(right.startLabel),
  );

export const executeCommand = (
  command: KaiCommand | null,
  state: KaiState,
): { nextState: KaiState; result: PaletteResult } => {
  // this function is the deterministic half of the architecture: commands in, state updates out.
  if (!command) {
    return {
      nextState: state,
      result: {
        mode: "error",
        title: "Kai needs clarification",
        detail: "I couldn't confidently map that request to a supported command.",
      },
    };
  }

  switch (command.type) {
    case "show_tasks": {
      const tasks = state.tasks.filter((task) => task.lane === "today" && task.status === "todo");
      return {
        nextState: { ...state, activeView: "tasks" },
        result: {
          mode: "task_list",
          title: command.range === "today" ? "Today" : titleCase(command.range),
          detail: `${tasks.length} reminders`,
          tasks,
        },
      };
    }
    case "show_calendar":
      return {
        nextState: { ...state, activeView: "calendar" },
        result: {
          mode: "schedule",
          title: command.range === "day" ? "Today" : titleCase(command.range),
          detail: "Your schedule is up to date",
          events: state.events,
        },
      };
    case "sync_canvas": {
      const nextQueueItem = {
        id: `sync-${crypto.randomUUID()}`,
        type: "sync_canvas",
        status: "pending" as const,
        description: "Syncing with Canvas...",
      };

      return {
        nextState: { ...state, syncQueue: [nextQueueItem, ...state.syncQueue], activeView: "tasks" },
        result: {
          mode: "sync",
          title: "Canvas import queued",
          detail: `Pulling ${command.range} assignments into local tasks.`,
        },
      };
    }
    case "sync_google_calendar": {
      const nextQueueItem = {
        id: `sync-${crypto.randomUUID()}`,
        type: "sync_google_calendar",
        status: "pending" as const,
        description: "Syncing with Google Calendar...",
      };

      return {
        nextState: { ...state, syncQueue: [nextQueueItem, ...state.syncQueue] },
        result: {
          mode: "sync",
          title: "Google Calendar sync started",
          detail: "Local data stays available while sync completes in the background.",
        },
      };
    }
    case "create_reminder": {
      const task = buildTask(command.title, command.datetimeLabel);
      return {
        nextState: { ...state, tasks: [task, ...state.tasks], activeView: "today" },
        result: {
          mode: "success",
          title: "Reminder created",
          detail: `Reminder created for ${command.datetimeLabel}`,
        },
      };
    }
    case "create_event": {
      const event = buildEvent(command.title, command.startLabel, command.endLabel);
      const queueItem = {
        id: `sync-${crypto.randomUUID()}`,
        type: "create_google_event",
        status: "pending" as const,
        description: `Queued ${command.title} for Google Calendar sync`,
      };

      return {
        nextState: {
          ...state,
          activeView: "calendar",
          events: sortEvents([event, ...state.events]),
          syncQueue: [queueItem, ...state.syncQueue],
        },
        result: {
          mode: "success",
          title: "Time block created",
          detail: `${command.title} from ${command.startLabel} to ${command.endLabel}`,
        },
      };
    }
  }
};
