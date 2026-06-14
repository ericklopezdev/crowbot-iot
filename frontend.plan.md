# Frontend Plan — Crowbot Parent Dashboard

> Objetivo: dashboard para padres, **simple**, poca arquitectura. Lo importante es
> (1) los **types** que reflejan el backend 1:1, (2) la **funcionalidad** contra la
> API, (3) una **UI excelente** blanco/negro estilo Notion con dark/light.
> Conexión al backend con **axios**. Modo **mock** (JSON inyectable) para capturas
> sin levantar el backend.

---

## 1. Stack (minimalista a propósito)

| Pieza            | Elección                  | Por qué                                            |
|------------------|---------------------------|----------------------------------------------------|
| Build            | **Vite + React + TS**     | cero config, HMR rápido                            |
| Routing          | **react-router-dom**      | 4-5 rutas, nada más                               |
| HTTP             | **axios**                 | pedido explícito; interceptor para el JWT          |
| Data fetching    | hook propio `useApi`      | sin react-query; un hook de ~30 líneas alcanza    |
| Estado global    | **React Context**         | solo auth + theme; nada de Redux/Zustand           |
| Estilos          | **CSS variables + CSS modules** (o plain CSS) | tematizar B/N con `--vars`, sin Tailwind |
| Charts           | **a definir** (ver §7)    | barras/heatmap simples; quizá sin librería         |

> Regla: si una pieza no aparece arriba, no se agrega sin discutir. "No mucha arquitectura".

---

## 2. Estructura de carpetas (`web/`)

```
web/
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
├── .env.example            # VITE_API_URL, VITE_USE_MOCKS
└── src/
    ├── main.tsx
    ├── App.tsx             # router + providers
    ├── types/
    │   └── api.ts          # ⭐ tipos 1:1 con el backend (ver §4)
    ├── api/
    │   ├── client.ts       # instancia axios + interceptor JWT
    │   ├── endpoints.ts    # funciones tipadas: login(), getOverview(), ...
    │   └── index.ts        # switch real ↔ mock según VITE_USE_MOCKS
    ├── mocks/
    │   ├── fixtures.ts      # ⭐ data mock tipada (ver §5)
    │   └── handlers.ts      # mock que devuelve fixtures con la misma firma que endpoints
    ├── hooks/
    │   ├── useApi.ts        # { data, loading, error } genérico
    │   └── useAuth.ts       # login/logout/token desde AuthContext
    ├── context/
    │   ├── AuthContext.tsx  # token en localStorage + estado
    │   └── ThemeContext.tsx # light/dark + localStorage + <html data-theme>
    ├── styles/
    │   ├── theme.css        # ⭐ :root y [data-theme=dark] (paleta Notion)
    │   └── base.css         # reset + tipografía + utilidades
    ├── components/
    │   ├── ui/              # Button, Card, Badge, Table, Spinner, ThemeToggle, EmptyState
    │   └── charts/          # AreaBars, UsageHeatmap, TopicChips
    └── pages/
        ├── LoginPage.tsx
        ├── DashboardLayout.tsx   # sidebar (hijos + theme toggle) + <Outlet/>
        ├── ChildOverviewPage.tsx # overview + areas + topics + usage + recs
        └── InteractionsPage.tsx  # tabla de interacciones (paginada)
```

---

## 3. Superficie del backend (lo que consume el front)

Base URL: `http://localhost:8080`. Auth: `Authorization: Bearer <access_token>`.

| Método | Ruta                                          | Auth | Devuelve                         |
|--------|-----------------------------------------------|------|----------------------------------|
| POST   | `/api/auth/signup`                            | no   | `TokenResponse`                  |
| POST   | `/api/auth/login`                             | no   | `TokenResponse`                  |
| POST   | `/api/auth/refresh`                           | no   | `TokenResponse`                  |
| GET    | `/api/children`                               | sí   | `Child[]`                        |
| GET    | `/api/children/{id}`                           | sí   | `Child`                          |
| POST   | `/api/children`                               | sí   | `Child` (201)                    |
| GET    | `/api/children/{id}/overview`                  | sí   | `Overview`                       |
| GET    | `/api/children/{id}/usage?from&to`             | sí   | `UsageBucket[]`                  |
| GET    | `/api/children/{id}/topics`                    | sí   | `TopicGraphNode[]`               |
| GET    | `/api/children/{id}/areas`                      | sí   | `AreaBreakdown[]`                |
| GET    | `/api/children/{id}/recommendations`           | sí   | `Recommendation[]`               |
| GET    | `/api/children/{id}/interactions?limit&offset` | sí   | `Interaction[]`                  |
| GET    | `/api/devices`                                 | sí   | `Device[]`                       |

