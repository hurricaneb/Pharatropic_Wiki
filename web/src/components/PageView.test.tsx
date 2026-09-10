import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PageView } from './PageView';
import type { Page } from '../types';

vi.mock('../api', () => ({
  wikiAPI: {
    getBacklinks: vi.fn(),
    deleteAttachment: vi.fn(),
  },
  getApiHost: () => 'http://localhost:8080',
}));

import { wikiAPI } from '../api';

function makePage(overrides: Partial<Page> & { id: number; slug: string; title: string }): Page {
  return {
    summary: '',
    content: '',
    is_public: true,
    views: 0,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

const noop = () => {};

beforeEach(() => {
  vi.mocked(wikiAPI.getBacklinks).mockResolvedValue([]);
  vi.mocked(wikiAPI.deleteAttachment).mockResolvedValue(undefined);
});

describe('PageView', () => {
  it('renders the page title and content', () => {
    const page = makePage({ id: 1, slug: 'x', title: 'My Page', content: 'Hello world' });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    expect(screen.getByText('My Page')).toBeInTheDocument();
    expect(screen.getByText('Hello world')).toBeInTheDocument();
  });

  it('shows a public badge for a public page and a private badge otherwise', () => {
    const pub = makePage({ id: 1, slug: 'x', title: 'X', is_public: true });
    const { rerender } = render(<PageView page={pub} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    expect(screen.getByText('Publik')).toBeInTheDocument();

    const priv = makePage({ id: 2, slug: 'y', title: 'Y', is_public: false });
    rerender(<PageView page={priv} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    expect(screen.getByText('Privat')).toBeInTheDocument();
  });

  it('shows Edit/Delete actions only when a currentUser is provided', () => {
    const page = makePage({ id: 1, slug: 'x', title: 'X' });
    const { rerender } = render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    expect(screen.queryByText('Redigera')).not.toBeInTheDocument();

    rerender(
      <PageView page={page} currentUser={{ id: 1, username: 'admin', email: 'a@b.com', role: 'admin', created_at: '' }} onEdit={noop} onViewHistory={noop} onDelete={noop} />
    );
    expect(screen.getByText('Redigera')).toBeInTheDocument();
  });

  it('shows a breadcrumb with the parent title when the page has a parent', () => {
    const parent = { id: 1, slug: 'parent', title: 'Parent Page' } as Page;
    const page = makePage({ id: 2, slug: 'child', title: 'Child Page', parent });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    expect(screen.getByText('Parent Page')).toBeInTheDocument();
    expect(screen.getByTitle('Gå till "Parent Page"')).toBeInTheDocument();
  });

  it('navigates to the parent when the breadcrumb is clicked', async () => {
    const user = userEvent.setup();
    const onSelectPage = vi.fn();
    const parent = { id: 1, slug: 'parent', title: 'Parent Page' } as Page;
    const page = makePage({ id: 2, slug: 'child', title: 'Child Page', parent });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} onSelectPage={onSelectPage} />);

    await user.click(screen.getByText('Parent Page'));
    expect(onSelectPage).toHaveBeenCalledWith('parent');
  });

  it('shows no breadcrumb for a top-level page', () => {
    const page = makePage({ id: 1, slug: 'x', title: 'Top Level' });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    expect(screen.queryByText('Top Level')).toBeTruthy(); // the h1 title itself
    expect(document.querySelector('[title^="Gå till"]')).not.toBeInTheDocument();
  });

  it('shows a subpages section listing each child', () => {
    const page = makePage({
      id: 1,
      slug: 'parent',
      title: 'Parent',
      children: [
        makePage({ id: 2, slug: 'a', title: 'Child A', is_public: true }),
        makePage({ id: 3, slug: 'b', title: 'Child B', is_public: false }),
      ],
    });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    expect(screen.getByText('Undersidor (2)')).toBeInTheDocument();
    expect(screen.getByText('Child A')).toBeInTheDocument();
    expect(screen.getByText('Child B')).toBeInTheDocument();
  });

  it('navigates when a subpage card is clicked', async () => {
    const user = userEvent.setup();
    const onSelectPage = vi.fn();
    const page = makePage({
      id: 1,
      slug: 'parent',
      title: 'Parent',
      children: [makePage({ id: 2, slug: 'child-a', title: 'Child A' })],
    });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} onSelectPage={onSelectPage} />);

    await user.click(screen.getByText('Child A'));
    expect(onSelectPage).toHaveBeenCalledWith('child-a');
  });

  it('shows no subpages section when there are no children', () => {
    const page = makePage({ id: 1, slug: 'x', title: 'X' });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    expect(screen.queryByText(/Undersidor/)).not.toBeInTheDocument();
  });

  it('renders attachments with size formatting and a download link', () => {
    const page = makePage({
      id: 1,
      slug: 'x',
      title: 'X',
      attachments: [
        { id: 1, page_id: 1, filename: 'f.png', original_name: 'photo.png', file_path: '/uploads/f.png', mime_type: 'image/png', file_size: 2048, created_at: '' },
      ],
    });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    expect(screen.getByText('photo.png')).toBeInTheDocument();
    expect(screen.getByText('2.0 KB')).toBeInTheDocument();
    expect(screen.getByText('Öppna').closest('a')).toHaveAttribute('href', 'http://localhost:8080/uploads/f.png');
  });

  it('fetches and displays backlinks for the current page', async () => {
    vi.mocked(wikiAPI.getBacklinks).mockResolvedValue([
      { id: 5, slug: 'linker', title: 'Linker Page' } as Page,
    ]);
    const page = makePage({ id: 1, slug: 'x', title: 'X' });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);

    await waitFor(() => expect(screen.getByText('Linker Page')).toBeInTheDocument());
    expect(wikiAPI.getBacklinks).toHaveBeenCalledWith('x');
  });

  it('shows no backlinks section when the fetch fails', async () => {
    vi.mocked(wikiAPI.getBacklinks).mockRejectedValue(new Error('network error'));
    const page = makePage({ id: 1, slug: 'x', title: 'X' });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);

    await waitFor(() => expect(wikiAPI.getBacklinks).toHaveBeenCalled());
    expect(screen.queryByText(/Backlinks/)).not.toBeInTheDocument();
  });

  it('renders an existing wikilink as a clickable internal link', () => {
    // fireEvent is used instead of userEvent here: userEvent's more
    // realistic pointer-event simulation lets jsdom's real (unimplemented)
    // anchor navigation win the race against React's onClick/preventDefault
    // for a genuine <a href> element, even though the component itself
    // correctly calls preventDefault — fireEvent dispatches the click
    // synchronously and exercises the same handler without that flakiness.
    const onSelectPage = vi.fn();
    const target = makePage({ id: 2, slug: 'malet', title: 'Målet' });
    const page = makePage({ id: 1, slug: 'x', title: 'X', content: 'See [[Målet]] for info.' });
    render(<PageView page={page} pages={[target]} onEdit={noop} onViewHistory={noop} onDelete={noop} onSelectPage={onSelectPage} />);

    const link = screen.getByText('Målet');
    expect(link).toHaveClass('wikilink-exists');
    fireEvent.click(link);
    expect(onSelectPage).toHaveBeenCalledWith('malet');
  });

  it('renders a normal (non-wikilink) markdown link unchanged, opening in a new tab', () => {
    const page = makePage({ id: 1, slug: 'x', title: 'X', content: '[External](https://example.com)' });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    const link = screen.getByText('External');
    expect(link).toHaveAttribute('href', 'https://example.com');
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noreferrer');
  });

  it('formats attachment sizes in B, KB and MB as appropriate', () => {
    const page = makePage({
      id: 1,
      slug: 'x',
      title: 'X',
      attachments: [
        { id: 1, page_id: 1, filename: 'a', original_name: 'small.txt', file_path: '/uploads/a', mime_type: 'text/plain', file_size: 500, created_at: '' },
        { id: 2, page_id: 1, filename: 'b', original_name: 'big.zip', file_path: '/uploads/b', mime_type: 'application/zip', file_size: 5 * 1024 * 1024, created_at: '' },
      ],
    });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);
    expect(screen.getByText('500 B')).toBeInTheDocument();
    expect(screen.getByText('5.0 MB')).toBeInTheDocument();
  });

  it('copies the markdown snippet for an attachment to the clipboard', () => {
    // fireEvent, not userEvent: clicking a button whose content is an SVG
    // icon trips the same userEvent pointer-simulation quirk seen with the
    // wikilink anchors above, for reasons unrelated to the component logic.
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });

    const page = makePage({
      id: 1,
      slug: 'x',
      title: 'X',
      attachments: [{ id: 1, page_id: 1, filename: 'a', original_name: 'photo.png', file_path: '/uploads/a', mime_type: 'image/png', file_size: 10, created_at: '' }],
    });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);

    fireEvent.click(screen.getByTitle('Kopiera Markdown-länk'));
    expect(writeText).toHaveBeenCalledWith('![photo.png](/uploads/a)');
  });

  it('deletes an attachment after confirmation and refreshes the page', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    const user = userEvent.setup();
    const onRefreshPage = vi.fn();

    const page = makePage({
      id: 1,
      slug: 'x',
      title: 'X',
      attachments: [{ id: 1, page_id: 1, filename: 'a', original_name: 'photo.png', file_path: '/uploads/a', mime_type: 'image/png', file_size: 10, created_at: '' }],
    });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} onRefreshPage={onRefreshPage} />);

    await user.click(screen.getByTitle('Ta bort bilaga'));
    await waitFor(() => expect(wikiAPI.deleteAttachment).toHaveBeenCalledWith(1));
    expect(onRefreshPage).toHaveBeenCalled();
  });

  it('does not delete an attachment when confirmation is declined', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    const user = userEvent.setup();

    const page = makePage({
      id: 1,
      slug: 'x',
      title: 'X',
      attachments: [{ id: 1, page_id: 1, filename: 'a', original_name: 'photo.png', file_path: '/uploads/a', mime_type: 'image/png', file_size: 10, created_at: '' }],
    });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);

    await user.click(screen.getByTitle('Ta bort bilaga'));
    expect(wikiAPI.deleteAttachment).not.toHaveBeenCalled();
  });

  it('shows an alert when deleting an attachment fails', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    vi.mocked(wikiAPI.deleteAttachment).mockRejectedValue(new Error('Misslyckades att radera bilagan.'));
    const user = userEvent.setup();

    const page = makePage({
      id: 1,
      slug: 'x',
      title: 'X',
      attachments: [{ id: 1, page_id: 1, filename: 'a', original_name: 'photo.png', file_path: '/uploads/a', mime_type: 'image/png', file_size: 10, created_at: '' }],
    });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} />);

    await user.click(screen.getByTitle('Ta bort bilaga'));
    await waitFor(() => expect(window.alert).toHaveBeenCalledWith('Misslyckades att radera bilagan.'));
  });

  it('navigates when a backlink card is clicked', async () => {
    const onSelectPage = vi.fn();
    vi.mocked(wikiAPI.getBacklinks).mockResolvedValue([{ id: 9, slug: 'linker', title: 'Linker Page' } as Page]);
    const user = userEvent.setup();
    const page = makePage({ id: 1, slug: 'x', title: 'X' });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} onSelectPage={onSelectPage} />);

    await waitFor(() => expect(screen.getByText('Linker Page')).toBeInTheDocument());
    await user.click(screen.getByText('Linker Page'));
    expect(onSelectPage).toHaveBeenCalledWith('linker');
  });

  it('renders a missing wikilink and offers to create it', () => {
    const onCreateMissingPage = vi.fn();
    const page = makePage({ id: 1, slug: 'x', title: 'X', content: 'See [[Does Not Exist]] here.' });
    render(<PageView page={page} onEdit={noop} onViewHistory={noop} onDelete={noop} onCreateMissingPage={onCreateMissingPage} />);

    const link = screen.getByText(/Does Not Exist/);
    expect(link.closest('a')).toHaveClass('wikilink-missing');
    fireEvent.click(link);
    expect(onCreateMissingPage).toHaveBeenCalledWith('Does Not Exist');
  });
});
