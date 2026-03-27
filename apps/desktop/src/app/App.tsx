import { useEffect, useRef, useState } from "react";
import { CommandPalette } from "../features/command-center/CommandPalette";
import { Dashboard } from "../features/dashboard/Dashboard";
import { initialKaiState } from "../data/mockData";
import { executeCommand, parseCommand } from "../lib/commandEngine";
import { bindSurfaceListener, hideKaiWindow, isTauriRuntime, setPaletteHeight, warmLocalParser } from "../lib/desktop";
import type { DesktopSurface, KaiState, PaletteResult, ViewMode } from "../lib/types";

const loadingState: PaletteResult = {
  mode: "loading",
  title: "Understanding command",
  detail: "Executing locally",
};

const PALETTE_COLLAPSED_HEIGHT = 88;
const PALETTE_MIN_EXPANDED_HEIGHT = 170;
const PALETTE_MAX_HEIGHT = 520;

export const App = () => {
  const [state, setState] = useState<KaiState>(initialKaiState);
  const [surface, setSurface] = useState<DesktopSurface>("palette");
  const [browserVisible, setBrowserVisible] = useState(true);
  const surfaceRef = useRef<DesktopSurface>("palette");
  const paletteRef = useRef<HTMLDivElement>(null);

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

  const handleHide = () => {
    if (isTauriRuntime()) {
      void hideKaiWindow();
      return;
    }

    setBrowserVisible(false);
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
          onClose={handleHide}
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
          onSelectView={handleSelectView}
          onClose={handleHide}
        />
      )}
    </div>
  );
};
