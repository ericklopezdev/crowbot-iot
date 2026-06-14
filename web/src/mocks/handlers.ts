import type { Api } from "../api/endpoints";
import type { TokenResponse } from "../types/api";
import {
  areasFor,
  children,
  interactionsFor,
  overviewFor,
  recsFor,
  topicsFor,
  usageFor,
} from "./fixtures";

const delay = <T>(value: T, ms = 150): Promise<T> =>
  new Promise((resolve) => setTimeout(() => resolve(value), ms));

const fakeToken: TokenResponse = {
  access_token: "mock.access.token",
  refresh_token: "mock.refresh.token",
  expires_in: 900,
};

export const mockApi: Api = {
  login: () => delay(fakeToken),
  listChildren: () => delay(children),
  getOverview: (id) => delay(overviewFor(id)),
  getUsage: (id) => delay(usageFor(id)),
  getTopics: (id) => delay(topicsFor(id)),
  getAreas: (id) => delay(areasFor(id)),
  getRecommendations: (id) => delay(recsFor(id)),
  getInteractions: (id, limit, offset) => delay(interactionsFor(id, limit, offset)),
};
