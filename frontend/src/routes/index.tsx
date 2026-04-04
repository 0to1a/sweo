import { createRoute } from "@tanstack/react-router";
import { rootRoute } from "./__root";
import { useSSE } from "../hooks/useSSE";
import { SessionCard } from "../components/SessionCard";
import type { SessionStatus } from "../lib/types";

export const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: SessionsPage,
});

const statusOrder: SessionStatus[] = [
  "spawning",
  "working",
  "waiting_input",
  "pr_open",
  "ci_failed",
  "changes_requested",
  "review_pending",
  "merged",
  "cleanup",
  "done",
  "errored",
  "stuck",
];

function SessionsPage() {
  const { sessions } = useSSE();

  const sorted = [...sessions].sort(
    (a, b) => statusOrder.indexOf(a.Status) - statusOrder.indexOf(b.Status),
  );

  const active = sessions.filter(
    (s) => !["done", "cleanup", "merged", "errored"].includes(s.Status),
  );
  const terminal = sessions.filter((s) =>
    ["done", "cleanup", "merged", "errored"].includes(s.Status),
  );

  return (
    <div>
      <div className="stats-bar">
        <span>
          Total: <span className="stat-value">{sessions.length}</span>
        </span>
        <span>
          Active: <span className="stat-value">{active.length}</span>
        </span>
        <span>
          Completed: <span className="stat-value">{terminal.length}</span>
        </span>
      </div>

      {sorted.length === 0 ? (
        <div className="empty-state">
          <h3>No sessions yet</h3>
          <p>
            Tag a GitHub issue with <code>agent:todo</code> to get started.
          </p>
        </div>
      ) : (
        <div className="session-grid">
          {sorted.map((session) => (
            <SessionCard key={session.ID} session={session} />
          ))}
        </div>
      )}
    </div>
  );
}
