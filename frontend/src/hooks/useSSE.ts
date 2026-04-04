import { useState, useEffect, useRef } from "react";
import type { Session, SSESnapshot } from "../lib/types";

export function useSSE() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [connected, setConnected] = useState(false);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const es = new EventSource("/api/events");
    esRef.current = es;

    es.onopen = () => setConnected(true);

    es.onmessage = (event) => {
      try {
        const data: SSESnapshot = JSON.parse(event.data);
        if (data.type === "snapshot" && data.sessions) {
          setSessions(data.sessions);
        }
      } catch {
        // ignore parse errors
      }
    };

    es.onerror = () => {
      setConnected(false);
    };

    return () => {
      es.close();
      esRef.current = null;
    };
  }, []);

  return { sessions, connected };
}
