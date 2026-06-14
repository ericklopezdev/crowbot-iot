import type { Api } from "./endpoints";
import { realApi } from "./endpoints";
import { mockApi } from "../mocks/handlers";

export const USE_MOCKS = import.meta.env.VITE_USE_MOCKS === "true";

export const api: Api = USE_MOCKS ? mockApi : realApi;

export type { Api };
