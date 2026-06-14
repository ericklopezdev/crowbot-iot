import { useParams } from "react-router-dom";
import { api } from "../api";
import { useApi } from "../hooks/useApi";
import { Async, Card } from "../components/ui";
import { AreaBars, TopicChips, UsageHeatmap, areaLabel } from "../components/charts";
import { ChildTabs } from "./ChildTabs";
import "./layout.css";

function ageFrom(birthdate: string | null): string | null {
  if (!birthdate) return null;
  const b = new Date(birthdate);
  const now = new Date();
  let age = now.getFullYear() - b.getFullYear();
  const m = now.getMonth() - b.getMonth();
  if (m < 0 || (m === 0 && now.getDate() < b.getDate())) age--;
  return `${age} años`;
}

export function ChildOverviewPage() {
  const { childId = "" } = useParams();

  const overview = useApi(() => api.getOverview(childId), [childId]);
  const usage = useApi(() => api.getUsage(childId), [childId]);
  const topics = useApi(() => api.getTopics(childId), [childId]);
  const recs = useApi(() => api.getRecommendations(childId), [childId]);

  return (
    <Async loading={overview.loading} error={overview.error} data={overview.data}>
      {(ov) => (
        <>
          <div className="content-head">
            <div>
              <h1>{ov.child.name}</h1>
              <span className="muted">
                {[ageFrom(ov.child.birthdate), `${ov.total_interactions} interacciones`]
                  .filter(Boolean)
                  .join(" · ")}
              </span>
            </div>
          </div>

          <ChildTabs childId={childId} active="overview" />

          <div className="stat-grid">
            <Card title="Interacciones totales">
              <div className="stat-value">{ov.total_interactions}</div>
            </Card>
            <Card title="Áreas activas">
              <div className="stat-value">{ov.areas.length}</div>
            </Card>
            <Card title="Temas explorados">
              <div className="stat-value">{topics.data?.length ?? "—"}</div>
            </Card>
            <Card title="Área principal">
              <div className="stat-value" style={{ fontSize: 20 }}>
                {ov.areas[0] ? areaLabel(ov.areas[0].cognitive_area) : "—"}
              </div>
            </Card>
          </div>

          <div className="dash-grid">
            <Card title="Áreas cognitivas">
              <Async
                loading={false}
                error={null}
                data={ov.areas}
                isEmpty={(d) => d.length === 0}
              >
                {(d) => <AreaBars data={d} />}
              </Async>
            </Card>

            <Card title="Temas de interés">
              <Async
                loading={topics.loading}
                error={topics.error}
                data={topics.data}
                isEmpty={(d) => d.length === 0}
              >
                {(d) => <TopicChips data={d} />}
              </Async>
            </Card>

            <Card title="Actividad por hora" style={{ gridColumn: "1 / -1" }}>
              <Async
                loading={usage.loading}
                error={usage.error}
                data={usage.data}
                isEmpty={(d) => d.length === 0}
              >
                {(d) => <UsageHeatmap data={d} />}
              </Async>
            </Card>

            <Card title="Recomendaciones" style={{ gridColumn: "1 / -1" }}>
              <Async
                loading={recs.loading}
                error={recs.error}
                data={recs.data}
                isEmpty={(d) => d.length === 0}
              >
                {(d) =>
                  d.map((r) => (
                    <div className="rec" key={r.id}>
                      <div className="rec-area">{areaLabel(r.area)}</div>
                      <div className="rec-title">{r.title}</div>
                      <div className="rec-body">{r.body}</div>
                    </div>
                  ))
                }
              </Async>
            </Card>
          </div>
        </>
      )}
    </Async>
  );
}
