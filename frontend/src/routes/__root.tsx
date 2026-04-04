import { createRootRoute, Link, Outlet } from "@tanstack/react-router";
import { useSSE } from "../hooks/useSSE";

export const rootRoute = createRootRoute({
  component: RootLayout,
});

function RootLayout() {
  const { connected } = useSSE();

  return (
    <div className="app">
      <header className="header">
        <h1>sweo</h1>
        <nav>
          <Link to="/" className="nav-link" activeProps={{ className: "active" }}>
            Sessions
          </Link>
          <span style={{ color: connected ? "var(--green)" : "var(--red)", fontSize: "0.75rem" }}>
            {connected ? "● Connected" : "● Disconnected"}
          </span>
        </nav>
      </header>
      <Outlet />
    </div>
  );
}
