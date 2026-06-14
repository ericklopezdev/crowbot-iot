import type {
  AreaBreakdown,
  Child,
  Interaction,
  Overview,
  Recommendation,
  TopicGraphNode,
  UsageBucket,
} from "../types/api";

// Data mock tipada y realista para capturas sin backend.

export const children: Child[] = [
  {
    id: "c1111111-1111-1111-1111-111111111111",
    account_id: "a1111111-1111-1111-1111-111111111111",
    name: "Mateo",
    birthdate: "2018-05-01",
    avatar: "",
    created_at: "2026-06-01T10:00:00-05:00",
  },
  {
    id: "c2222222-2222-2222-2222-222222222222",
    account_id: "a1111111-1111-1111-1111-111111111111",
    name: "Lucía",
    birthdate: "2016-09-14",
    avatar: "",
    created_at: "2026-06-02T10:00:00-05:00",
  },
];

const areasByChild: Record<string, AreaBreakdown[]> = {
  "c1111111-1111-1111-1111-111111111111": [
    { cognitive_area: "science", interaction_count: 42 },
    { cognitive_area: "language", interaction_count: 31 },
    { cognitive_area: "creativity", interaction_count: 24 },
    { cognitive_area: "logic-math", interaction_count: 18 },
    { cognitive_area: "socio-emotional", interaction_count: 11 },
    { cognitive_area: "social-world", interaction_count: 7 },
  ],
  "c2222222-2222-2222-2222-222222222222": [
    { cognitive_area: "language", interaction_count: 38 },
    { cognitive_area: "socio-emotional", interaction_count: 22 },
    { cognitive_area: "creativity", interaction_count: 15 },
    { cognitive_area: "science", interaction_count: 9 },
  ],
};

const topicsByChild: Record<string, TopicGraphNode[]> = {
  "c1111111-1111-1111-1111-111111111111": [
    topic("dinosaurs", "Dinosaurios", "science", 19, "2026-06-13T18:20:00-05:00"),
    topic("space", "Espacio", "science", 14, "2026-06-12T17:00:00-05:00"),
    topic("animals", "Animales", "science", 12, "2026-06-13T09:10:00-05:00"),
    topic("stories", "Cuentos", "language", 11, "2026-06-11T20:00:00-05:00"),
    topic("drawing", "Dibujar", "creativity", 9, "2026-06-10T16:30:00-05:00"),
    topic("numbers", "Números", "logic-math", 7, "2026-06-09T15:00:00-05:00"),
    topic("feelings", "Emociones", "socio-emotional", 5, "2026-06-08T19:00:00-05:00"),
  ],
  "c2222222-2222-2222-2222-222222222222": [
    topic("stories", "Cuentos", "language", 21, "2026-06-13T20:10:00-05:00"),
    topic("friends", "Amigos", "socio-emotional", 13, "2026-06-12T18:00:00-05:00"),
    topic("music", "Música", "creativity", 8, "2026-06-11T17:00:00-05:00"),
  ],
};

const recsByChild: Record<string, Recommendation[]> = {
  "c1111111-1111-1111-1111-111111111111": [
    rec(
      1,
      "science",
      "Visita un museo de historia natural",
      "Mateo pregunta mucho por dinosaurios y fósiles. Una visita guiada puede convertir su curiosidad en una experiencia concreta.",
    ),
    rec(
      2,
      "logic-math",
      "Juegos de conteo en casa",
      "Su interés por los números es incipiente. Contar objetos cotidianos (escalones, frutas) refuerza la noción de cantidad jugando.",
    ),
  ],
  "c2222222-2222-2222-2222-222222222222": [
    rec(
      3,
      "language",
      "Lectura compartida cada noche",
      "Lucía disfruta los cuentos. Leer juntos y pedirle que prediga el final estimula comprensión y vocabulario.",
    ),
  ],
};

export function overviewFor(childId: string): Overview {
  const child = children.find((c) => c.id === childId) ?? children[0];
  const areas = areasByChild[child.id] ?? [];
  const topics = topicsByChild[child.id] ?? [];
  return {
    child,
    total_interactions: areas.reduce((s, a) => s + a.interaction_count, 0),
    areas,
    top_topics: topics.slice(0, 5),
  };
}

