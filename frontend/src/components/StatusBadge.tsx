import type { SessionStatus } from "../lib/types";

export function StatusBadge({ status }: { status: SessionStatus }) {
  return <span className={`badge badge-${status}`}>{status.replace("_", " ")}</span>;
}

export function StatusDot({ status }: { status: SessionStatus }) {
  const colorMap: Record<string, string> = {
    working: "dot-green",
    spawning: "dot-blue",
    waiting_input: "dot-blue",
    pr_open: "dot-blue",
    review_pending: "dot-blue",
    ci_failed: "dot-yellow",
    changes_requested: "dot-yellow",
    errored: "dot-red",
    stuck: "dot-red",
    merged: "dot-green",
    done: "dot-gray",
    cleanup: "dot-gray",
  };

  return <span className={`dot ${colorMap[status] ?? "dot-gray"}`} />;
}
