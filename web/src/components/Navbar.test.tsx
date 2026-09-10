import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Navbar } from './Navbar';
import type { User } from '../types';

const baseProps = {
  searchQuery: '',
  onSearchChange: vi.fn(),
  onNewPage: vi.fn(),
  onOpenApiModal: vi.fn(),
  onHomeClick: vi.fn(),
  healthOk: true,
  currentUser: null as User | null,
  onOpenLogin: vi.fn(),
  onOpenApiKeys: vi.fn(),
  onOpenAdminUsers: vi.fn(),
  onLogout: vi.fn(),
};

describe('Navbar', () => {
  it('shows a login button when no user is logged in', () => {
    render(<Navbar {...baseProps} />);
    expect(screen.getByText('Logga in')).toBeInTheDocument();
    expect(screen.queryByText('Ny Sida')).not.toBeInTheDocument();
  });

  it('shows the username and write actions when logged in', () => {
    const user: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    render(<Navbar {...baseProps} currentUser={user} />);
    expect(screen.getByText('henrik')).toBeInTheDocument();
    expect(screen.getByText('Ny Sida')).toBeInTheDocument();
    expect(screen.getByText('API-nycklar')).toBeInTheDocument();
  });

  it('only shows the admin users button for admin roles', () => {
    const admin: User = { id: 1, username: 'admin', email: 'a@example.com', role: 'admin', created_at: '' };
    const { rerender } = render(<Navbar {...baseProps} currentUser={admin} />);
    expect(screen.getByText('Användare')).toBeInTheDocument();

    const regular: User = { id: 2, username: 'bob', email: 'b@example.com', role: 'user', created_at: '' };
    rerender(<Navbar {...baseProps} currentUser={regular} />);
    expect(screen.queryByText('Användare')).not.toBeInTheDocument();
  });

  it('calls onSearchChange as the user types', async () => {
    const user = userEvent.setup();
    const onSearchChange = vi.fn();
    render(<Navbar {...baseProps} onSearchChange={onSearchChange} />);
    await user.type(screen.getByPlaceholderText('Sök i alla wikisidor...'), 'x');
    expect(onSearchChange).toHaveBeenCalledWith('x');
  });

  it('calls onLogout when the logout button is clicked', async () => {
    const user = userEvent.setup();
    const onLogout = vi.fn();
    const loggedInUser: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    render(<Navbar {...baseProps} currentUser={loggedInUser} onLogout={onLogout} />);
    await user.click(screen.getByTitle('Logga ut'));
    expect(onLogout).toHaveBeenCalled();
  });

  it('calls onNewPage when the new-page button is clicked', async () => {
    const user = userEvent.setup();
    const onNewPage = vi.fn();
    const loggedInUser: User = { id: 1, username: 'henrik', email: 'h@example.com', role: 'user', created_at: '' };
    render(<Navbar {...baseProps} currentUser={loggedInUser} onNewPage={onNewPage} />);
    await user.click(screen.getByText('Ny Sida'));
    expect(onNewPage).toHaveBeenCalled();
  });

  it('shows a health indicator that reflects the healthOk prop', () => {
    const { container, rerender } = render(<Navbar {...baseProps} healthOk={true} />);
    const dot = container.querySelector('span[style*="border-radius: 50%"]');
    expect(dot).toHaveStyle({ backgroundColor: 'var(--success)' });

    rerender(<Navbar {...baseProps} healthOk={false} />);
    const dotAfter = container.querySelector('span[style*="border-radius: 50%"]');
    expect(dotAfter).toHaveStyle({ backgroundColor: 'var(--danger)' });
  });
});
