import { http } from "./client";
import type {
  AreaBreakdown,
  Child,
  Interaction,
  Overview,
  Recommendation,
  TokenResponse,
  TopicGraphNode,
  UsageBucket,
} from "../types/api";

export interface Api {
  login(email: string, password: string): Promise<TokenResponse>;
  listChildren(): Promise<Child[]>;
  getOverview(childId: string): Promise<Overview>;
  getUsage(childId: string): Promise<UsageBucket[]>;
  getTopics(childId: string): Promise<TopicGraphNode[]>;
  getAreas(childId: string): Promise<AreaBreakdown[]>;
  getRecommendations(childId: string): Promise<Recommendation[]>;
  getInteractions(childId: string, limit: number, offset: number): Promise<Interaction[]>;
}

export const realApi: Api = {
  async login(email, password) {
    const { data } = await http.post<TokenResponse>("/api/auth/login", { email, password });
    return data;
  },
  async listChildren() {
    const { data } = await http.get<Child[]>("/api/children");
    return data;
  },
  async getOverview(childId) {
    const { data } = await http.get<Overview>(`/api/children/${childId}/overview`);
    return data;
  },
  async getUsage(childId) {
    const { data } = await http.get<UsageBucket[]>(`/api/children/${childId}/usage`);
    return data;
  },
  async getTopics(childId) {
    const { data } = await http.get<TopicGraphNode[]>(`/api/children/${childId}/topics`);
    return data;
  },
  async getAreas(childId) {
    const { data } = await http.get<AreaBreakdown[]>(`/api/children/${childId}/areas`);
    return data;
  },
  async getRecommendations(childId) {
    const { data } = await http.get<Recommendation[]>(`/api/children/${childId}/recommendations`);
    return data;
  },
  async getInteractions(childId, limit, offset) {
    const { data } = await http.get<Interaction[]>(
      `/api/children/${childId}/interactions?limit=${limit}&offset=${offset}`,
    );
    return data;
  },
};
