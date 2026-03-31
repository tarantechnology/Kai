import type { NoteItem } from "./types";

const NOTES_STORAGE_KEY = "kai.notes.v1";
const QUICK_NOTE_STORAGE_KEY = "kai.quick-note.v1";

const isBrowser = () => typeof window !== "undefined";

export const loadNotes = (): NoteItem[] => {
  if (!isBrowser()) {
    return [];
  }

  const raw = window.localStorage.getItem(NOTES_STORAGE_KEY);

  if (!raw) {
    return [];
  }

  try {
    const parsed = JSON.parse(raw) as unknown;

    if (!Array.isArray(parsed)) {
      return [];
    }

    return parsed.filter(
      (note): note is NoteItem =>
        typeof note === "object" &&
        note !== null &&
        typeof note.id === "string" &&
        typeof note.title === "string" &&
        typeof note.body === "string" &&
        typeof note.updatedAt === "string",
    );
  } catch {
    return [];
  }
};

export const saveNotes = (notes: NoteItem[]) => {
  if (!isBrowser()) {
    return;
  }

  window.localStorage.setItem(NOTES_STORAGE_KEY, JSON.stringify(notes));
};

export const loadQuickNote = () => {
  if (!isBrowser()) {
    return "";
  }

  return window.localStorage.getItem(QUICK_NOTE_STORAGE_KEY) ?? "";
};

export const saveQuickNote = (value: string) => {
  if (!isBrowser()) {
    return;
  }

  window.localStorage.setItem(QUICK_NOTE_STORAGE_KEY, value);
};