> Errores: JSON `{ "error": "mensaje" }`. 401 → redirigir a login. 404 → empty state.
> Listas siempre vienen como `[]` (nunca `null`) gracias a `orEmpty` en el backend.

---

## 4. ⭐ Types (`src/types/api.ts`)

> Reflejan exactamente el JSON del backend. Confirmado en smoke test:
> `pgtype.Date`→`"YYYY-MM-DD"`, `pgtype.UUID`→`string | null`,
> `pgtype.Timestamptz`→ISO string `| null`.

```ts
export type UUID = string;
export type ISODate = string;       // "2018-05-01"
export type ISODateTime = string;   // "2026-06-14T00:00:00-05:00"

// ---- auth ----
export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number; // segundos
}

// ---- core ----
export interface Child {
  id: UUID;
  account_id: UUID;
  name: string;
  birthdate: ISODate | null;
  avatar: string;
  created_at: ISODateTime;
}

export interface Device {
  id: UUID;
  name: string;
  serial_number: string | null;
  firmware_version: string;
  last_seen_at: ISODateTime | null;
  claimed_at: ISODateTime | null;
}

export interface Interaction {
  id: UUID;
  device_id: UUID;
  child_id: UUID | null;
  session_id: UUID | null;
  transcript: string;
  response_text: string;
  stt_latency_ms: number;
  llm_latency_ms: number;
  tts_latency_ms: number;
  total_latency_ms: number;
  llm_model: string;
  llm_tokens_in: number;
  llm_tokens_out: number;
  created_at: ISODateTime;
}

// ---- dashboard ----
export interface AreaBreakdown {
  cognitive_area: string;       // language|logic-math|science|socio-emotional|creativity|social-world
  interaction_count: number;
}

export interface TopicGraphNode {
  id: UUID;
  slug: string;
  label: string;
  area: string;
  parent_id: UUID | null;
  interaction_count: number;
  last_seen_at: ISODateTime;
}

export interface UsageBucket {
  bucket: ISODateTime;          // inicio de la hora
  interaction_count: number;
}

export interface Recommendation {
  id: UUID;
  child_id: UUID;
  topic_id: UUID | null;
  area: string;
  title: string;
  body: string;
  status: string;               // p.ej. "new"
  generated_at: ISODateTime;
}

export interface Overview {
  child: Child;
  total_interactions: number;
  areas: AreaBreakdown[];
  top_topics: TopicGraphNode[]; // top 5
}
```

---

## 5. ⭐ Modo mock (capturas sin backend)

- Flag `VITE_USE_MOCKS=true` en `.env` → `src/api/index.ts` exporta los handlers mock
  en lugar de los reales. Misma firma tipada, así las páginas no cambian.
- `src/mocks/fixtures.ts`: data realista y tipada con los `interface` de §4
  (1 padre, 2 hijos, ~20 interacciones, áreas/topics/usage poblados, 2-3 recs).
  Datos pensados para que las gráficas se vean bien en screenshots.
- Cada handler mock simula `await delay(150)` para que los spinners aparezcan.

```ts
// src/api/index.ts (idea)
import * as real from "./endpoints";
import * as mock from "../mocks/handlers";
export const api = import.meta.env.VITE_USE_MOCKS === "true" ? mock : real;
```

---

## 6. ⭐ Theming Notion (B/N + dark/light) — `src/styles/theme.css`

Paleta neutra, sin acentos de color (Notion). Acento = el propio texto/borde.

```css
:root {
  --bg:        #ffffff;
  --bg-subtle: #f7f7f5;   /* gris Notion */
  --surface:   #ffffff;
  --border:    #e9e9e7;
  --text:      #191919;
  --text-muted:#787774;
  --accent:    #191919;   /* B/N: el acento es negro */
  --radius: 8px;
  --font: -apple-system, "Segoe UI", Inter, system-ui, sans-serif;
}
[data-theme="dark"] {
  --bg:        #191919;
  --bg-subtle: #202020;
  --surface:   #252525;
  --border:    #373737;
  --text:      #ededed;
  --text-muted:#9b9b9b;
  --accent:    #ededed;
}
```

