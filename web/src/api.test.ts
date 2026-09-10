import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { wikiAPI, getApiHost, getApiBase } from './api';

function mockFetchOnce(status: number, body: unknown, ok?: boolean) {
  const okVal = ok ?? (status >= 200 && status < 300);
  (globalThis.fetch as any).mockResolvedValueOnce({
    ok: okVal,
    status,
    json: async () => body,
  });
}

beforeEach(() => {
  localStorage.clear();
  globalThis.fetch = vi.fn();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('getApiHost / getApiBase', () => {
  it('returns the window origin by default', () => {
    expect(getApiHost()).toBe(window.location.origin);
  });

  it('points at localhost:8080 when served from the Vite dev server (port 5173)', () => {
    const original = window.location;
    Object.defineProperty(window, 'location', {
      value: { ...original, port: '5173', origin: 'http://localhost:5173' },
      writable: true,
    });
    expect(getApiHost()).toBe('http://localhost:8080');
    Object.defineProperty(window, 'location', { value: original, writable: true });
  });

  it('appends /api/v1 to the host', () => {
    expect(getApiBase()).toBe(`${getApiHost()}/api/v1`);
  });
});

describe('wikiAPI.login / logout / getMe', () => {
  it('stores the token on successful login', async () => {
    mockFetchOnce(200, { token: 'abc123', user: { id: 1, username: 'admin' } });
    const res = await wikiAPI.login('admin', 'admin');
    expect(res.token).toBe('abc123');
    expect(localStorage.getItem('ptc_auth_token')).toBe('abc123');
  });

  it('throws and does not store a token on failed login', async () => {
    mockFetchOnce(401, { error: 'Felaktigt användarnamn eller lösenord' });
    await expect(wikiAPI.login('admin', 'wrong')).rejects.toThrow('Felaktigt användarnamn eller lösenord');
    expect(localStorage.getItem('ptc_auth_token')).toBeNull();
  });

  it('falls back to a generic HTTP error message when the body has no error field', async () => {
    mockFetchOnce(500, {});
    await expect(wikiAPI.login('x', 'y')).rejects.toThrow('HTTP-fel 500');
  });

  it('logout removes the stored token', async () => {
    localStorage.setItem('ptc_auth_token', 'abc123');
    await wikiAPI.logout();
    expect(localStorage.getItem('ptc_auth_token')).toBeNull();
  });

  it('getMe returns null without calling fetch when there is no token', async () => {
    const result = await wikiAPI.getMe();
    expect(result).toBeNull();
    expect(fetch).not.toHaveBeenCalled();
  });

  it('getMe returns the user when a token is present and the call succeeds', async () => {
    localStorage.setItem('ptc_auth_token', 'abc123');
    mockFetchOnce(200, { user: { id: 1, username: 'admin' } });
    const result = await wikiAPI.getMe();
    expect(result?.username).toBe('admin');
  });

  it('getMe clears the token and returns null when the call fails', async () => {
    localStorage.setItem('ptc_auth_token', 'stale-token');
    mockFetchOnce(401, { error: 'ogiltig token' });
    const result = await wikiAPI.getMe();
    expect(result).toBeNull();
    expect(localStorage.getItem('ptc_auth_token')).toBeNull();
  });

  it('sends the Authorization header when a token is present', async () => {
    localStorage.setItem('ptc_auth_token', 'my-token');
    mockFetchOnce(200, { user: { id: 1 } });
    await wikiAPI.getMe();
    const [, options] = (fetch as any).mock.calls[0];
    expect(options.headers.Authorization).toBe('Bearer my-token');
  });
});

describe('wikiAPI.changePassword', () => {
  it('sends the current and new password as a PUT request', async () => {
    mockFetchOnce(200, { message: 'ok' });
    await wikiAPI.changePassword('old', 'newpassword123');
    const [url, options] = (fetch as any).mock.calls[0];
    expect(url).toContain('/auth/password');
    expect(options.method).toBe('PUT');
    expect(JSON.parse(options.body)).toEqual({ current_password: 'old', new_password: 'newpassword123' });
  });
});

describe('wikiAPI admin & user-key methods', () => {
  it('adminCreateUser accepts an object payload', async () => {
    mockFetchOnce(201, { data: { id: 2, username: 'ny' } });
    const user = await wikiAPI.adminCreateUser({ username: 'ny', email: 'ny@example.com', password: 'pw', role: 'user' });
    expect(user.username).toBe('ny');
  });

  it('adminCreateUser accepts legacy positional string arguments', async () => {
    mockFetchOnce(201, { data: { id: 3, username: 'legacy' } });
    const user = await wikiAPI.adminCreateUser('legacy', 'pw', 'admin');
    const [, options] = (fetch as any).mock.calls[0];
    expect(JSON.parse(options.body)).toEqual({ username: 'legacy', password: 'pw', role: 'admin' });
    expect(user.username).toBe('legacy');
  });

  it('adminListUsers returns the data array, defaulting to empty', async () => {
    mockFetchOnce(200, {});
    expect(await wikiAPI.adminListUsers()).toEqual([]);
  });

  it('listUserApiKeys and its alias userListApiKeys both work', async () => {
    mockFetchOnce(200, { data: [{ id: 1, name: 'k1' }] });
    expect(await wikiAPI.listUserApiKeys()).toHaveLength(1);
    mockFetchOnce(200, { data: [{ id: 1, name: 'k1' }] });
    expect(await wikiAPI.userListApiKeys()).toHaveLength(1);
  });

  it('createUserApiKey falls back from key to api_key field', async () => {
    mockFetchOnce(201, { api_key: 'ptc_key_fallback', data: { id: 1 } });
    const res = await wikiAPI.createUserApiKey('Name');
    expect(res.key).toBe('ptc_key_fallback');
    expect(res.api_key).toBe('ptc_key_fallback');
  });

  it('userCreateApiKey delegates to createUserApiKey', async () => {
    mockFetchOnce(201, { key: 'ptc_key_direct', data: { id: 1 } });
    const res = await wikiAPI.userCreateApiKey('Name', '30d');
    expect(res.key).toBe('ptc_key_direct');
  });

  it('revokeUserApiKey and its alias both issue a DELETE', async () => {
    mockFetchOnce(200, {});
    await wikiAPI.revokeUserApiKey(5);
    expect((fetch as any).mock.calls[0][1].method).toBe('DELETE');

    mockFetchOnce(200, {});
    await wikiAPI.userRevokeApiKey(5);
    expect((fetch as any).mock.calls[1][1].method).toBe('DELETE');
  });
});

describe('wikiAPI pages', () => {
  it('listPages builds a query string only for provided filters', async () => {
    mockFetchOnce(200, { data: [] });
    await wikiAPI.listPages();
    expect((fetch as any).mock.calls[0][0]).not.toContain('?');

    mockFetchOnce(200, { data: [] });
    await wikiAPI.listPages('hej', 'go');
    const url = (fetch as any).mock.calls[1][0];
    expect(url).toContain('search=hej');
    expect(url).toContain('tag=go');
  });

  it('getPage, createPage, updatePage, deletePage hit the right endpoints', async () => {
    mockFetchOnce(200, { data: { slug: 'x' } });
    await wikiAPI.getPage('x');
    expect((fetch as any).mock.calls[0][0]).toContain('/pages/x');

    mockFetchOnce(201, { data: { slug: 'y' } });
    await wikiAPI.createPage({ title: 'Y', content: 'z' });
    const [createUrl, createOpts] = (fetch as any).mock.calls[1];
    expect(createUrl).toContain('/pages');
    expect(createOpts.method).toBe('POST');

    mockFetchOnce(200, { data: { slug: 'x' } });
    await wikiAPI.updatePage('x', { content: 'new' });
    expect((fetch as any).mock.calls[2][1].method).toBe('PUT');

    mockFetchOnce(200, {});
    await wikiAPI.deletePage('x');
    expect((fetch as any).mock.calls[3][1].method).toBe('DELETE');
  });

  it('getRevisions, revertRevision, getBacklinks default to sensible empty results', async () => {
    mockFetchOnce(200, {});
    expect(await wikiAPI.getRevisions('x')).toEqual([]);

    mockFetchOnce(200, { data: { slug: 'x' } });
    await wikiAPI.revertRevision('x', 3);
    expect((fetch as any).mock.calls[1][0]).toContain('/revert/3');

    mockFetchOnce(200, {});
    expect(await wikiAPI.getBacklinks('x')).toEqual([]);
  });

  it('searchPages URL-encodes the query and reads the results field', async () => {
    mockFetchOnce(200, { results: [{ slug: 'found' }] });
    const results = await wikiAPI.searchPages('a b/c');
    expect((fetch as any).mock.calls[0][0]).toContain(encodeURIComponent('a b/c'));
    expect(results).toHaveLength(1);
  });

  it('listTags returns the data array', async () => {
    mockFetchOnce(200, { data: [{ id: 1, name: 'go', slug: 'go' }] });
    expect(await wikiAPI.listTags()).toHaveLength(1);
  });
});

describe('wikiAPI attachments', () => {
  it('uploadAttachment posts multipart form data and returns the response', async () => {
    mockFetchOnce(201, { data: { id: 1 }, markdown: '![x](y)' });
    const file = new File(['content'], 'test.png', { type: 'image/png' });
    const res = await wikiAPI.uploadAttachment('my-slug', file);
    expect(res.markdown).toBe('![x](y)');

    const [url, options] = (fetch as any).mock.calls[0];
    expect(url).toContain('/pages/my-slug/attachments');
    expect(options.body).toBeInstanceOf(FormData);
  });

  it('uploadAttachment throws with the server error message on failure', async () => {
    mockFetchOnce(400, { error: 'Sidan hittades inte' });
    const file = new File(['content'], 'test.png', { type: 'image/png' });
    await expect(wikiAPI.uploadAttachment('finns-inte', file)).rejects.toThrow('Sidan hittades inte');
  });

  it('getAttachments and deleteAttachment hit the right endpoints', async () => {
    mockFetchOnce(200, { data: [] });
    expect(await wikiAPI.getAttachments('x')).toEqual([]);

    mockFetchOnce(200, {});
    await wikiAPI.deleteAttachment(9);
    expect((fetch as any).mock.calls[1][0]).toContain('/attachments/9');
  });
});

describe('wikiAPI.getHealth', () => {
  it('returns the health payload', async () => {
    mockFetchOnce(200, { status: 'ok', service: 'wiki-api' });
    const res = await wikiAPI.getHealth();
    expect(res.status).toBe('ok');
  });
});
