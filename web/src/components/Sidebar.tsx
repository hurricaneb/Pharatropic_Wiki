import React from 'react';
import type { Page, Tag } from '../types';
import { FileText, Tag as TagIcon, Eye, Lock } from 'lucide-react';

interface SidebarProps {
  pages: Page[];
  tags: Tag[];
  activeSlug: string | null;
  activeTag: string | null;
  onSelectPage: (slug: string) => void;
  onSelectTag: (tagSlug: string | null) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  pages,
  tags,
  activeSlug,
  activeTag,
  onSelectPage,
  onSelectTag,
}) => {
  return (
    <aside style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
      {/* Pages Section */}
      <div className="glass-panel" style={{ padding: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '14px' }}>
          <h3 style={{ fontSize: '0.95rem', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <FileText size={16} color="var(--primary)" />
            Alla Sidor ({pages.length})
          </h3>
        </div>

        {pages.length === 0 ? (
          <div style={{ padding: '20px 0', textAlign: 'center', color: 'var(--text-subtle)', fontSize: '0.85rem' }}>
            Inga sidor hittades.
          </div>
        ) : (
          <ul style={{ listStyle: 'none', display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {pages.map((p) => {
              const isActive = p.slug === activeSlug;
              return (
                <li key={p.id}>
                  <button
                    onClick={() => onSelectPage(p.slug)}
                    style={{
                      width: '100%',
                      textAlign: 'left',
                      padding: '10px 14px',
                      borderRadius: 'var(--radius-md)',
                      border: '1px solid',
                      borderColor: isActive ? 'var(--primary)' : 'transparent',
                      background: isActive ? 'var(--primary-light)' : 'transparent',
                      color: isActive ? '#fff' : 'var(--text-main)',
                      cursor: 'pointer',
                      fontSize: '0.9rem',
                      fontWeight: isActive ? 600 : 400,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      transition: 'all 0.15s ease',
                    }}
                    className="sidebar-item"
                  >
                    <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'flex', alignItems: 'center', gap: '6px' }}>
                      {!p.is_public && (
                        <span title="Privat sida (kräver inloggning)" style={{ display: 'inline-flex', alignItems: 'center' }}>
                          <Lock size={12} color="var(--accent)" />
                        </span>
                      )}
                      {p.title}
                    </span>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', fontSize: '0.75rem', color: 'var(--text-subtle)' }}>
                      <Eye size={12} />
                      {p.views}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      {/* Tags Section */}
      <div className="glass-panel" style={{ padding: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '14px' }}>
          <h3 style={{ fontSize: '0.95rem', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <TagIcon size={16} color="var(--accent)" />
            Kategorier & Taggar
          </h3>
          {activeTag && (
            <button 
              onClick={() => onSelectTag(null)} 
              style={{ background: 'none', border: 'none', color: 'var(--accent)', cursor: 'pointer', fontSize: '0.75rem' }}
            >
              Rensa
            </button>
          )}
        </div>

        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
          {tags.map((t) => (
            <span
              key={t.id}
              className={`tag-badge ${activeTag === t.slug ? 'active' : ''}`}
              onClick={() => onSelectTag(activeTag === t.slug ? null : t.slug)}
            >
              #{t.name}
            </span>
          ))}
          {tags.length === 0 && (
            <span style={{ fontSize: '0.85rem', color: 'var(--text-subtle)' }}>Inga taggar än.</span>
          )}
        </div>
      </div>
    </aside>
  );
};
