import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import type { Page } from '../types';
import { Edit3, History, Trash2, Tag as TagIcon, Eye, Calendar, Code } from 'lucide-react';

interface PageViewProps {
  page: Page;
  onEdit: () => void;
  onViewHistory: () => void;
  onDelete: () => void;
  onOpenApiModal: () => void;
}

export const PageView: React.FC<PageViewProps> = ({
  page,
  onEdit,
  onViewHistory,
  onDelete,
  onOpenApiModal,
}) => {
  const formattedDate = new Date(page.updated_at).toLocaleDateString('sv-SE', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });

  return (
    <article className="glass-panel" style={{ padding: '36px', display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Header & Meta */}
      <div style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', flexWrap: 'wrap', gap: '16px' }}>
          <div>
            <h1 style={{ fontSize: '2.4rem', fontWeight: 800, color: '#fff', marginBottom: '8px', letterSpacing: '-0.02em' }}>
              {page.title}
            </h1>

            {page.summary && (
              <p style={{ fontSize: '1.05rem', color: 'var(--text-muted)', marginBottom: '14px' }}>
                {page.summary}
              </p>
            )}

            <div style={{ display: 'flex', alignItems: 'center', gap: '18px', fontSize: '0.85rem', color: 'var(--text-subtle)', flexWrap: 'wrap' }}>
              <span style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Calendar size={15} color="var(--primary)" />
                Uppdaterad {formattedDate}
              </span>
              <span style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Eye size={15} color="var(--accent)" />
                {page.views} visningar
              </span>
              {page.revisions && page.revisions.length > 0 && (
                <span style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer', color: 'var(--primary)' }} onClick={onViewHistory}>
                  <History size={15} />
                  {page.revisions.length} revisioner
                </span>
              )}
            </div>
          </div>

          {/* Action Bar */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <button className="btn btn-secondary" onClick={onEdit} title="Redigera denna wiki-sida">
              <Edit3 size={16} />
              <span>Redigera</span>
            </button>
            <button className="btn btn-secondary" onClick={onViewHistory} title="Visa ändringshistorik">
              <History size={16} />
              <span>Historik</span>
            </button>
            <button className="btn btn-secondary" onClick={onOpenApiModal} title="Se API JSON för denna sida">
              <Code size={16} />
              <span>API JSON</span>
            </button>
            <button className="btn btn-danger" onClick={onDelete} title="Radera sida">
              <Trash2 size={16} />
            </button>
          </div>
        </div>

        {/* Tags */}
        {page.tags && page.tags.length > 0 && (
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginTop: '16px' }}>
            <TagIcon size={14} color="var(--text-subtle)" />
            {page.tags.map((t) => (
              <span key={t.id} className="tag-badge">
                #{t.name}
              </span>
            ))}
          </div>
        )}
      </div>

      {/* Rendered Markdown Body */}
      <div className="markdown-body">
        <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]}>
          {page.content}
        </ReactMarkdown>
      </div>
    </article>
  );
};
