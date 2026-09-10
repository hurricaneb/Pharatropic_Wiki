import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ForcePasswordChangeModal } from './ForcePasswordChangeModal';

vi.mock('../api', () => ({
  wikiAPI: { changePassword: vi.fn() },
}));

import { wikiAPI } from '../api';

beforeEach(() => {
  vi.mocked(wikiAPI.changePassword).mockReset();
});

describe('ForcePasswordChangeModal', () => {
  it('rejects a new password shorter than 8 characters without calling the API', async () => {
    const user = userEvent.setup();
    render(<ForcePasswordChangeModal onSuccess={() => {}} />);

    await user.type(screen.getByLabelText('Nuvarande lösenord'), 'admin');
    await user.type(screen.getByLabelText(/Nytt lösenord/), 'short');
    await user.type(screen.getByLabelText('Bekräfta nytt lösenord'), 'short');
    await user.click(screen.getByText('Byt lösenord'));

    expect(screen.getByText('Det nya lösenordet måste vara minst 8 tecken.')).toBeInTheDocument();
    expect(wikiAPI.changePassword).not.toHaveBeenCalled();
  });

  it('rejects mismatched password confirmation without calling the API', async () => {
    const user = userEvent.setup();
    render(<ForcePasswordChangeModal onSuccess={() => {}} />);

    await user.type(screen.getByLabelText('Nuvarande lösenord'), 'admin');
    await user.type(screen.getByLabelText(/Nytt lösenord/), 'newpassword1');
    await user.type(screen.getByLabelText('Bekräfta nytt lösenord'), 'newpassword2');
    await user.click(screen.getByText('Byt lösenord'));

    expect(screen.getByText('Lösenorden matchar inte.')).toBeInTheDocument();
    expect(wikiAPI.changePassword).not.toHaveBeenCalled();
  });

  it('calls the API with the entered passwords and onSuccess on a valid submission', async () => {
    const user = userEvent.setup();
    vi.mocked(wikiAPI.changePassword).mockResolvedValue(undefined);
    const onSuccess = vi.fn();

    render(<ForcePasswordChangeModal onSuccess={onSuccess} />);
    await user.type(screen.getByLabelText('Nuvarande lösenord'), 'admin');
    await user.type(screen.getByLabelText(/Nytt lösenord/), 'newpassword1');
    await user.type(screen.getByLabelText('Bekräfta nytt lösenord'), 'newpassword1');
    await user.click(screen.getByText('Byt lösenord'));

    await waitFor(() => expect(onSuccess).toHaveBeenCalled());
    expect(wikiAPI.changePassword).toHaveBeenCalledWith('admin', 'newpassword1');
  });

  it('shows the server error message when the change fails', async () => {
    const user = userEvent.setup();
    vi.mocked(wikiAPI.changePassword).mockRejectedValue(new Error('felaktigt nuvarande lösenord'));

    render(<ForcePasswordChangeModal onSuccess={() => {}} />);
    await user.type(screen.getByLabelText('Nuvarande lösenord'), 'wrong');
    await user.type(screen.getByLabelText(/Nytt lösenord/), 'newpassword1');
    await user.type(screen.getByLabelText('Bekräfta nytt lösenord'), 'newpassword1');
    await user.click(screen.getByText('Byt lösenord'));

    await waitFor(() => expect(screen.getByText('felaktigt nuvarande lösenord')).toBeInTheDocument());
  });

  it('has no dismiss/close control, since the change is mandatory', () => {
    render(<ForcePasswordChangeModal onSuccess={() => {}} />);
    expect(screen.queryByRole('button', { name: /stäng|close|×/i })).not.toBeInTheDocument();
  });
});
