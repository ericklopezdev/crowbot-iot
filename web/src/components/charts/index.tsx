import type { AreaBreakdown, TopicGraphNode, UsageBucket } from "../../types/api";
import { Badge } from "../ui";
import "./charts.css";

const AREA_LABELS: Record<string, string> = {
  language: "Lenguaje",
  "logic-math": "Lógica-Mate",
  science: "Ciencia",
  "socio-emotional": "Socioemocional",
  creativity: "Creatividad",
  "social-world": "Mundo social",
};

export function areaLabel(area: string): string {
  return AREA_LABELS[area] ?? area;
}

export function AreaBars({ data }: { data: AreaBreakdown[] }) {
  const max = Math.max(1, ...data.map((d) => d.interaction_count));
  return (
    <div className="bars">
      {data.map((d) => (
        <div className="bar-row" key={d.cognitive_area}>
          <span className="bar-label" title={areaLabel(d.cognitive_area)}>
            {areaLabel(d.cognitive_area)}
          </span>
          <div className="bar-track">
            <div className="bar-fill" style={{ width: `${(d.interaction_count / max) * 100}%` }} />
          </div>
          <span className="bar-value">{d.interaction_count}</span>
        </div>
      ))}
    </div>
  );
}

export function TopicChips({ data }: { data: TopicGraphNode[] }) {
  return (
    <div className="chips">
      {data.map((t) => (
        <Badge key={t.id} label={t.label} count={t.interaction_count} />
      ))}
    </div>
  );
}

const DAYS = ["Dom", "Lun", "Mar", "Mié", "Jue", "Vie", "Sáb"];

export function UsageHeatmap({ data }: { data: UsageBucket[] }) {
  // Agrupar por (offset de día desde hoy, hora). 7 días recientes.
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const grid: number[][] = Array.from({ length: 7 }, () => Array(24).fill(0));
  let max = 1;
  for (const b of data) {
    const d = new Date(b.bucket);
    const dayStart = new Date(d);
    dayStart.setHours(0, 0, 0, 0);
    const offset = Math.round((today.getTime() - dayStart.getTime()) / 86400000);
    if (offset < 0 || offset > 6) continue;
    const row = 6 - offset; // fila 0 = hace 6 días, fila 6 = hoy
    grid[row][d.getHours()] += b.interaction_count;
    max = Math.max(max, grid[row][d.getHours()]);
  }

  const rowDate = (row: number) => {
    const dt = new Date(today);
    dt.setDate(today.getDate() - (6 - row));
    return DAYS[dt.getDay()];
  };

  const opacity = (n: number) => (n === 0 ? 0.06 : 0.2 + (n / max) * 0.8);

  return (
    <div>
      <div className="heatmap">
        {grid.map((row, ri) => (
          <div className="heat-row" key={ri}>
            <span className="heat-day">{rowDate(ri)}</span>
            {row.map((n, hi) => (
              <div
                className="heat-cell"
                key={hi}
                style={{ opacity: opacity(n) }}
                title={`${n} interacciones · ${hi}:00`}
              />
            ))}
          </div>
        ))}
        <div className="heat-hours">
          <span />
          {Array.from({ length: 24 }, (_, h) => (
            <span key={h}>{h % 6 === 0 ? h : ""}</span>
          ))}
        </div>
      </div>
      <div className="heat-legend">
        <span>menos</span>
        <div className="heat-cell" style={{ opacity: 0.2 }} />
        <div className="heat-cell" style={{ opacity: 0.5 }} />
        <div className="heat-cell" style={{ opacity: 0.85 }} />
        <span>más</span>
      </div>
    </div>
  );
}
