import { useState } from "react";
import { useParams } from "react-router-dom";
import { api } from "../api";
import { useApi } from "../hooks/useApi";
import { Async, Button, Card } from "../components/ui";
import { ChildTabs } from "./ChildTabs";
import "./layout.css";

const PAGE = 25;

function fmtTime(iso: string): string {
  return new Date(iso).toLocaleString("es", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function InteractionsPage() {
  const { childId = "" } = useParams();
  const [page, setPage] = useState(0);

  const { data, loading, error } = useApi(
    () => api.getInteractions(childId, PAGE, page * PAGE),
    [childId, page],
  );

  return (
    <>
      <div className="content-head">
        <h1>Interacciones</h1>
      </div>
      <ChildTabs childId={childId} active="interactions" />

      <Card>
        <Async loading={loading} error={error} data={data} isEmpty={(d) => d.length === 0}>
          {(rows) => (
            <>
              <table className="table">
                <thead>
                  <tr>
                    <th>Niño dijo</th>
                    <th>Crowbot respondió</th>
                    <th>Latencia</th>
                    <th>Cuándo</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((it) => (
                    <tr key={it.id}>
                      <td className="cell-strong">{it.transcript}</td>
                      <td className="cell-muted">{it.response_text}</td>
                      <td className="mono cell-muted">{it.total_latency_ms} ms</td>
                      <td className="cell-muted">{fmtTime(it.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>

              <div className="pager">
                <Button disabled={page === 0} onClick={() => setPage((p) => Math.max(0, p - 1))}>
                  Anterior
                </Button>
                <span className="muted">Página {page + 1}</span>
                <Button disabled={rows.length < PAGE} onClick={() => setPage((p) => p + 1)}>
                  Siguiente
                </Button>
              </div>
            </>
          )}
        </Async>
      </Card>
    </>
  );
}
