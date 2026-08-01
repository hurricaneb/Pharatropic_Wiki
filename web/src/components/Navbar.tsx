import React from 'react';
import { BookOpen, Plus, Search, Terminal } from 'lucide-react';

interface NavbarProps {
  searchQuery: string;
  onSearchChange: (q: string) => void;
  onNewPage: () => void;
  onOpenApiModal: () => void;
  onHomeClick: () => void;
  healthOk: boolean;
}

export const Navbar: React.FC<NavbarProps> = ({
  searchQuery,
  onSearchChange,
  onNewPage,
  onOpenApiModal,
  onHomeClick,
  healthOk,
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
        <div style={{ flex: 1, maxWidth: '480px', position: 'relative' }}>
          <Search size={18} color="var(--text-muted)" style={{ position: 'absolute', left: 14, top: '50%', transform: 'translateY(-50%)' }} />
          <input
            type="text"
            className="input-field"
            placeholder="Sök i alla wikisidor och titlar..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            style={{ paddingLeft: '42px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)' }}
          />
        </div>

        {/* Action Buttons */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <button className="btn btn-secondary" onClick={onOpenApiModal} title="Visa REST API documentation">
            <Terminal size={16} />
            <span>REST API</span>
          </button>
          <button className="btn btn-primary" onClick={onNewPage}>
            <Plus size={18} />
            <span>Ny Sida</span>
          </button>
        </div>
      </div>
    </header>
  );
};
