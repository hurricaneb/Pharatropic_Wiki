import React from 'react';
import { BookOpen, Plus, Search, Terminal, Key, LogIn, LogOut, Users } from 'lucide-react';
import type { User } from '../types';

interface NavbarProps {
  searchQuery: string;
  onSearchChange: (q: string) => void;
  onNewPage: () => void;
  onOpenApiModal: () => void;
  onHomeClick: () => void;
  healthOk: boolean;
  currentUser: User | null;
  onOpenLogin: () => void;
  onOpenApiKeys: () => void;
  onOpenAdminUsers: () => void;
  onLogout: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  searchQuery,
  onSearchChange,
  onNewPage,
  onOpenApiModal,
  onHomeClick,
  healthOk,
  currentUser,
  onOpenLogin,
  onOpenApiKeys,
  onOpenAdminUsers,
  onLogout,
}) => {
  return (
    <header className="glass-panel" style={{ borderRadius: 0, borderTop: 0, borderLeft: 0, borderRight: 0, padding: '14px 28px' }}>
      <div style={{ maxWidth: '1400px', margin: '0 auto', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '20px' }}>
        {/* Brand / Logo */}
        <div 
          onClick={onHomeClick} 
          style={{ display: 'flex', alignItems: 'center', gap: '12px', cursor: 'pointer', userSelect: 'none' }}
        >
          <div style={{ 
            background: 'linear-gradient(135deg, #6366f1, #06b6d4)', 
            padding: '10px', 
            borderRadius: '12px', 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            boxShadow: '0 4px 12px rgba(99, 102, 241, 0.4)'
          }}>
            <BookOpen size={24} color="#fff" />
          </div>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <h1 style={{ fontSize: '1.3rem', fontWeight: 800, background: 'linear-gradient(90deg, #fff, #a5b4fc)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', lineHeight: 1.2 }}>
                Pharatropic Wiki
              </h1>
              <span style={{ fontSize: '0.68rem', fontWeight: 700, padding: '2px 6px', borderRadius: '4px', background: 'rgba(99, 102, 241, 0.25)', color: '#a5b4fc', border: '1px solid rgba(99, 102, 241, 0.4)' }}>
                PTC Wiki
              </span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.75rem', color: 'var(--text-muted)' }}>
              <span style={{ display: 'inline-block', width: 7, height: 7, borderRadius: '50%', backgroundColor: healthOk ? 'var(--success)' : 'var(--danger)' }}></span>
              Go REST API + React UI
            </div>
          </div>
        </div>

        {/* Search Bar */}
        <div style={{ flex: 1, maxWidth: '440px', position: 'relative' }}>
          <Search size={18} color="var(--text-muted)" style={{ position: 'absolute', left: 14, top: '50%', transform: 'translateY(-50%)' }} />
          <input
            type="text"
            className="input-field"
            placeholder="Sök i alla wikisidor..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            style={{ paddingLeft: '42px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)' }}
          />
        </div>

        {/* Action Buttons & User Menu */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <button className="btn btn-secondary" onClick={onOpenApiModal} title="Visa REST API dokumentation" style={{ padding: '6px 12px', fontSize: '0.82rem' }}>
            <Terminal size={15} />
            <span>REST API</span>
          </button>
          
          <button className="btn btn-primary" onClick={onNewPage} style={{ padding: '6px 14px', fontSize: '0.85rem' }}>
            <Plus size={16} />
            <span>Ny Sida</span>
          </button>

          {/* User Auth Section */}
          {currentUser ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginLeft: '6px', borderLeft: '1px solid var(--border-color)', paddingLeft: '14px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.85rem', color: '#fff', fontWeight: 600 }}>
                <div style={{ width: 28, height: 28, borderRadius: '50%', background: 'var(--primary)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '0.75rem' }}>
                  {currentUser.username[0].toUpperCase()}
                </div>
                <span>{currentUser.username}</span>
              </div>

              <button
                className="btn btn-secondary"
                onClick={onOpenApiKeys}
                title="Mina API-nycklar"
                style={{ padding: '6px 10px', fontSize: '0.8rem' }}
              >
                <Key size={14} color="var(--accent)" />
                <span>Mina API-nycklar</span>
              </button>

              {currentUser.role === 'admin' && (
                <button
                  className="btn btn-secondary"
                  onClick={onOpenAdminUsers}
                  title="Användarhantering (Admin)"
                  style={{ padding: '6px 10px', fontSize: '0.8rem' }}
                >
                  <Users size={14} color="var(--primary)" />
                  <span>Användare</span>
                </button>
              )}

              <button
                className="btn btn-danger"
                onClick={onLogout}
                title="Logga ut"
                style={{ padding: '6px 10px', fontSize: '0.8rem' }}
              >
                <LogOut size={14} />
              </button>
            </div>
          ) : (
            <button
              className="btn btn-secondary"
              onClick={onOpenLogin}
              style={{ marginLeft: '6px', padding: '6px 14px', fontSize: '0.85rem' }}
            >
              <LogIn size={15} />
              <span>Logga in</span>
            </button>
          )}
        </div>
      </div>
    </header>
  );
};
