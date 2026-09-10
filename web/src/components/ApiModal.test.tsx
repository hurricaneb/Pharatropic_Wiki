import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ApiModal } from './ApiModal';
import type { Page } from '../types';

describe('ApiModal', () => {
  it('defaults to the welcome page slug when no page is given', () => {
    render(<ApiModal onClose={() => {}} />);
    expect(screen.getByText('GET /api/v1/pages/valkommen-till-wikin')).toBeInTheDocument();
  });

  it('uses the given page slug in the example snippets', () => {
    const page = { id: 1, slug: 'my-page', title: 'My Page' } as Page;
    render(<ApiModal page={page} onClose={() => {}} />);
    expect(screen.getByText('GET /api/v1/pages/my-page')).toBeInTheDocument();
  });

  it('copies a snippet to the clipboard when its copy button is clicked', async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });

    render(<ApiModal onClose={() => {}} />);
    const copyButtons = screen.getAllByTitle('Kopiera cURL-kommando');
    await user.click(copyButtons[0]);

    expect(writeText).toHaveBeenCalled();
  });

  it('calls onClose when the close button is clicked', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(<ApiModal onClose={onClose} />);
    await user.click(screen.getByRole('button', { name: '' }));
    expect(onClose).toHaveBeenCalled();
  });
});
