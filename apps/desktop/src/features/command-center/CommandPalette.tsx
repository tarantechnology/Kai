import { Check, LoaderCircle, Search, X } from "lucide-react";
import { forwardRef } from "react";
import { GlassPanel } from "../../components/GlassPanel";
import type { EventItem, PaletteResult, TaskItem } from "../../lib/types";

interface CommandPaletteProps {
  query: string;
  result: PaletteResult;
  onQueryChange: (value: string) => void;
  onSubmit: () => void;
}

const TaskPreview = ({ tasks }: { tasks: TaskItem[] }) => (
  <div className="palette-list">
    {tasks.map((task) => (
      <div key={task.id} className="palette-list-row">
        <span>{task.title}</span>
        <span>{task.dueAt ?? task.dueLabel ?? ""}</span>
      </div>
    ))}
  </div>
);

const EventPreview = ({ events }: { events: EventItem[] }) => (
  <div className="palette-list">
    {events.slice(0, 4).map((event) => (
      <div key={event.id} className="palette-list-row">
        <span>{event.title}</span>
        <span>{event.startLabel}</span>
      </div>
    ))}
  </div>
);

export const CommandPalette = forwardRef<HTMLDivElement, CommandPaletteProps>(
  ({ query, result, onQueryChange, onSubmit }, ref) => (
    <GlassPanel ref={ref} className={`palette ${result.mode === "idle" ? "is-collapsed" : "is-expanded"}`}>
      <form
        className="palette-input-row"
        onSubmit={(event) => {
          event.preventDefault();
          onSubmit();
        }}
      >
        <Search size={22} className="icon-muted" />
        <input
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          className="palette-input"
          placeholder="What else do I have to do today?"
        />
        {query.trim().length > 0 ? (
          <button
            type="button"
            className="ghost-icon-button ghost-icon-button-small"
            onClick={() => onQueryChange("")}
            aria-label="Clear search"
          >
            <X size={14} />
          </button>
        ) : (
          <div className="palette-input-spacer" aria-hidden="true" />
        )}
      </form>

      <div className={`palette-result ${result.mode === "idle" ? "is-hidden" : ""}`}>
        {result.mode === "loading" && (
          <div className="palette-status">
            <LoaderCircle size={18} className="spin accent-icon" />
            <span>Understanding command...</span>
          </div>
        )}

        {result.mode === "success" && (
          <div className="palette-success">
            <div className="success-mark">
              <Check size={26} />
            </div>
            <div>
              <h3>{result.title}</h3>
              <p>{result.detail}</p>
            </div>
          </div>
        )}

        {result.mode === "task_list" && (
          <>
            <div className="palette-meta">
              <span>{result.title}</span>
              <span>{result.detail}</span>
            </div>
            {result.tasks && <TaskPreview tasks={result.tasks} />}
          </>
        )}

        {result.mode === "schedule" && (
          <>
            <div className="palette-meta">
              <span>{result.title}</span>
              <span>{result.detail}</span>
            </div>
            {result.events && <EventPreview events={result.events} />}
          </>
        )}

        {result.mode === "sync" && (
          <div className="palette-status">
            <LoaderCircle size={18} className="spin accent-icon" />
            <div>
              <h3>{result.title}</h3>
              <p>{result.detail}</p>
            </div>
          </div>
        )}

        {result.mode === "error" && (
          <div className="palette-status is-error">
            <span>{result.title}</span>
            <p>{result.detail}</p>
          </div>
        )}
      </div>
    </GlassPanel>
  ),
);

CommandPalette.displayName = "CommandPalette";
