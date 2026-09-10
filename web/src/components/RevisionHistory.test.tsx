import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { RevisionHistory } from './RevisionHistory';
import type { Page, Revision } from '../types';

const page: Page = {
  id: 1,
  slug: 'x',
  title: 'X Page',
  summary: '',
  content: '',
  is_public: true,
  views: 0,
  created_at: '',
  updated_at: '',
};

function makeRevisions(): Revision[] {
  return [
    { id: 3, page_id: 1, title: 'X Page', content: 'newest content', comment: 'Third edit', author: 'admin', created_at: '2026-03-03T00:00:00Z' },
    { id: 2, page_id: 1, title: 'X Page', content: 'middle content', comment: 'Second edit', author: 'admin', created_at: '2026-02-02T00:00:00Z' },
    { id: 1, page_id: 1, title: 'X Page', content: 'oldest content', comment: '', author: 'admin', created_at: '2026-01-01T00:00:00Z' },
  ];
}

beforeEach(() => {
  vi.spyOn(window, 'confirm').mockReturnValue(true);
  vi.spyOn(window, 'alert').mockImplementation(() => {});
});

describe('RevisionHistory', () => {
  it('shows a placeholder when there are no revisions', () => {
    render(<RevisionHistory page={page} revisions={[]} onBack={() => {}} />);
    expect(screen.getByText('Välj en revision till vänster för att granska innehållet.')).toBeInTheDocument();
  });

  it('selects the most recent revision by default and labels it "Nuvarande"', () => {
    render(<RevisionHistory page={page} revisions={makeRevisions()} onBack={() => {}} />);
    expect(screen.getByText('newest content')).toBeInTheDocument();
    expect(screen.getByText('Nuvarande')).toBeInTheDocument();
  });

  it('numbers revisions in chronological order (oldest = #1)', () => {
    render(<RevisionHistory page={page} revisions={makeRevisions()} onBack={() => {}} />);
    expect(screen.getByText('Revision #3')).toBeInTheDocument();
    expect(screen.getByText('Revision #1')).toBeInTheDocument();
  });

  it('shows a fallback label for a revision with no comment', () => {
    render(<RevisionHistory page={page} revisions={makeRevisions()} onBack={() => {}} />);
    expect(screen.getByText('Ingen kommentar')).toBeInTheDocument();
  });

  it('rewrites /uploads/ links in the previewed content to point at the API host', () => {
    const revs = [
      { id: 1, page_id: 1, title: 'X Page', content: '![img](/uploads/pic.png)', comment: '', author: 'admin', created_at: '2026-01-01T00:00:00Z' },
    ];
    const { container } = render(<RevisionHistory page={page} revisions={revs} onBack={() => {}} />);
    const img = container.querySelector('img');
    expect(img?.getAttribute('src')).toContain('/uploads/pic.png');
  });

  it('switches the preview when a different revision is clicked', async () => {
    const user = userEvent.setup();
    render(<RevisionHistory page={page} revisions={makeRevisions()} onBack={() => {}} />);

    await user.click(screen.getByText('Revision #1'));
    expect(screen.getByText('oldest content')).toBeInTheDocument();
  });

  it('calls onBack when the back button is clicked', async () => {
    const user = userEvent.setup();
    const onBack = vi.fn();
    render(<RevisionHistory page={page} revisions={makeRevisions()} onBack={onBack} />);
    await user.click(screen.getByText('Tillbaka till sidan'));
    expect(onBack).toHaveBeenCalled();
  });

  it('does not show a revert button when onRevertRevision is not provided', () => {
    render(<RevisionHistory page={page} revisions={makeRevisions()} onBack={() => {}} />);
    expect(screen.queryByText('Återställ till denna version')).not.toBeInTheDocument();
  });

  it('reverts the selected revision after confirmation', async () => {
    const user = userEvent.setup();
    const onRevertRevision = vi.fn().mockResolvedValue(undefined);
    render(<RevisionHistory page={page} revisions={makeRevisions()} onBack={() => {}} onRevertRevision={onRevertRevision} />);

    await user.click(screen.getByText('Återställ till denna version'));
    await waitFor(() => expect(onRevertRevision).toHaveBeenCalledWith(3));
    expect(window.confirm).toHaveBeenCalled();
  });

  it('does not revert when the confirmation is declined', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    const user = userEvent.setup();
    const onRevertRevision = vi.fn();
    render(<RevisionHistory page={page} revisions={makeRevisions()} onBack={() => {}} onRevertRevision={onRevertRevision} />);

    await user.click(screen.getByText('Återställ till denna version'));
    expect(onRevertRevision).not.toHaveBeenCalled();
  });

  it('shows an alert when reverting fails', async () => {
    const user = userEvent.setup();
    const onRevertRevision = vi.fn().mockRejectedValue(new Error('Kunde inte återställa'));
    render(<RevisionHistory page={page} revisions={makeRevisions()} onBack={() => {}} onRevertRevision={onRevertRevision} />);

    await user.click(screen.getByText('Återställ till denna version'));
    await waitFor(() => expect(window.alert).toHaveBeenCalledWith('Kunde inte återställa'));
  });
});
