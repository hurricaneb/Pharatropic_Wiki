import React from 'react';
import type { Page, Tag } from '../types';
import { FileText, Tag as TagIcon, Eye, Lock, CornerDownRight, ChevronRight, ChevronDown } from 'lucide-react';

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
  // Pages whose parent isn't in the current (possibly filtered) list are
  // rendered as roots too, so search/tag filtering never hides a match.
  const isRoot = (p: Page) => !p.parent_id || !pages.some((pp) => pp.id === p.parent_id);
  const rootPages = pages.filter(isRoot);
  const childrenOf = (parentId: number) => pages.filter((p) => p.parent_id === parentId);

  // A parent's subpages are shown only while that parent (or one of its
  // subpages) is the open page — keeps the list short when a page has many
  // subpages, instead of always spelling them all out.
  const isExpanded = (p: Page, kids: Page[]) =>
    p.slug === activeSlug || kids.some((c) => c.slug === activeSlug);

  const renderPageButton = (p: Page, isChild: boolean, hasChildren: boolean, expanded: boolean) => {
    const isActive = p.slug === activeSlug;
    return (
      <button
        onClick={() => onSelectPage(p.slug)}
        style={{
          width: '100%',
          textAlign: 'left',
          padding: isChild ? '8px 14px 8px 30px' : '10px 14px',
          borderRadius: 'var(--radius-md)',
          border: '1px solid',
          borderColor: isActive ? 'var(--primary)' : 'transparent',
          background: isActive ? 'var(--primary-light)' : 'transparent',
          color: isActive ? '#fff' : 'var(--text-main)',
          cursor: 'pointer',
          fontSize: isChild ? '0.83rem' : '0.9rem',
          fontWeight: isActive ? 600 : 400,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          transition: 'all 0.15s ease',
        }}
        className="sidebar-item"
      >
        <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'flex', alignItems: 'center', gap: '6px' }}>
          {!isChild && hasChildren ? (
            expanded ? (
              <ChevronDown size={14} color="var(--text-subtle)" style={{ flexShrink: 0 }} />
            ) : (
              <ChevronRight size={14} color="var(--text-subtle)" style={{ flexShrink: 0 }} />
            )
          ) : (
            isChild && <CornerDownRight size={12} color="var(--text-subtle)" style={{ flexShrink: 0 }} />
          )}
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
    );
  };

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
          <ul style={{ listStyle: 'none', display: 'flex', flexDirection: 'column', gap: '4px' }}>
            {rootPages.map((p) => {
              const kids = childrenOf(p.id);
              const hasKids = kids.length > 0;
              const expanded = hasKids && isExpanded(p, kids);
              return (
                <React.Fragment key={p.id}>
                  <li>{renderPageButton(p, false, hasKids, expanded)}</li>
                  {expanded && kids.map((child) => (
                    <li key={child.id}>{renderPageButton(child, true, false, false)}</li>
                  ))}
                </React.Fragment>
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
