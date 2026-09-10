import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthModal } from './AuthModal';

vi.mock('../api', () => ({
  wikiAPI: { login: vi.fn() },
}));

import { wikiAPI } from '../api';

beforeEach(() => {
  vi.mocked(wikiAPI.login).mockReset();
});

describe('AuthModal', () => {
  it('calls onSuccess and onClose with the logged-in user on successful login', async () => {
    const user = userEvent.setup();
    const loggedInUser = { id: 1, username: 'admin', email: 'a@b.com', role: 'admin', created_at: '' };
    vi.mocked(wikiAPI.login).mockResolvedValue({ token: 't', user: loggedInUser });
    const onSuccess = vi.fn();
    const onClose = vi.fn();

    render(<AuthModal onClose={onClose} onSuccess={onSuccess} />);
    await user.type(screen.getByPlaceholderText('T.ex. admin'), 'admin');
    await user.type(screen.getByPlaceholderText('••••••••'), 'admin');
    await user.click(screen.getByRole('button', { name: 'Logga in' }));

    await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(loggedInUser));
    expect(onClose).toHaveBeenCalled();
    expect(wikiAPI.login).toHaveBeenCalledWith('admin', 'admin');
  });

  it('shows an error message and does not close on failed login', async () => {
    const user = userEvent.setup();
    vi.mocked(wikiAPI.login).mockRejectedValue(new Error('Felaktigt användarnamn eller lösenord'));
    const onSuccess = vi.fn();
    const onClose = vi.fn();

    render(<AuthModal onClose={onClose} onSuccess={onSuccess} />);
    await user.type(screen.getByPlaceholderText('T.ex. admin'), 'admin');
    await user.type(screen.getByPlaceholderText('••••••••'), 'wrong');
    await user.click(screen.getByRole('button', { name: 'Logga in' }));

    await waitFor(() => expect(screen.getByText('Felaktigt användarnamn eller lösenord')).toBeInTheDocument());
    expect(onSuccess).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('calls onClose when the close (X) button is clicked', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const { container } = render(<AuthModal onClose={onClose} onSuccess={() => {}} />);
    const closeButton = container.querySelector('button');
    await user.click(closeButton!);
    expect(onClose).toHaveBeenCalled();
  });
});
