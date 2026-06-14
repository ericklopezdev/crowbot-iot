import { NavLink, Outlet, useParams } from "react-router-dom";
import { api, USE_MOCKS } from "../api";
import { useApi } from "../hooks/useApi";
import { useAuth } from "../context/AuthContext";
import { ThemeToggle } from "../components/ui/ThemeToggle";
import { Button, Spinner } from "../components/ui";
import "./layout.css";

export function DashboardLayout() {
  const { childId } = useParams();
  const { logout } = useAuth();
  const { data: children, loading } = useApi(() => api.listChildren(), []);

  return (
    <div className="layout">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-dot" />
          Crowbot
        </div>

        <div className="side-label">Niños</div>
        {loading && <Spinner />}
        {children?.map((c) => (
          <NavLink
            key={c.id}
            to={`/children/${c.id}`}
            className={({ isActive }) =>
              `nav-item ${isActive || childId === c.id ? "active" : ""}`
            }
          >
            <span className="avatar">{c.name.charAt(0).toUpperCase()}</span>
            {c.name}
          </NavLink>
        ))}

        <div className="sidebar-footer">
          <ThemeToggle />
          {!USE_MOCKS && (
            <Button variant="default" onClick={logout}>
              Salir
            </Button>
          )}
        </div>
      </aside>

      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}
