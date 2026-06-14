import { useNavigate } from "react-router-dom";

export function ChildTabs({
  childId,
  active,
}: {
  childId: string;
  active: "overview" | "interactions";
}) {
  const nav = useNavigate();
  return (
    <div className="subtle-tabs">
      <button
        className={`tab ${active === "overview" ? "active" : ""}`}
        onClick={() => nav(`/children/${childId}`)}
      >
        Resumen
      </button>
      <button
        className={`tab ${active === "interactions" ? "active" : ""}`}
        onClick={() => nav(`/children/${childId}/interactions`)}
      >
        Interacciones
      </button>
    </div>
  );
}
