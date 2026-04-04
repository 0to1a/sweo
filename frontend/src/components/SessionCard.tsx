import { Link } from "@tanstack/react-router";
import type { Session } from "../lib/types";
import { StatusBadge } from "./StatusBadge";

function formatAge(createdAt: string): string {
  const ms = Date.now() - new Date(createdAt).getTime();
  const mins = Math.floor(ms / 60000);
  if (mins < 60) return `${mins}m`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h`;
  return `${Math.floor(hours / 24)}d`;
}

export function SessionCard({ session }: { session: Session }) {
  return (
    <Link
      to="/sessions/$sessionId"
      params={{ sessionId: session.ID }}
      className="session-card"
      style={{ textDecoration: "none", color: "inherit" }}
    >
      <div className="session-card-header">
        <h3>{session.ID}</h3>
        <StatusBadge status={session.Status} />
      </div>
      <div className="session-card-body">
        {session.IssueTitle && (
          <div className="row">
            <span className="label">Issue</span>
            <span className="value">#{session.IssueID} {session.IssueTitle.slice(0, 40)}</span>
          </div>
        )}
        <div className="row">
          <span className="label">Project</span>
          <span className="value">{session.ProjectID}</span>
        </div>
        <div className="row">
          <span className="label">Agent</span>
          <span className="value">{session.Agent}</span>
        </div>
        {session.PRNumber > 0 && (
          <div className="row">
            <span className="label">PR</span>
            <span className="value">#{session.PRNumber}</span>
          </div>
        )}
        {session.Branch && (
          <div className="row">
            <span className="label">Branch</span>
            <span className="value">{session.Branch}</span>
          </div>
        )}
        <div className="row">
          <span className="label">Age</span>
          <span className="value">{formatAge(session.CreatedAt)}</span>
        </div>
      </div>
    </Link>
  );
}
