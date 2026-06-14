import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { useApi } from "./hooks/useApi";
import { api } from "./api";
import { useAuth } from "./context/AuthContext";
import { DashboardLayout } from "./pages/DashboardLayout";
import { ChildOverviewPage } from "./pages/ChildOverviewPage";
import { InteractionsPage } from "./pages/InteractionsPage";
import { LoginPage } from "./pages/LoginPage";
import { Spinner } from "./components/ui";

function RequireAuth({ children }: { children: React.ReactNode }) {
  const { isAuthed } = useAuth();
  return isAuthed ? <>{children}</> : <Navigate to="/login" replace />;
}

// Redirige "/" al primer niño disponible.
function HomeRedirect() {
  const { data, loading, error } = useApi(() => api.listChildren(), []);
  if (loading) return <Spinner />;
  if (error || !data || data.length === 0) return <Navigate to="/login" replace />;
  return <Navigate to={`/children/${data[0].id}`} replace />;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          element={
            <RequireAuth>
              <DashboardLayout />
            </RequireAuth>
          }
        >
          <Route path="/" element={<HomeRedirect />} />
          <Route path="/children/:childId" element={<ChildOverviewPage />} />
          <Route path="/children/:childId/interactions" element={<InteractionsPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
