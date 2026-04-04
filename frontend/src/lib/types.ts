export interface Session {
  ID: string;
  ProjectID: string;
  Status: SessionStatus;
  Branch: string;
  IssueID: string;
  IssueTitle: string;
  PRURL: string;
  PRNumber: number;
  WorkspacePath: string;
  TmuxName: string;
  Agent: string;
  CreatedAt: string;
  Metadata: Record<string, string>;
}

export type SessionStatus =
  | "spawning"
  | "working"
  | "waiting_input"
  | "pr_open"
  | "ci_failed"
  | "changes_requested"
  | "review_pending"
  | "merged"
  | "cleanup"
  | "done"
  | "errored"
  | "stuck";

export interface Project {
  name: string;
  repo: string;
  agent: string;
  defaultBranch: string;
}

export interface SSESnapshot {
  type: "snapshot";
  timestamp: string;
  sessions: Session[];
}
