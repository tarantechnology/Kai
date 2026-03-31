import { useEffect, useMemo, useRef, useState } from "react";
import { CommandPalette } from "../features/command-center/CommandPalette";
import { Dashboard } from "../features/dashboard/Dashboard";
import { initialKaiState } from "../data/mockData";
import { executeCommand, parseCommand } from "../lib/commandEngine";
import { bindSurfaceListener, isTauriRuntime, setPaletteHeight, warmLocalParser } from "../lib/desktop";
import { loadNotes, loadQuickNote, saveNotes, saveQuickNote } from "../lib/persistence";
import type { DesktopSurface, KaiState, NoteItem, PaletteResult, ViewMode } from "../lib/types";

const loadingState: PaletteResult = {
  mode: "loading",
  title: "Understanding command",
  detail: "Executing locally",
};

const PALETTE_COLLAPSED_HEIGHT = 88;
const PALETTE_MIN_EXPANDED_HEIGHT = 170;
const PALETTE_MAX_HEIGHT = 520;

const buildInitialNotes = (): NoteItem[] => {
  const persisted = loadNotes();

  if (persisted.length > 0) {
    return [...persisted].sort((left, right) => right.updatedAt.localeCompare(left.updatedAt));
  }

  return [];
};

const deriveNoteTitle = (body: string) => {
  const firstLine = body
    .split("\n")
    .map((line) => line.trim())
    .find(Boolean);

  return firstLine ? firstLine.slice(0, 48) : "Untitled Note";
};

