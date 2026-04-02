import { type FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { GlassPanel } from "../components/GlassPanel";
import { initialKaiState } from "../data/mockData";
import { CommandPalette } from "../features/command-center/CommandPalette";
import { Dashboard } from "../features/dashboard/Dashboard";
import {
  fetchGoogleAuthStatus,
  fetchKaiAuthStatus,
  logoutKai,
  signInWithEmail,
  signUpWithEmail,
  startGoogleAuth,
  type EmailAuthPayload,
  type KaiAuthStatus,
} from "../lib/backend";
import { executeCommand, parseCommand } from "../lib/commandEngine";
import { bindSurfaceListener, centerKaiWindow, isTauriRuntime, setPaletteHeight, warmLocalParser } from "../lib/desktop";
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

interface AuthGateProps {
  pending: boolean;
  error: string | null;
  onGoogleSignIn: () => void;
  onEmailSignIn: (payload: EmailAuthPayload) => Promise<void>;
  onEmailSignUp: (payload: EmailAuthPayload) => Promise<void>;
}

const AuthGate = ({ pending, error, onGoogleSignIn, onEmailSignIn, onEmailSignUp }: AuthGateProps) => {
  const [mode, setMode] = useState<"sign-in" | "sign-up">("sign-in");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const payload: EmailAuthPayload = {
      email,
      password,
      name,
    };

    void (mode === "sign-in" ? onEmailSignIn(payload) : onEmailSignUp(payload));
  };

  return (
    <GlassPanel className="auth-shell">
      <div className="auth-layout">
        <section className="auth-hero">
          <div className="auth-hero-copy">
            <div className="auth-brand">
              <div className="brand-mark" />
              <span>Kai</span>
            </div>
            <div className="auth-copy">
              <h1>Everything aligned before your day begins.</h1>
              <p>Kai brings your reminders, notes, and connected calendars into one calm desktop workspace.</p>
            </div>
          </div>
        </section>

        <section className="auth-panel">
          <div className="auth-panel-header">
            <div className="auth-copy auth-copy-compact">
              <h2>{mode === "sign-in" ? "Sign in" : "Create your account"}</h2>
              <p>{mode === "sign-in" ? "Pick up where you left off." : "Start with Google or a Kai account."}</p>
            </div>

            <div className="toolbar-pill segmented-pill auth-mode-switcher">
              <button type="button" className={mode === "sign-in" ? "is-active" : ""} onClick={() => setMode("sign-in")}>
                Sign in
              </button>
              <button type="button" className={mode === "sign-up" ? "is-active" : ""} onClick={() => setMode("sign-up")}>
                Create account
              </button>
            </div>
          </div>

          <button type="button" className="auth-google-button" onClick={onGoogleSignIn} disabled={pending}>
            Continue with Google
          </button>

          <div className="auth-divider">
            <span>or use email</span>
          </div>

          <form className="auth-form" onSubmit={handleSubmit}>
            {mode === "sign-up" ? (
              <label className="auth-field">
                <span>Name</span>
                <input value={name} onChange={(event) => setName(event.target.value)} placeholder="Your name" />
              </label>
            ) : null}

            <label className="auth-field">
              <span>Email</span>
              <input
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder="you@example.com"
                autoComplete="email"
              />
            </label>

            <label className="auth-field">
              <span>Password</span>
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder={mode === "sign-in" ? "Your password" : "Create a password"}
                autoComplete={mode === "sign-in" ? "current-password" : "new-password"}
              />
            </label>

            {error ? <p className="auth-error">{error}</p> : null}

            <button type="submit" className="auth-submit-button" disabled={pending}>
              {pending ? "Working..." : mode === "sign-in" ? "Sign in" : "Create account"}
            </button>
          </form>
        </section>
      </div>
    </GlassPanel>
  );
};