export const areasFor = (id: string) => areasByChild[id] ?? [];
export const topicsFor = (id: string) => topicsByChild[id] ?? [];
export const recsFor = (id: string) => recsByChild[id] ?? [];

// Heatmap: 7 días x 24 horas con un patrón realista (picos tarde/noche).
export function usageFor(childId: string): UsageBucket[] {
  const seed = childId.charCodeAt(1);
  const out: UsageBucket[] = [];
  const now = new Date("2026-06-14T00:00:00-05:00");
  for (let d = 6; d >= 0; d--) {
    for (let h = 0; h < 24; h++) {
      const peak = h >= 16 && h <= 20 ? 3 : h >= 9 && h <= 12 ? 1.5 : 0.3;
      const n = Math.max(0, Math.round(peak * pseudo(seed + d * 24 + h)));
      if (n === 0) continue;
      const bucket = new Date(now);
      bucket.setDate(now.getDate() - d);
      bucket.setHours(h, 0, 0, 0);
      out.push({ bucket: bucket.toISOString(), interaction_count: n });
    }
  }
  return out;
}

export function interactionsFor(childId: string, limit: number, offset: number): Interaction[] {
  const all = sampleInteractions(childId);
  return all.slice(offset, offset + limit);
}

// ---- helpers ----
function topic(
  slug: string,
  label: string,
  area: string,
  count: number,
  lastSeen: string,
): TopicGraphNode {
  return {
    id: `70000000-0000-0000-0000-${slug.padEnd(12, "0").slice(0, 12)}`,
    slug,
    label,
    area,
    parent_id: null,
    interaction_count: count,
    last_seen_at: lastSeen,
  };
}

function rec(n: number, area: string, title: string, body: string): Recommendation {
  return {
    id: `60000000-0000-0000-0000-${String(n).padStart(12, "0")}`,
    child_id: "c1111111-1111-1111-1111-111111111111",
    topic_id: null,
    area,
    title,
    body,
    status: "new",
    generated_at: "2026-06-13T21:00:00-05:00",
  };
}

function pseudo(n: number): number {
  const x = Math.sin(n * 12.9898) * 43758.5453;
  return x - Math.floor(x);
}

const prompts = [
  ["¿por qué los dinosaurios se extinguieron?", "Hace mucho tiempo un gran meteorito cambió el clima de la Tierra..."],
  ["cuéntame un cuento de un dragón", "Había una vez un dragón pequeño que tenía miedo de volar..."],
  ["¿cuántas patas tiene una araña?", "Las arañas tienen ocho patas, ¡dos más que los insectos!"],
  ["¿de qué está hecha la luna?", "La luna está hecha sobre todo de roca y polvo gris..."],
  ["quiero dibujar un cohete", "¡Buena idea! Empieza por un triángulo arriba y un tubo largo abajo..."],
  ["estoy triste hoy", "Lamento que te sientas así. ¿Quieres contarme qué pasó?"],
];

function sampleInteractions(childId: string): Interaction[] {
  const out: Interaction[] = [];
  for (let i = 0; i < 24; i++) {
    const [t, r] = prompts[i % prompts.length];
    const created = new Date("2026-06-14T00:00:00-05:00");
    created.setHours(created.getHours() - i * 5);
    out.push({
      id: `1d57096f-d7a6-49d4-8ec9-${String(i).padStart(12, "0")}`,
      device_id: "d1111111-1111-1111-1111-111111111111",
      child_id: childId,
      session_id: `aaaa0000-0000-0000-0000-${String(i).padStart(12, "0")}`,
      transcript: t,
      response_text: r,
      stt_latency_ms: 120 + (i % 5) * 10,
      llm_latency_ms: 540 + (i % 7) * 25,
      tts_latency_ms: 210 + (i % 4) * 15,
      total_latency_ms: 900 + (i % 9) * 30,
      llm_model: "gemini-1.5-flash",
      llm_tokens_in: 48 + i,
      llm_tokens_out: 96 + i * 2,
      created_at: created.toISOString(),
    });
  }
  return out;
}
