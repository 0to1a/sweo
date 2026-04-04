import { createRoute, Link } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { rootRoute } from "./__root";
import { StatusBadge } from "../components/StatusBadge";
import { Terminal } from "../components/Terminal";
import type { Session } from "../lib/types";

export const sessionRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/sessions/$sessionId",
  component: SessionDetailPage,
});

function SessionDetailPage() {
  const { sessionId } = sessionRoute.useParams();
  const [session, setSession] = useState<Session | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch(`/api/sessions/${sessionId}`)
      .then((res) => {
        if (!res.ok) throw new Error("Session not found");
        return res.json();
      })
      .then(setSession)
      .catch((err) => setError(err.message));

    // Refresh session data every 10s
    const interval = setInterval(() => {
      fetch(`/api/sessions/${sessionId}`)
        .then((res) => res.json())
        .then(setSession)
        .catch(() => {});
    }, 10000);

    return () => clearInterval(interval);
  }, [sessionId]);

  if (error) {
    return (
      <div className="empty-state">
        <h3>Session not found</h3>
        <p>{error}</p>
        <Link to="/">Back to sessions</Link>
      </div>
    );
  }

  if (!session) {
    return <div className="empty-state"><p>Loading...</p></div>;
  }

  return (
    <div className="session-detail">
      <div className="detail-header">
        <Link to="/" style={{ color: "var(--text-dim)", fontSize: "0.875rem" }}>
          ← Back
        </Link>
        <h2>{session.ID}</h2>
        <StatusBadge status={session.Status} />
      </div>

      <div className="detail-meta">
        <MetaItem label="Project" value={session.ProjectID} />
        <MetaItem label="Agent" value={session.Agent} />
        <MetaItem label="Status" value={session.Status} />
        {session.IssueID && (
          <MetaItem label="Issue" value={`#${session.IssueID} ${session.IssueTitle}`} />
        )}
        {session.Branch && <MetaItem label="Branch" value={session.Branch} />}
        {session.PRNumber > 0 && (
          <MetaItem label="PR" value={`#${session.PRNumber}`} link={session.PRURL} />
        )}
        <MetaItem label="Created" value={new Date(session.CreatedAt).toLocaleString()} />
      </div>

      <div style={{ display: "flex", gap: "8px" }}>
        {session.Status === "stuck" && (
          <button
            className="btn btn-primary"
            onClick={() => {
              fetch(`/api/sessions/${session.ID}/resume`, { method: "POST" })
                .then(() => setSession({ ...session, Status: "working" }));
            }}
          >
            Resume
          </button>
        )}
        <button
          className="btn btn-danger"
          onClick={() => {
            if (confirm(`Kill session ${session.ID}?`)) {
              fetch(`/api/sessions/${session.ID}/kill`, { method: "POST" })
                .then(() => setSession({ ...session, Status: "done" }));
            }
          }}
        >
          Kill Session
        </button>
      </div>

      <h3 style={{ fontSize: "1rem", fontWeight: 600 }}>Terminal</h3>
      <Terminal sessionId={session.ID} />
    </div>
  );
}

function MetaItem({ label, value, link }: { label: string; value: string; link?: string }) {
  return (
    <div className="meta-item">
      <div className="label">{label}</div>
      <div className="value">
        {link ? (
          <a href={link} target="_blank" rel="noopener noreferrer">{value}</a>
        ) : (
          value
        )}
      </div>
    </div>
  );
}
