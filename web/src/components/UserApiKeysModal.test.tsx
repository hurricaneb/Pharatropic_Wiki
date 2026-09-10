import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { UserApiKeysModal } from './UserApiKeysModal';

vi.mock('../api', () => ({
  wikiAPI: {
    listUserApiKeys: vi.fn(),
    createUserApiKey: vi.fn(),
    revokeUserApiKey: vi.fn(),
  },
}));

import { wikiAPI } from '../api';

const keys = [
  { id: 1, user_id: 1, name: 'CI Key', prefix: 'ptc_key_abc...', active: true, expires_at: null, last_used_at: null, created_at: '2026-01-01T00:00:00Z' },
];

beforeEach(() => {
  vi.mocked(wikiAPI.listUserApiKeys).mockReset().mockResolvedValue(keys);
  vi.mocked(wikiAPI.createUserApiKey).mockReset();
  vi.mocked(wikiAPI.revokeUserApiKey).mockReset();
  vi.spyOn(window, 'confirm').mockReturnValue(true);
});

describe('UserApiKeysModal', () => {
  it('loads and displays existing keys by their prefix', async () => {
    render(<UserApiKeysModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('CI Key')).toBeInTheDocument());
    expect(screen.getByText('ptc_key_abc...')).toBeInTheDocument();
  });

  it('shows an empty state when there are no keys', async () => {
    vi.mocked(wikiAPI.listUserApiKeys).mockResolvedValue([]);
    render(<UserApiKeysModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('Inga API-nycklar skapade än.')).toBeInTheDocument());
  });

  it('shows an error message when loading keys fails', async () => {
    vi.mocked(wikiAPI.listUserApiKeys).mockRejectedValue(new Error('Kunde inte hämta API-nycklar'));
    render(<UserApiKeysModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('Kunde inte hämta API-nycklar')).toBeInTheDocument());
  });

  it('creates a new key and reveals the raw secret exactly once', async () => {
    const user = userEvent.setup();
    vi.mocked(wikiAPI.createUserApiKey).mockResolvedValue({
      key: 'ptc_key_rawsecret123',
      api_key: 'ptc_key_rawsecret123',
      data: { id: 2, user_id: 1, name: 'New Key', prefix: 'ptc_key_raw...', active: true, expires_at: null, last_used_at: null, created_at: '' },
    });
    render(<UserApiKeysModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('CI Key')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Ex: CI/CD Deployment Key'), 'New Key');
    await user.click(screen.getByText('Skapa'));

    await waitFor(() => expect(screen.getByDisplayValue('ptc_key_rawsecret123')).toBeInTheDocument());
    expect(wikiAPI.createUserApiKey).toHaveBeenCalledWith('New Key', '30d');
  });

  it('shows an error message when key creation fails', async () => {
    const user = userEvent.setup();
    vi.mocked(wikiAPI.createUserApiKey).mockRejectedValue(new Error('Nyckelnamn krävs'));
    render(<UserApiKeysModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('CI Key')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Ex: CI/CD Deployment Key'), 'x');
    await user.click(screen.getByText('Skapa'));

    await waitFor(() => expect(screen.getByText('Nyckelnamn krävs')).toBeInTheDocument());
  });

  it('revokes a key after confirmation and refreshes the list', async () => {
    const user = userEvent.setup();
    vi.mocked(wikiAPI.revokeUserApiKey).mockResolvedValue(undefined);
    render(<UserApiKeysModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('CI Key')).toBeInTheDocument());

    await user.click(screen.getByTitle('Återkalla / Ta bort nyckel'));

    await waitFor(() => expect(wikiAPI.revokeUserApiKey).toHaveBeenCalledWith(1));
    expect(wikiAPI.listUserApiKeys).toHaveBeenCalledTimes(2);
  });

  it('shows an alert when revoking a key fails', async () => {
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    vi.mocked(wikiAPI.revokeUserApiKey).mockRejectedValue(new Error('Misslyckades att återkalla nyckeln'));
    const user = userEvent.setup();
    render(<UserApiKeysModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('CI Key')).toBeInTheDocument());

    await user.click(screen.getByTitle('Återkalla / Ta bort nyckel'));
    await waitFor(() => expect(window.alert).toHaveBeenCalledWith('Misslyckades att återkalla nyckeln'));
  });

  it('reveals the new secret once, honors the chosen expiry, and can be dismissed', async () => {
    // Not asserting the clipboard write here: userEvent.setup() installs its
    // own navigator.clipboard stub for user.copy()/paste(), which takes
    // precedence over a manually-defined one in this environment. The same
    // one-line writeText(...) call is already verified working end-to-end
    // in ApiModal's and PageView's copy-button tests.
    vi.mocked(wikiAPI.createUserApiKey).mockResolvedValue({
      key: 'ptc_key_reveal',
      api_key: 'ptc_key_reveal',
      data: { id: 2, user_id: 1, name: 'Reveal Key', prefix: 'ptc_key_rev...', active: true, expires_at: null, last_used_at: null, created_at: '' },
    });

    const user = userEvent.setup();
    render(<UserApiKeysModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('CI Key')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Ex: CI/CD Deployment Key'), 'Reveal Key');
    await user.selectOptions(screen.getByRole('combobox'), '7d');
    await user.click(screen.getByText('Skapa'));
    await waitFor(() => expect(screen.getByDisplayValue('ptc_key_reveal')).toBeInTheDocument());
    expect(wikiAPI.createUserApiKey).toHaveBeenCalledWith('Reveal Key', '7d');

    await user.click(screen.getByText('Stäng hemlig visning'));
    expect(screen.queryByDisplayValue('ptc_key_reveal')).not.toBeInTheDocument();
  });

  it('does not revoke when confirmation is declined', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    const user = userEvent.setup();
    render(<UserApiKeysModal onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText('CI Key')).toBeInTheDocument());

    await user.click(screen.getByTitle('Återkalla / Ta bort nyckel'));
    expect(wikiAPI.revokeUserApiKey).not.toHaveBeenCalled();
  });
});