export const App = () => {
  const [state, setState] = useState<KaiState>(initialKaiState);
  const [surface, setSurface] = useState<DesktopSurface>("palette");
  const [browserVisible, setBrowserVisible] = useState(true);
  const [quickNoteDraft, setQuickNoteDraft] = useState(() => loadQuickNote());
  const [notes, setNotes] = useState<NoteItem[]>(() => buildInitialNotes());
  const [selectedNoteId, setSelectedNoteId] = useState<string | null>(null);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const surfaceRef = useRef<DesktopSurface>("palette");
  const paletteRef = useRef<HTMLDivElement>(null);

  const sortedNotes = useMemo(
    () => [...notes].sort((left, right) => right.updatedAt.localeCompare(left.updatedAt)),
    [notes],
  );
  const selectedNote = sortedNotes.find((note) => note.id === selectedNoteId) ?? sortedNotes[0] ?? null;

  const handleSubmit = () => {
    const sourceText = state.paletteQuery.trim();

    if (!sourceText) {
      return;
    }

    setState((current) => ({ ...current, paletteResult: loadingState }));

    void parseCommand(sourceText)
      .then((parsed) => {
        setState((current) => {
          if (!parsed.command) {
            return {
              ...current,
              paletteResult: {
                mode: "error",
                title: "Kai needs clarification",
                detail: parsed.clarification ?? "I couldn't confidently map that request to a supported command.",
              },
            };
          }

          const { nextState, result } = executeCommand(parsed.command, current);

          return {
            ...nextState,
            paletteResult: result,
          };
        });
      })
      .catch((error: unknown) => {
        const detail =
          error instanceof Error
            ? error.message
            : "Kai could not reach the active local parser backend. Make sure the configured runtime is available.";

        setState((current) => ({
          ...current,
          paletteResult: {
            mode: "error",
            title: "Local parser unavailable",
            detail,
          },
        }));
      });
  };

  const handleQueryChange = (value: string) => {
    setState((current) => ({
      ...current,
      paletteQuery: value,
      paletteResult:
        value.trim().length === 0
          ? { mode: "idle", title: "Quick command", detail: "Create reminders, block time, or sync providers." }
          : current.paletteResult,
    }));
  };

  const handleSelectView = (activeView: ViewMode) => {
    setState((current) => ({ ...current, activeView }));
  };

  const handleQuickNoteChange = (value: string) => {
    setQuickNoteDraft(value);
  };

  const handleSelectNote = (noteId: string) => {
    setSelectedNoteId(noteId);
    setState((current) => ({ ...current, activeView: "notes" }));
  };

  const handleCreateNote = () => {
    const now = new Date().toISOString();
    const newNote: NoteItem = {
      id: crypto.randomUUID(),
      title: "Untitled Note",
      body: "",
      updatedAt: now,
    };

    setNotes((current) => [newNote, ...current]);
    setSelectedNoteId(newNote.id);
    setState((current) => ({ ...current, activeView: "notes" }));
  };

  const handleDeleteSelectedNote = () => {
    const activeNoteId = selectedNote?.id;

    if (!activeNoteId) {
      return;
    }

    setNotes((current) => current.filter((note) => note.id !== activeNoteId));
  };

  const handleUpdateSelectedNote = (body: string) => {
    const activeNoteId = selectedNote?.id;

    if (!activeNoteId) {
      return;
    }

    setNotes((current) =>
      current.map((note) =>
        note.id === activeNoteId
          ? {
              ...note,
              body,
              title: deriveNoteTitle(body),
              updatedAt: new Date().toISOString(),
            }
          : note,
      ),
    );
  };

  useEffect(() => {
    let isMounted = true;

    const setupDesktop = async () => {
      const unlisten = await bindSurfaceListener((nextSurface) => {
        if (!isMounted) {
          return;
        }

        setSurface(nextSurface);
        setBrowserVisible(true);
      });

      return unlisten;
    };

    let cleanup: (() => void) | undefined;

    void setupDesktop().then((unlisten) => {
      cleanup = unlisten;
    });

    if (!isTauriRuntime()) {
      const handleKeydown = (event: KeyboardEvent) => {
        if (!(event.metaKey || event.ctrlKey)) {
          return;
        }

        if (event.key === "/") {
          event.preventDefault();
          setSurface("palette");
          setBrowserVisible((current) => !current || surfaceRef.current !== "palette");
        }

        if (event.shiftKey && event.key === ":") {
          event.preventDefault();
          setSurface("dashboard");
          setBrowserVisible((current) => !current || surfaceRef.current !== "dashboard");
        }

        if (event.key === "Escape") {
          setBrowserVisible(false);
        }
      };

      window.addEventListener("keydown", handleKeydown);

      return () => {
        window.removeEventListener("keydown", handleKeydown);
        cleanup?.();
        isMounted = false;
      };
    }

    return () => {
      cleanup?.();
      isMounted = false;
    };
  }, []);

  useEffect(() => {
    surfaceRef.current = surface;
  }, [surface]);

  useEffect(() => {
    saveQuickNote(quickNoteDraft);
  }, [quickNoteDraft]);

  useEffect(() => {
    saveNotes(notes);
  }, [notes]);

  useEffect(() => {
    if (!selectedNoteId && sortedNotes[0]) {
      setSelectedNoteId(sortedNotes[0].id);
      return;
    }

    if (selectedNoteId && !sortedNotes.some((note) => note.id === selectedNoteId)) {
      setSelectedNoteId(sortedNotes[0]?.id ?? null);
    }
  }, [selectedNoteId, sortedNotes]);

  useEffect(() => {
    if (state.activeView === "notes" && sortedNotes.length === 0) {
      const now = new Date().toISOString();
      const newNote: NoteItem = {
        id: crypto.randomUUID(),
        title: "Untitled Note",
        body: "",
        updatedAt: now,
      };

      setNotes([newNote]);
      setSelectedNoteId(newNote.id);
    }
  }, [state.activeView, sortedNotes]);

  useEffect(() => {
    if (!isTauriRuntime()) {
      return;
    }

    void warmLocalParser().catch(() => undefined);
  }, []);

  useEffect(() => {
    if (surface !== "palette" || !isTauriRuntime()) {
      return;
    }

    window.requestAnimationFrame(() => {
      const measuredHeight =
        state.paletteResult.mode === "idle"
          ? PALETTE_COLLAPSED_HEIGHT
          : Math.min(
              PALETTE_MAX_HEIGHT,
              Math.max(PALETTE_MIN_EXPANDED_HEIGHT, Math.ceil((paletteRef.current?.scrollHeight ?? 0) + 2)),
            );

      void setPaletteHeight(measuredHeight);
    });
  }, [surface, state.paletteResult]);

  if (!browserVisible && !isTauriRuntime()) {
    return <div className="app-shell app-shell-hidden" />;
  }

  return (
    <div className={`app-shell surface-${surface}`}>
      {surface === "palette" && (
        <CommandPalette
          ref={paletteRef}
          query={state.paletteQuery}
          result={state.paletteResult}
          onQueryChange={handleQueryChange}
          onSubmit={handleSubmit}
        />
      )}

      {surface === "dashboard" && (
        <Dashboard
          activeView={state.activeView}
          tasks={state.tasks}
          events={state.events}
          assignments={state.assignments}
          accounts={state.accounts}
          syncQueue={state.syncQueue}
          quickNoteDraft={quickNoteDraft}
          onQuickNoteDraftChange={handleQuickNoteChange}
          notes={sortedNotes}
          selectedNoteId={selectedNote?.id ?? null}
          selectedNoteBody={selectedNote?.body ?? ""}
          onSelectNote={handleSelectNote}
          onCreateNote={handleCreateNote}
          onDeleteSelectedNote={handleDeleteSelectedNote}
          onUpdateSelectedNote={handleUpdateSelectedNote}
          sidebarCollapsed={sidebarCollapsed}
          onToggleSidebar={() => setSidebarCollapsed((current) => !current)}
          onSelectView={handleSelectView}
        />
      )}
    </div>
  );
};
