import { useEffect, useState } from "react";

interface State<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

// Hook genérico de fetch. `deps` re-dispara la llamada (p.ej. childId).
export function useApi<T>(fetcher: () => Promise<T>, deps: unknown[]): State<T> {
  const [state, setState] = useState<State<T>>({ data: null, loading: true, error: null });

  useEffect(() => {
    let alive = true;
    setState({ data: null, loading: true, error: null });
    fetcher()
      .then((data) => alive && setState({ data, loading: false, error: null }))
      .catch((err) => {
        const msg = err?.response?.data?.error ?? err?.message ?? "Error desconocido";
        if (alive) setState({ data: null, loading: false, error: msg });
      });
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return state;
}
