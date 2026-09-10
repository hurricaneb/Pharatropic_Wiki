import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PageEditor } from './PageEditor';
import type { Page } from '../types';

vi.mock('../api', () => ({
  wikiAPI: { uploadAttachment: vi.fn() },
  getApiHost: () => 'http://localhost:8080',
}));

import { wikiAPI } from '../api';

beforeEach(() => {
  vi.mocked(wikiAPI.uploadAttachment).mockReset();
});

function makePage(overrides: Partial<Page> & { id: number; slug: string; title: string }): Page {
  return {
    summary: '',
    content: '',
    is_public: false,
    views: 0,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('PageEditor', () => {
  it('shows a validation error and does not call onSave when title/content are only whitespace', async () => {
    // The <textarea>/<input> both carry the native `required` attribute,
    // which blocks submission of a truly empty value before React ever
    // sees it — so the only way to reach the component's own .trim() guard
    // is a value that is non-empty but whitespace-only.
    const user = userEvent.setup();
    const onSave = vi.fn();
    render(<PageEditor onSave={onSave} onCancel={() => {}} />);

    await user.type(screen.getByPlaceholderText('T.ex. Projektarkitektur eller API-guide'), '   ');
    await user.type(screen.getByPlaceholderText(/Skriv din markdown här/), '   ');
    await user.click(screen.getByText('Spara Wiki-sida'));

    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByText('Titel och innehåll kan inte vara tomma.')).toBeInTheDocument();
  });

  it('submits a new page with the entered title, content and default parent (none)', async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<PageEditor onSave={onSave} onCancel={() => {}} />);

    await user.type(screen.getByPlaceholderText('T.ex. Projektarkitektur eller API-guide'), 'My New Page');
    await user.type(screen.getByPlaceholderText(/Skriv din markdown här/), 'Some content');
    await user.click(screen.getByText('Spara Wiki-sida'));

    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    const payload = onSave.mock.calls[0][0];
    expect(payload.title).toBe('My New Page');
    expect(payload.content).toBe('Some content');
    expect(payload.parent_slug).toBe('');
  });

  it('parses comma-separated tags, trimming and dropping empty entries', async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<PageEditor onSave={onSave} onCancel={() => {}} />);

    await user.type(screen.getByPlaceholderText('T.ex. Projektarkitektur eller API-guide'), 'T');
    await user.type(screen.getByPlaceholderText(/Skriv din markdown här/), 'C');
    await user.type(screen.getByPlaceholderText('T.ex. api, guide, go, react'), 'go,  , react ,api');
    await user.click(screen.getByText('Spara Wiki-sida'));

    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(onSave.mock.calls[0][0].tags).toEqual(['go', 'react', 'api']);
  });

  it('offers only top-level pages (excluding itself) as parent options when creating a page', () => {
    const pages: Page[] = [
      makePage({ id: 1, slug: 'top-a', title: 'Top A' }),
      makePage({ id: 2, slug: 'top-b', title: 'Top B' }),
      makePage({ id: 3, slug: 'child-of-a', title: 'Child Of A', parent_id: 1 }),
    ];
    render(<PageEditor pages={pages} onSave={vi.fn()} onCancel={() => {}} />);

    const select = screen.getByRole('combobox') as HTMLSelectElement;
    const optionLabels = Array.from(select.options).map((o) => o.textContent);
    expect(optionLabels).toContain('Top A');
    expect(optionLabels).toContain('Top B');
    expect(optionLabels).not.toContain('Child Of A');
  });

  it('excludes the page being edited from its own parent options', () => {
    const editing = makePage({ id: 1, slug: 'self', title: 'Self Page' });
    const pages: Page[] = [editing, makePage({ id: 2, slug: 'other', title: 'Other Top Level' })];
    render(<PageEditor initialPage={editing} pages={pages} onSave={vi.fn()} onCancel={() => {}} />);

    const select = screen.getByRole('combobox') as HTMLSelectElement;
    const optionLabels = Array.from(select.options).map((o) => o.textContent);
    expect(optionLabels).not.toContain('Self Page');
    expect(optionLabels).toContain('Other Top Level');
  });

  it('shows an explanatory note instead of the selector for a page that already has subpages', () => {
    const parent = makePage({
      id: 1,
      slug: 'parent',
      title: 'Parent',
      children: [makePage({ id: 2, slug: 'child', title: 'Child', parent_id: 1 })],
    });
    render(<PageEditor initialPage={parent} pages={[parent]} onSave={vi.fn()} onCancel={() => {}} />);

    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
    expect(screen.getByText(/har egna undersidor och kan därför inte bli en undersida själv/)).toBeInTheDocument();
  });

  it('preselects the current parent when editing a subpage', () => {
    const parent = makePage({ id: 1, slug: 'parent', title: 'Parent Page' });
    const child = makePage({ id: 2, slug: 'child', title: 'Child Page', parent_id: 1, parent });
    render(<PageEditor initialPage={child} pages={[parent, child]} onSave={vi.fn()} onCancel={() => {}} />);

    const select = screen.getByRole('combobox') as HTMLSelectElement;
    expect(select.value).toBe('parent');
  });

  it('forces parent_slug to empty on submit when the page has its own subpages, even if state was stale', async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);
    const parent = makePage({
      id: 1,
      slug: 'parent',
      title: 'Parent',
      content: 'existing',
      children: [makePage({ id: 2, slug: 'child', title: 'Child', parent_id: 1 })],
    });
    render(<PageEditor initialPage={parent} pages={[parent]} onSave={onSave} onCancel={() => {}} />);

    await user.click(screen.getByText('Spara Wiki-sida'));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(onSave.mock.calls[0][0].parent_slug).toBe('');
  });

  it('populates fields from initialPage when editing', () => {
    const page = makePage({ id: 1, slug: 'edit-me', title: 'Edit Me', content: 'body text', summary: 'a summary', is_public: true });
    render(<PageEditor initialPage={page} onSave={vi.fn()} onCancel={() => {}} />);

    expect(screen.getByDisplayValue('Edit Me')).toBeInTheDocument();
    expect(screen.getByDisplayValue('a summary')).toBeInTheDocument();
    expect(screen.getByText(/Redigera: Edit Me/)).toBeInTheDocument();
  });

  it('calls onCancel when the cancel button is clicked', async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(<PageEditor onSave={vi.fn()} onCancel={onCancel} />);
    await user.click(screen.getByText('Avbryt'));
    expect(onCancel).toHaveBeenCalled();
  });

  it('switches between editor, split and preview tabs', async () => {
    const user = userEvent.setup();
    render(<PageEditor onSave={vi.fn()} onCancel={() => {}} />);

    await user.click(screen.getByText('Redigerare'));
    expect(screen.queryByText('*Ingen förhandsgranskning än...*')).not.toBeInTheDocument();

    await user.click(screen.getByText('Förhandsgranska'));
    expect(screen.queryByPlaceholderText(/Skriv din markdown här/)).not.toBeInTheDocument();

    await user.click(screen.getByText('Delad Vy'));
    expect(screen.getByPlaceholderText(/Skriv din markdown här/)).toBeInTheDocument();
  });

  it('toggles page visibility via the radio labels', async () => {
    const user = userEvent.setup();
    render(<PageEditor onSave={vi.fn()} onCancel={() => {}} />);

    // Defaults to private for a new page
    expect(screen.getByText('🔒 Privat sida')).toBeInTheDocument();
    await user.click(screen.getByText('🌐 Publik sida'));
    const publicRadio = screen.getAllByRole('radio')[1] as HTMLInputElement;
    expect(publicRadio.checked).toBe(true);

    await user.click(screen.getByText('🔒 Privat sida'));
    const privateRadio = screen.getAllByRole('radio')[0] as HTMLInputElement;
    expect(privateRadio.checked).toBe(true);
  });

  it('inserts a wikilink placeholder into the content on button click', async () => {
    const user = userEvent.setup();
    render(<PageEditor onSave={vi.fn()} onCancel={() => {}} />);

    const textarea = screen.getByPlaceholderText(/Skriv din markdown här/) as HTMLTextAreaElement;
    await user.type(textarea, 'Existing');
    await user.click(screen.getByText('+ [[Wikilänk]]'));

    expect(textarea.value).toContain('[[Sidtitel]]');
  });

  it('updates the summary field', async () => {
    const user = userEvent.setup();
    render(<PageEditor onSave={vi.fn()} onCancel={() => {}} />);
    await user.type(screen.getByPlaceholderText('Kort beskrivning av innehållet...'), 'A summary');
    expect(screen.getByDisplayValue('A summary')).toBeInTheDocument();
  });

  it('does not show the attach-file control when creating a new page', () => {
    render(<PageEditor onSave={vi.fn()} onCancel={() => {}} />);
    expect(screen.queryByText(/Bifoga fil/)).not.toBeInTheDocument();
  });

  it('uploads a file and appends the returned markdown snippet to the content', async () => {
    const user = userEvent.setup();
    vi.mocked(wikiAPI.uploadAttachment).mockResolvedValue({
      data: { id: 1, page_id: 1, filename: 'f.png', original_name: 'f.png', file_path: '/uploads/f.png', mime_type: 'image/png', file_size: 10, created_at: '' },
      markdown: '![f.png](/uploads/f.png)',
    });
    const existing = makePage({ id: 1, slug: 'existing', title: 'Existing', content: 'Start' });
    const { container } = render(<PageEditor initialPage={existing} onSave={vi.fn()} onCancel={() => {}} />);

    const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(['content'], 'f.png', { type: 'image/png' });
    await user.upload(fileInput, file);

    await waitFor(() => expect(wikiAPI.uploadAttachment).toHaveBeenCalledWith('existing', file));
    await waitFor(() => expect(screen.getByDisplayValue(/!\[f\.png\]\(\/uploads\/f\.png\)/)).toBeInTheDocument());
  });

  it('shows an error message when file upload fails', async () => {
    const user = userEvent.setup();
    vi.mocked(wikiAPI.uploadAttachment).mockRejectedValue(new Error('Misslyckades att ladda upp filen.'));
    const existing = makePage({ id: 1, slug: 'existing', title: 'Existing', content: 'Start' });
    const { container } = render(<PageEditor initialPage={existing} onSave={vi.fn()} onCancel={() => {}} />);

    const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(['content'], 'f.png', { type: 'image/png' });
    await user.upload(fileInput, file);

    await waitFor(() => expect(screen.getByText('Misslyckades att ladda upp filen.')).toBeInTheDocument());
  });

  it('shows an error message when onSave rejects', async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockRejectedValue(new Error('Servern svarade inte'));
    render(<PageEditor onSave={onSave} onCancel={() => {}} />);

    await user.type(screen.getByPlaceholderText('T.ex. Projektarkitektur eller API-guide'), 'T');
    await user.type(screen.getByPlaceholderText(/Skriv din markdown här/), 'C');
    await user.click(screen.getByText('Spara Wiki-sida'));

    await waitFor(() => expect(screen.getByText('Servern svarade inte')).toBeInTheDocument());
  });
});
