import type { Page, CreatePageInput, UpdatePageInput, Revision, Tag, Attachment, User, UserApiKey } from './types';

export const getApiHost = (): string => {
  if (typeof window === 'undefined') return 'http://localhost:8080';
  // If Vite dev server on port 5173, point to backend on port 8080
  if (window.location.port === '5173') {
    return 'http://localhost:8080';
  }
  // Production / single binary server: use current browser origin
  return window.location.origin;
};

export const getApiBase = (): string => {
  if (import.meta.env.VITE_API_URL) {
    return import.meta.env.VITE_API_URL;
  }
  return `${getApiHost()}/api/v1`;
};

function getAuthHeader(): Record<string, string> {
  const token = localStorage.getItem('ptc_auth_token');
  if (token) {
    return { Authorization: `Bearer ${token}` };
  }
  return {};
}

async function fetchJSON<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${getApiBase()}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeader(),
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

  // Auth API
  async login(username: string, password: string): Promise<{ token: string; user: User }> {
    const res = await fetchJSON<{ token: string; user: User }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
    if (res.token) {
      localStorage.setItem('ptc_auth_token', res.token);
    }
    return res;
  },

  async logout(): Promise<void> {
    localStorage.removeItem('ptc_auth_token');
  },

  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    await fetchJSON('/auth/password', {
      method: 'PUT',
      body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
    });
  },

  async getMe(): Promise<User | null> {
    const token = localStorage.getItem('ptc_auth_token');
    if (!token) return null;
    try {
      const res = await fetchJSON<{ user: User }>('/auth/me');
      return res.user || null;
    } catch {
      localStorage.removeItem('ptc_auth_token');
      return null;
    }
  },

  async adminCreateUser(data: { username: string; email?: string; password: string; role: string } | string, passwordArg?: string, roleArg?: string): Promise<User> {
    let body: any;
    if (typeof data === 'object') {
      body = data;
    } else {
      body = { username: data, password: passwordArg, role: roleArg };
    }
    const res = await fetchJSON<{ data: User }>('/admin/users', {
      method: 'POST',
      body: JSON.stringify(body),
    });
    return res.data;
  },

  async adminListUsers(): Promise<User[]> {
    const res = await fetchJSON<{ data: User[] }>('/admin/users');
    return res.data || [];
  },

  async listUserApiKeys(): Promise<UserApiKey[]> {
    const res = await fetchJSON<{ data: UserApiKey[] }>('/user/keys');
    return res.data || [];
  },

  async userListApiKeys(): Promise<UserApiKey[]> {
    return this.listUserApiKeys();
  },

  async createUserApiKey(name: string, expiresAt?: string): Promise<{ key: string; api_key: string; data: UserApiKey }> {
    const res = await fetchJSON<{ key?: string; api_key?: string; data: UserApiKey }>('/user/keys', {
      method: 'POST',
      body: JSON.stringify({ name, expires: expiresAt, expires_at: expiresAt }),
    });
    const secretKey = res.key || res.api_key || '';
    return { key: secretKey, api_key: secretKey, data: res.data };
  },

  async userCreateApiKey(name: string, expiresAt?: string): Promise<{ key: string; api_key?: string; data: UserApiKey }> {
    return this.createUserApiKey(name, expiresAt);
  },

  async revokeUserApiKey(id: number): Promise<void> {
    await fetchJSON(`/user/keys/${id}`, {
      method: 'DELETE',
    });
  },

  async userRevokeApiKey(id: number): Promise<void> {
    return this.revokeUserApiKey(id);
  },

  // Wiki Pages API
  async listPages(search?: string, tag?: string): Promise<Page[]> {
    const params = new URLSearchParams();
    if (search) params.append('search', search);
    if (tag) params.append('tag', tag);
    const queryString = params.toString() ? `?${params.toString()}` : '';

    const res = await fetchJSON<{ data: Page[] }>(`/pages${queryString}`);
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

  async revertRevision(slug: string, revisionId: number): Promise<Page> {
    const res = await fetchJSON<{ data: Page }>(`/pages/${slug}/revert/${revisionId}`, {
      method: 'POST',
    });
    return res.data;
  },

  async getBacklinks(slug: string): Promise<Page[]> {
    const res = await fetchJSON<{ data: Page[] }>(`/pages/${slug}/backlinks`);
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

  async uploadAttachment(slug: string, file: File): Promise<{ data: Attachment; markdown: string }> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch(`${getApiBase()}/pages/${slug}/attachments`, {
      method: 'POST',
      headers: {
        ...getAuthHeader(),
      },
      body: formData,
    });

    if (!response.ok) {
      const errData = await response.json().catch(() => ({}));
      throw new Error(errData.error || 'Misslyckades att ladda upp filen');
    }

    return response.json();
  },

  async getAttachments(slug: string): Promise<Attachment[]> {
    const res = await fetchJSON<{ data: Attachment[] }>(`/pages/${slug}/attachments`);
    return res.data || [];
  },

  async deleteAttachment(id: number): Promise<void> {
    await fetchJSON(`/attachments/${id}`, {
      method: 'DELETE',
    });
  },
};
