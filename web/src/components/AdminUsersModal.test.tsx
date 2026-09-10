import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AdminUsersModal } from './AdminUsersModal';

vi.mock('../api', () => ({
  wikiAPI: { adminListUsers: vi.fn(), adminCreateUser: vi.fn() },
}));

import { wikiAPI } from '../api';

const users = [
  { id: 1, username: 'admin', email: 'admin@example.com', role: 'admin', created_at: '2026-01-01T00:00:00Z' },
  { id: 2, username: 'bob', email: 'bob@example.com', role: 'user', created_at: '2026-01-02T00:00:00Z' },
];

beforeEach(() => {
  vi.mocked(wikiAPI.adminListUsers).mockReset().mockResolvedValue(users);
  vi.mocked(wikiAPI.adminCreateUser).mockReset();
});

describe('AdminUsersModal', () => {
  it('loads and displays the existing users', async () => {
    render(<AdminUsersModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('Befintliga Användare (2)')).toBeInTheDocument());
    expect(screen.getByText('bob')).toBeInTheDocument();
    expect(screen.getByText('Befintliga Användare (2)')).toBeInTheDocument();
  });

  it('shows an error message when loading users fails', async () => {
    vi.mocked(wikiAPI.adminListUsers).mockRejectedValue(new Error('Kunde inte hämta användarlista'));
    render(<AdminUsersModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('Kunde inte hämta användarlista')).toBeInTheDocument());
  });

  it('creates a new user and shows a success message, then refreshes the list', async () => {
    const user = userEvent.setup();
    vi.mocked(wikiAPI.adminCreateUser).mockResolvedValue({ id: 3, username: 'ny', email: 'ny@example.com', role: 'user', created_at: '' });
    render(<AdminUsersModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('Befintliga Användare (2)')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Ex: henrik'), 'ny');
    await user.type(screen.getByPlaceholderText('henrik@pharatropic.local'), 'ny@example.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'pw123456');
    await user.click(screen.getByText('Skapa Konto'));

    await waitFor(() => expect(screen.getByText('Användarkontot "ny" har skapats!')).toBeInTheDocument());
    expect(wikiAPI.adminCreateUser).toHaveBeenCalledWith({ username: 'ny', email: 'ny@example.com', password: 'pw123456', role: 'user' });
    expect(wikiAPI.adminListUsers).toHaveBeenCalledTimes(2); // initial + refresh after create
  });

  it('shows an error message when user creation fails', async () => {
    const user = userEvent.setup();
    vi.mocked(wikiAPI.adminCreateUser).mockRejectedValue(new Error('en användare med det namnet finns redan'));
    render(<AdminUsersModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('Befintliga Användare (2)')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Ex: henrik'), 'admin');
    await user.type(screen.getByPlaceholderText('henrik@pharatropic.local'), 'x@example.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'pw123456');
    await user.click(screen.getByText('Skapa Konto'));

    await waitFor(() => expect(screen.getByText('en användare med det namnet finns redan')).toBeInTheDocument());
  });

  it('calls onClose when the close button is clicked', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const { container } = render(<AdminUsersModal onClose={onClose} />);
    await waitFor(() => expect(screen.getByText('Befintliga Användare (2)')).toBeInTheDocument());
    await user.click(container.querySelector('button')!);
    expect(onClose).toHaveBeenCalled();
  });
});
