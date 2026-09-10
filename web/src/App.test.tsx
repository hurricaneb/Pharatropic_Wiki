import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from './App';
import type { Page, Tag, User } from './types';

vi.mock('./api', () => ({
  wikiAPI: {
    getHealth: vi.fn(),
    getMe: vi.fn(),
    listPages: vi.fn(),
    listTags: vi.fn(),
    getPage: vi.fn(),
    createPage: vi.fn(),
    updatePage: vi.fn(),
    deletePage: vi.fn(),
    getRevisions: vi.fn(),
    revertRevision: vi.fn(),
    logout: vi.fn(),
    login: vi.fn(),
    getBacklinks: vi.fn(),
    deleteAttachment: vi.fn(),
  },
  getApiHost: () => 'http://localhost:8080',
}));

import { wikiAPI } from './api';

function makePage(overrides: Partial<Page> & { id: number; slug: string; title: string }): Page {
  return {
    summary: '',
    content: `Content of ${overrides.title}`,
    is_public: true,
    views: 0,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

const pageA = makePage({ id: 1, slug: 'page-a', title: 'Page A' });
const pageB = makePage({ id: 2, slug: 'page-b', title: 'Page B' });
const tags: Tag[] = [{ id: 1, name: 'Go', slug: 'go' }];

function setupDefaultMocks() {
  vi.mocked(wikiAPI.getHealth).mockResolvedValue({ status: 'ok', service: 'wiki-api' });
  vi.mocked(wikiAPI.getMe).mockResolvedValue(null);
  vi.mocked(wikiAPI.listPages).mockResolvedValue([pageA, pageB]);
  vi.mocked(wikiAPI.listTags).mockResolvedValue(tags);
  vi.mocked(wikiAPI.getPage).mockImplementation(async (slug: string) => {
    const found = [pageA, pageB].find((p) => p.slug === slug);
    if (!found) throw new Error('Sidan hittades inte');
    return found;
  });
  vi.mocked(wikiAPI.getBacklinks).mockResolvedValue([]);
}

beforeEach(() => {
  vi.resetAllMocks();
  setupDefaultMocks();
});

describe('App', () => {
  it('marks the backend unhealthy when the health check fails, without crashing', async () => {
    vi.mocked(wikiAPI.getHealth).mockRejectedValue(new Error('network down'));
    const { container } = render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());
    const dot = container.querySelector('span[style*="border-radius: 50%"]');
    expect(dot).toHaveStyle({ backgroundColor: 'var(--danger)' });
  });

  it('stays logged out without crashing when getMe fails', async () => {
    vi.mocked(wikiAPI.getMe).mockRejectedValue(new Error('unauthorized'));
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());
    expect(screen.getByText('Logga in')).toBeInTheDocument();
  });

  it('sets the document title according to the current view', async () => {
    const loggedInUser: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(loggedInUser);
    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());
    expect(document.title).toBe('Page A | PTC Wiki');

    await user.click(screen.getByText('Redigera'));
    expect(document.title).toBe('Redigerar: Page A | PTC Wiki');

    await user.click(screen.getByText('Avbryt'));
    await user.click(screen.getByText('Ny Sida'));
    expect(document.title).toBe('Skapa ny sida | PTC Wiki');
  });

  it('edits the active page and returns to view mode', async () => {
    const loggedInUser: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(loggedInUser);
    const updated = makePage({ id: 1, slug: 'page-a', title: 'Page A Updated', content: 'Updated content' });
    vi.mocked(wikiAPI.updatePage).mockResolvedValue(updated);

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('Redigera'));
    expect(screen.getByText(/Redigera: Page A/)).toBeInTheDocument();

    vi.mocked(wikiAPI.getPage).mockResolvedValue(updated);
    vi.mocked(wikiAPI.listPages).mockResolvedValue([updated, pageB]);
    await user.click(screen.getByText('Spara Wiki-sida'));

    await waitFor(() => expect(wikiAPI.updatePage).toHaveBeenCalledWith('page-a', expect.objectContaining({ content: 'Content of Page A' })));
    await waitFor(() => expect(screen.getByText('Updated content')).toBeInTheDocument());
  });

  it('shows an alert when fetching revision history fails', async () => {
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    vi.mocked(wikiAPI.getRevisions).mockRejectedValue(new Error('Kunde inte hämta ändringshistorik.'));
    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('Historik'));
    await waitFor(() => expect(window.alert).toHaveBeenCalledWith('Kunde inte hämta ändringshistorik.'));
  });

  it('reverts a revision end-to-end from the history view', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    vi.mocked(wikiAPI.getRevisions).mockResolvedValue([
      { id: 1, page_id: 1, title: 'Page A', content: 'v1', comment: 'first', author: 'a', created_at: '2026-01-01T00:00:00Z' },
    ]);
    const reverted = makePage({ id: 1, slug: 'page-a', title: 'Page A', content: 'reverted content' });
    vi.mocked(wikiAPI.revertRevision).mockResolvedValue(reverted);

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('Historik'));
    await waitFor(() => expect(screen.getByText(/Ändringshistorik/)).toBeInTheDocument());

    vi.mocked(wikiAPI.getPage).mockResolvedValue(reverted);
    await user.click(screen.getByText('Återställ till denna version'));

    await waitFor(() => expect(wikiAPI.revertRevision).toHaveBeenCalledWith('page-a', 1));
    await waitFor(() => expect(screen.getByText('reverted content')).toBeInTheDocument());
  });

  it('navigates home to the first page when the logo is clicked', async () => {
    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('Page B'));
    await waitFor(() => expect(screen.getByText('Content of Page B')).toBeInTheDocument());

    await user.click(screen.getByText('Pharatropic Wiki'));
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());
  });

  it('shows an error message when fetching the selected page fails', async () => {
    vi.mocked(wikiAPI.getPage).mockRejectedValue(new Error('Kunde inte hämta sidan.'));
    render(<App />);
    await waitFor(() => expect(screen.getByText('Kunde inte hämta sidan.')).toBeInTheDocument());
  });

  it('shows an alert when deleting the active page fails', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    const loggedInUser: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(loggedInUser);
    vi.mocked(wikiAPI.deletePage).mockRejectedValue(new Error('Misslyckades att ta bort sidan.'));

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByTitle('Radera sida'));
    await waitFor(() => expect(window.alert).toHaveBeenCalledWith('Misslyckades att ta bort sidan.'));
  });

  it('opens and closes the admin users modal for an admin user', async () => {
    const admin: User = { id: 1, username: 'admin', email: 'a@example.com', role: 'admin', created_at: '' };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(admin);

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('Användare'));
    expect(screen.getByText('Användarhantering (Admin)')).toBeInTheDocument();
  });

  it('opens the API keys modal for a logged-in user', async () => {
    const loggedInUser: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(loggedInUser);

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('API-nycklar'));
    expect(screen.getByText('Mina API-nycklar')).toBeInTheDocument();
  });

  it('shows the empty state and can start creating a page from it', async () => {
    vi.mocked(wikiAPI.listPages).mockResolvedValue([]);
    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Ingen wiki-sida vald')).toBeInTheDocument());

    await user.click(screen.getByText('Skapa Ny Sida'));
    expect(screen.getByText('Skapa Ny Wiki-sida')).toBeInTheDocument();
  });

  it('filters pages by tag when a tag is selected in the sidebar', async () => {
    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('#Go'));
    await waitFor(() => expect(wikiAPI.listPages).toHaveBeenCalledWith('', 'go'));
  });


  it('loads pages and auto-selects the first one', async () => {
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());
    expect(wikiAPI.listPages).toHaveBeenCalled();
    expect(wikiAPI.getPage).toHaveBeenCalledWith('page-a');
  });

  it('shows the empty state when there are no pages', async () => {
    vi.mocked(wikiAPI.listPages).mockResolvedValue([]);
    render(<App />);
    await waitFor(() => expect(screen.getByText('Ingen wiki-sida vald')).toBeInTheDocument());
  });

  it('shows an error banner when loading the page list fails', async () => {
    vi.mocked(wikiAPI.listPages).mockRejectedValue(new Error('Kunde inte ansluta till servern.'));
    render(<App />);
    await waitFor(() => expect(screen.getByText('Kunde inte ansluta till servern.')).toBeInTheDocument());
  });

  it('shows the logged-in username once getMe resolves a user', async () => {
    const user: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(user);
    render(<App />);
    await waitFor(() => expect(screen.getByText('henrik')).toBeInTheDocument());
  });

  it('switches to the clicked page in the sidebar', async () => {
    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('Page B'));
    await waitFor(() => expect(screen.getByText('Content of Page B')).toBeInTheDocument());
    expect(wikiAPI.getPage).toHaveBeenCalledWith('page-b');
  });

  it('opens the page editor in create mode and creates a new page', async () => {
    const loggedInUser: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(loggedInUser);
    const newPage = makePage({ id: 3, slug: 'new-page', title: 'New Page' });
    vi.mocked(wikiAPI.createPage).mockResolvedValue(newPage);

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('Ny Sida'));
    expect(screen.getByText('Skapa Ny Wiki-sida')).toBeInTheDocument();

    await user.type(screen.getByPlaceholderText('T.ex. Projektarkitektur eller API-guide'), 'New Page');
    await user.type(screen.getByPlaceholderText(/Skriv din markdown här/), 'Some new content');

    vi.mocked(wikiAPI.listPages).mockResolvedValue([pageA, pageB, newPage]);
    vi.mocked(wikiAPI.getPage).mockImplementation(async (slug: string) => {
      const found = [pageA, pageB, newPage].find((p) => p.slug === slug);
      if (!found) throw new Error('not found');
      return found;
    });

    await user.click(screen.getByText('Spara Wiki-sida'));

    await waitFor(() => expect(wikiAPI.createPage).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByText('Content of New Page')).toBeInTheDocument());
  });

  it('deletes the active page after confirmation', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    const loggedInUser: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(loggedInUser);
    vi.mocked(wikiAPI.deletePage).mockResolvedValue(undefined);

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    vi.mocked(wikiAPI.listPages).mockResolvedValue([pageB]);
    await user.click(screen.getByTitle('Radera sida'));

    await waitFor(() => expect(wikiAPI.deletePage).toHaveBeenCalledWith('page-a'));
  });

  it('does not delete when confirmation is declined', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    const loggedInUser: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(loggedInUser);

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByTitle('Radera sida'));
    expect(wikiAPI.deletePage).not.toHaveBeenCalled();
  });

  it('shows revision history and reverts to a selected revision', async () => {
    vi.mocked(wikiAPI.getRevisions).mockResolvedValue([
      { id: 2, page_id: 1, title: 'Page A', content: 'v2', comment: 'second', author: 'a', created_at: '2026-01-02T00:00:00Z' },
      { id: 1, page_id: 1, title: 'Page A', content: 'v1', comment: 'first', author: 'a', created_at: '2026-01-01T00:00:00Z' },
    ]);
    vi.mocked(wikiAPI.revertRevision).mockResolvedValue(pageA);

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('Historik'));
    await waitFor(() => expect(screen.getByText(/Ändringshistorik/)).toBeInTheDocument());

    await user.click(screen.getByText('Revision #1'));
    expect(screen.getByText('v1')).toBeInTheDocument();
  });

  it('opens the login modal and updates the navbar after a successful login', async () => {
    vi.mocked(wikiAPI.login).mockResolvedValue({
      token: 't',
      user: { id: 1, username: 'admin', email: 'a@example.com', role: 'admin', created_at: '' },
    });

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('Logga in'));
    await user.type(screen.getByPlaceholderText('T.ex. admin'), 'admin');
    await user.type(screen.getByPlaceholderText('••••••••'), 'admin');
    // Two "Logga in" buttons exist now: the navbar's own, and the modal's
    // submit button — the submit button is the only one with type="submit".
    await user.click(document.querySelector('button[type="submit"]')!);

    await waitFor(() => expect(screen.getByText('admin')).toBeInTheDocument());
  });

  it('logs out and clears the active page', async () => {
    const loggedInUser: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(loggedInUser);
    vi.mocked(wikiAPI.logout).mockResolvedValue(undefined);

    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('henrik')).toBeInTheDocument());

    await user.click(screen.getByTitle('Logga ut'));
    await waitFor(() => expect(screen.getByText('Logga in')).toBeInTheDocument());
  });

  it('blocks the app behind the forced password-change modal when required', async () => {
    const lockedUser: User = { id: 1, username: 'admin', email: 'a@example.com', role: 'admin', created_at: '', must_change_password: true };
    vi.mocked(wikiAPI.getMe).mockResolvedValue(lockedUser);

    render(<App />);
    await waitFor(() => expect(screen.getByText('Byt lösenord för att fortsätta')).toBeInTheDocument());
  });

  it('opens the API documentation modal', async () => {
    const user = userEvent.setup();
    render(<App />);
    await waitFor(() => expect(screen.getByText('Content of Page A')).toBeInTheDocument());

    await user.click(screen.getByText('REST API'));
    expect(screen.getByText('REST API Interaktiv Guide')).toBeInTheDocument();
  });

  it('creates a missing page from a broken wikilink with the title prefilled', async () => {
    const withLink = makePage({ id: 1, slug: 'page-a', title: 'Page A', content: 'See [[Nonexistent Page]] here.' });
    vi.mocked(wikiAPI.getPage).mockImplementation(async (slug: string) => {
      if (slug === 'page-a') return withLink;
      throw new Error('not found');
    });

    render(<App />);
    await waitFor(() => expect(screen.getByText(/Nonexistent Page/)).toBeInTheDocument());

    fireEvent.click(screen.getByText(/Nonexistent Page/));

    await waitFor(() => expect(screen.getByDisplayValue('Nonexistent Page')).toBeInTheDocument());
  });
});
