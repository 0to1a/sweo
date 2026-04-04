import { createRoute, Link } from "@tanstack/react-router";
import { rootRoute } from "./__root";
import { useSSE } from "../hooks/useSSE";
import { SessionCard } from "../components/SessionCard";
import { StatusBadge } from "../components/StatusBadge";
import type { Session, SessionStatus } from "../lib/types";

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

      <PRTable sessions={sessions} />
    </div>
  );
}

function PRTable({ sessions }: { sessions: Session[] }) {
  const prs = sessions.filter((s) => s.PRNumber > 0);

  if (prs.length === 0) return null;

  return (
    <div className="pr-section">
      <h2>Open Pull Requests</h2>
      <table className="pr-table">
        <thead>
          <tr>
            <th>PR</th>
            <th>Session</th>
            <th>Project</th>
            <th>Branch</th>
            <th>Issue</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {prs.map((s) => (
            <tr key={s.ID}>
              <td>
                <a href={s.PRURL} target="_blank" rel="noopener noreferrer">
                  #{s.PRNumber}
                </a>
              </td>
              <td>
                <Link to="/sessions/$sessionId" params={{ sessionId: s.ID }} style={{ color: "var(--text)" }}>
                  {s.ID}
                </Link>
              </td>
              <td>{s.ProjectID}</td>
              <td>{s.Branch}</td>
              <td>{s.IssueID ? `#${s.IssueID}` : "-"}</td>
              <td><StatusBadge status={s.Status} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
