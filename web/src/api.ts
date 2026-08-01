import type { Page, CreatePageInput, UpdatePageInput, Revision, Tag } from './types';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

async function fetchJSON<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': 'wiki-secret-api-key',
      ...options?.headers,
    },
    ...options,
  });

  if (!response.ok) {
    const errData = await response.json().catch(() => ({}));
    throw new Error(errData.error || `HTTP-fel ${response.status}`);
  }

  return response.json();
}

export const wikiAPI = {
  async getHealth(): Promise<{ status: string; service: string }> {
    return fetchJSON<{ status: string; service: string }>('/health');
  },

  async listPages(search?: string, tag?: string): Promise<Page[]> {
    const params = new URLSearchParams();
    if (search) params.append('search', search);
    if (tag) params.append('tag', tag);
    const queryStr = params.toString() ? `?${params.toString()}` : '';
    const res = await fetchJSON<{ data: Page[] }>(`/pages${queryStr}`);
    return res.data || [];
  },

  async getPage(slug: string): Promise<Page> {
    const res = await fetchJSON<{ data: Page }>(`/pages/${slug}`);
    return res.data;
  },

  async createPage(input: CreatePageInput): Promise<Page> {
    const res = await fetchJSON<{ data: Page }>('/pages', {
      method: 'POST',
      body: JSON.stringify(input),
    });
    return res.data;
  },

  async updatePage(slug: string, input: UpdatePageInput): Promise<Page> {
    const res = await fetchJSON<{ data: Page }>(`/pages/${slug}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    });
    return res.data;
  },

  async deletePage(slug: string): Promise<void> {
    await fetchJSON(`/pages/${slug}`, {
      method: 'DELETE',
    });
  },

  async getRevisions(slug: string): Promise<Revision[]> {
    const res = await fetchJSON<{ data: Revision[] }>(`/pages/${slug}/revisions`);
    return res.data || [];
  },

  async searchPages(query: string): Promise<Page[]> {
    const res = await fetchJSON<{ results: Page[] }>(`/search?q=${encodeURIComponent(query)}`);
    return res.results || [];
  },

  async listTags(): Promise<Tag[]> {
    const res = await fetchJSON<{ data: Tag[] }>('/tags');
    return res.data || [];
  },
};