export const App = () => {
  const [state, setState] = useState<KaiState>(initialKaiState);
  const [surface, setSurface] = useState<DesktopSurface>("palette");
  const [browserVisible, setBrowserVisible] = useState(true);
  const [quickNoteDraft, setQuickNoteDraft] = useState(() => loadQuickNote());
  const [notes, setNotes] = useState<NoteItem[]>(() => buildInitialNotes());
  const [selectedNoteId, setSelectedNoteId] = useState<string | null>(null);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [authState, setAuthState] = useState<KaiAuthStatus>({
    provider: "kai",
    status: "disconnected",
  });
  const [authPending, setAuthPending] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const surfaceRef = useRef<DesktopSurface>("palette");
  const authStateRef = useRef<KaiAuthStatus>({
    provider: "kai",
    status: "disconnected",
  });
  const paletteRef = useRef<HTMLDivElement>(null);

  const sortedNotes = useMemo(
    () => [...notes].sort((left, right) => right.updatedAt.localeCompare(left.updatedAt)),
    [notes],
  );
  const selectedNote = sortedNotes.find((note) => note.id === selectedNoteId) ?? sortedNotes[0] ?? null;
  const isAuthenticated = authState.status === "connected";

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

  const handleEmailAuth = async (payload: EmailAuthPayload, mode: "sign-in" | "sign-up") => {
    setAuthPending(true);
    setAuthError(null);

    try {
      const nextAuthState = mode === "sign-in" ? await signInWithEmail(payload) : await signUpWithEmail(payload);
      setAuthState(nextAuthState);
      setState((current) => ({ ...current, activeView: "today" }));
    } catch (error) {
      setAuthError(error instanceof Error ? error.message : "Kai sign-in failed.");
    } finally {
      setAuthPending(false);
    }
  };

  const handleLogout = async () => {
    try {
      await logoutKai();
      setAuthState({
        provider: "kai",
        status: "disconnected",
      });
      setAuthError(null);
    } catch (error) {
      setAuthError(error instanceof Error ? error.message : "Kai sign-out failed.");
    }
  };

  useEffect(() => {
    let isMounted = true;

    const setupDesktop = async () => {
      const unlisten = await bindSurfaceListener((nextSurface) => {
        if (!isMounted) {
          return;
        }

        setSurface(authStateRef.current.status === "connected" ? nextSurface : "dashboard");
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
          setSurface(authStateRef.current.status === "connected" ? "palette" : "dashboard");
          setBrowserVisible((current) => !current || surfaceRef.current !== "dashboard");
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
    authStateRef.current = authState;

    if (authState.status !== "connected" && surface === "palette") {
      setSurface("dashboard");
    }
  }, [authState, surface]);

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
    if (surface !== "palette" || !isTauriRuntime() || !isAuthenticated) {
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
  }, [isAuthenticated, surface, state.paletteResult]);

  useEffect(() => {
    let cancelled = false;

    const syncKaiAuthStatus = async () => {
      try {
        const status = await fetchKaiAuthStatus();

        if (!cancelled) {
          setAuthState(status);
          setAuthError(null);
        }
      } catch (error) {
        if (!cancelled) {
          setAuthState({
            provider: "kai",
            status: "disconnected",
          });
          setAuthError(error instanceof Error ? error.message : "Kai auth is unavailable.");
        }
      }
    };

    const syncGoogleAuthStatus = async () => {
      try {
        const status = await fetchGoogleAuthStatus();

        if (cancelled) {
          return;
        }

        setState((current) => ({
          ...current,
          accounts: current.accounts.map((account) =>
            account.provider === "google"
              ? {
                  ...account,
                  email: status.email,
                  status: status.status,
                }
              : account,
          ),
        }));
      } catch {
        if (cancelled) {
          return;
        }

        setState((current) => ({
          ...current,
          accounts: current.accounts.map((account) =>
            account.provider === "google"
              ? {
                  ...account,
                  status: "disconnected",
                }
              : account,
          ),
        }));
      }
    };

    void syncKaiAuthStatus();
    void syncGoogleAuthStatus();

    const interval = window.setInterval(() => {
      void syncKaiAuthStatus();
      void syncGoogleAuthStatus();
    }, 5000);

    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, []);

  useEffect(() => {
    if (!isTauriRuntime()) {
      return;
    }

    const animationFrame = window.requestAnimationFrame(() => {
      void centerKaiWindow();
    });

    return () => {
      window.cancelAnimationFrame(animationFrame);
    };
  }, [surface, sidebarCollapsed]);

  if (!browserVisible && !isTauriRuntime()) {
    return <div className="app-shell app-shell-hidden" />;
  }

  return (
    <div className={`app-shell surface-${surface}`}>
      {surface === "palette" && isAuthenticated ? (
        <CommandPalette
          ref={paletteRef}
          query={state.paletteQuery}
          result={state.paletteResult}
          onQueryChange={handleQueryChange}
          onSubmit={handleSubmit}
        />
      ) : null}

      {surface === "dashboard" || !isAuthenticated ? (
        isAuthenticated ? (
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
            authName={authState.name}
            authEmail={authState.email}
            onToggleSidebar={() => setSidebarCollapsed((current) => !current)}
            onStartGoogleAuth={startGoogleAuth}
            onLogout={handleLogout}
            onSelectView={handleSelectView}
          />
        ) : (
          <AuthGate
            pending={authPending}
            error={authError}
            onGoogleSignIn={startGoogleAuth}
            onEmailSignIn={(payload) => handleEmailAuth(payload, "sign-in")}
            onEmailSignUp={(payload) => handleEmailAuth(payload, "sign-up")}
          />
        )
      ) : null}
    </div>
  );
};