- `ThemeContext`: estado `'light' | 'dark'`, persistido en localStorage, aplica
  `document.documentElement.dataset.theme`. Default = `prefers-color-scheme`.
- `<ThemeToggle/>` en el sidebar (icono sol/luna).
- Principios visuales Notion: mucho espacio en blanco, bordes 1px sutiles, sombras
  mínimas o nulas, tipografía del sistema, esquinas redondeadas suaves, hover gris claro.

---

## 7. Dashboard — qué se renderiza

**`ChildOverviewPage`** (la pantalla estrella):
- Header: nombre del hijo + edad (de `birthdate`) + `total_interactions`.
- **Cards de áreas cognitivas** (`AreaBreakdown[]`): barras horizontales B/N
  (% del total). Sin librería: `<div>` con `width` = proporción.
- **Topics** (`TopicGraphNode[]`): chips con `label` + contador; ordenados por frecuencia.
- **Usage heatmap** (`UsageBucket[]`): grilla hora×día, celdas con opacidad según
  `interaction_count`. CSS puro.
- **Recommendations** (`Recommendation[]`): lista de cards `title` + `body`.
- Estados: loading (spinner), error (mensaje), **empty state** por sección.

**`InteractionsPage`**: tabla (`transcript`, `response_text`, `total_latency_ms`,
`created_at`) con paginación `limit`/`offset`.

> Charts: empezar **sin librería** (CSS). Si el heatmap/áreas piden más, evaluar
> `recharts`. Decisión diferida hasta tener data real en pantalla.

---

## 8. Auth flow (mínimo)

1. `LoginPage` → `POST /api/auth/login` → guarda `access_token` (+ refresh) en
   localStorage vía `AuthContext`.
2. `api/client.ts`: interceptor de request agrega `Authorization: Bearer`.
   Interceptor de response: en 401 → limpia token → redirige a `/login`.
3. Rutas protegidas: `<RequireAuth>` envuelve `DashboardLayout`.
4. Refresh: opcional en v1 (si el access expira, re-login). `refresh_token` queda
   guardado para implementarlo después sin cambiar tipos.

---

## 9. Orden de construcción (incremental, verificable)

> Estado al **2026-06-14**. Stack final: Vite 8 + React 19 + react-router-dom 7 + axios.

- [x] 1. **Scaffold** en `web/` (`react-ts`), `axios react-router-dom` instalados,
      `.env` con `VITE_USE_MOCKS=true`. Limpieza del boilerplate (App.css, assets).
- [x] 2. **Types** (`types/api.ts`) 1:1 con el backend + **fixtures** (`mocks/`):
      2 niños, áreas/topics/usage/recs/interacciones poblados.
- [x] 3. **Theme** (`theme.css` paleta Notion + `ThemeContext` + `ThemeToggle`) +
      **`ui/`** base (Card, Button, Badge, Spinner, EmptyState, `Async`).
- [x] 4. **DashboardLayout** + sidebar con lista de hijos (mock) + redirect `/` → 1er niño.
- [x] 5. **ChildOverviewPage** completa: stat-cards, `AreaBars`, `TopicChips`,
      `UsageHeatmap`, recomendaciones (todo charts CSS, sin librería).
- [x] 6. **InteractionsPage** con tabla + paginación `limit`/`offset`.
- [x] 7. **api real** (`endpoints.ts` + `client.ts` con interceptor JWT + 401→login) +
      **LoginPage** + `api/index.ts` que conmuta mock↔real según el flag.
- [x] 8a. `npm run build` (tsc + vite) **limpio**; dev server corriendo en mock.
- [x] 8b. **Bug fix**: TDZ en `fixtures.ts` (`recSeq` usado antes de init) causaba
      página en blanco; resuelto pasando el id como parámetro. UI ya renderiza.
- [ ] 8c. Flip `VITE_USE_MOCKS=false` → probar contra el backend real (DB ya seedeada,
      `parent@test.com` / `test1234`). **Pendiente.**
- [ ] 9. Pulido: responsive fino + QA visual dark/light en navegador. **Pendiente.**

---

## 10. Fuera de alcance v1

- Claim/assign de devices desde la UI (existe en backend; va en v2).
- Crear/editar hijos desde la UI (endpoint existe; v2).
- Refresh-token automático (guardado pero no usado en v1).
- i18n (UI en español hardcodeada de momento).
- Tests del front (manual + capturas por ahora).
```
