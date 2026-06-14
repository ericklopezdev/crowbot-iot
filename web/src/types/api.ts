// Tipos 1:1 con el JSON del backend Go (cwlb-server).
// pgtype.* serializa limpio: Date -> "YYYY-MM-DD", UUID -> string|null,
// Timestamptz -> ISO string|null. Verificado en el smoke test.

export type UUID = string;
export type ISODate = string; // "2018-05-01"
export type ISODateTime = string; // "2026-06-14T00:00:00-05:00"

// ---- auth ----
export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number; // segundos
}

export interface LoginRequest {
  email: string;
  password: string;
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
// language | logic-math | science | socio-emotional | creativity | social-world
export type CognitiveArea =
  | "language"
  | "logic-math"
  | "science"
  | "socio-emotional"
  | "creativity"
  | "social-world"
  | string;

export interface AreaBreakdown {
  cognitive_area: CognitiveArea;
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
  bucket: ISODateTime; // inicio de la hora
  interaction_count: number;
}

export interface Recommendation {
  id: UUID;
  child_id: UUID;
  topic_id: UUID | null;
  area: string;
  title: string;
  body: string;
  status: string;
  generated_at: ISODateTime;
}

export interface Overview {
  child: Child;
  total_interactions: number;
  areas: AreaBreakdown[];
  top_topics: TopicGraphNode[];
}

export interface ApiError {
  error: string;
}
