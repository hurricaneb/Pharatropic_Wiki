import React, { useState, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import type { Page, Attachment, User } from '../types';
import { Edit3, History, Trash2, Tag as TagIcon, Eye, Calendar, Paperclip, Download, Copy, Check, File, Image, Link2, Lock, Globe } from 'lucide-react';
import { wikiAPI } from '../api';

import { transformWikiLinks, slugify } from '../utils/wikiLink';

interface PageViewProps {
  page: Page;
  pages?: Page[];
  currentUser?: User | null;
  onEdit: () => void;
  onViewHistory: () => void;
  onDelete: () => void;
  onSelectPage?: (slug: string) => void;
  onCreateMissingPage?: (title: string) => void;
  onRefreshPage?: () => void;
}

export const PageView: React.FC<PageViewProps> = ({
  page,
  pages = [],
  currentUser,
  onEdit,
  onViewHistory,
  onDelete,
  onSelectPage,
  onCreateMissingPage,
  onRefreshPage,
}) => {
  const [copiedId, setCopiedId] = useState<number | null>(null);
  const [backlinks, setBacklinks] = useState<Page[]>([]);

  useEffect(() => {
    if (page?.slug) {
      wikiAPI.getBacklinks(page.slug)
        .then(setBacklinks)
        .catch(() => setBacklinks([]));
    }
  }, [page?.slug]);

  const formattedDate = new Date(page.updated_at).toLocaleDateString('sv-SE', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });

  const customUrlTransform = (url: string) => {
    const apiHost = window.location.hostname === 'localhost' && window.location.port !== '8080'
      ? 'http://localhost:8080'
      : '';

    let finalUrl = url;
    if (url.startsWith('/uploads/')) {
      finalUrl = `${apiHost}${url}`;
    }

    return finalUrl.replace(/ /g, '%20');
  };

  const handleCopyMarkdown = (att: Attachment) => {
    const isImg = att.mime_type.startsWith('image/');
    const encodedPath = att.file_path.replace(/ /g, '%20');
    const snippet = isImg ? `![${att.original_name}](${encodedPath})` : `[${att.original_name}](${encodedPath})`;
    navigator.clipboard.writeText(snippet);
    setCopiedId(att.id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const handleDeleteAttachment = async (id: number) => {
    if (!window.confirm('Är du säker på att du vill ta bort denna bilaga?')) return;
    try {
      await wikiAPI.deleteAttachment(id);
      if (onRefreshPage) onRefreshPage();
    } catch (err: any) {
      alert(err.message || 'Misslyckades att radera bilagan.');
    }
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const apiHost = window.location.hostname === 'localhost' ? 'http://localhost:8080' : window.location.origin;

  return (
    <article className="glass-panel" style={{ padding: '36px', display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Header & Meta */}
      <div style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', flexWrap: 'wrap', gap: '16px' }}>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap', marginBottom: '8px' }}>
              <h1 style={{ fontSize: '2.4rem', fontWeight: 800, color: '#fff', letterSpacing: '-0.02em', margin: 0 }}>
                {page.title}
              </h1>

              <span style={{
                fontSize: '0.75rem',
                fontWeight: 700,
                padding: '4px 10px',
                borderRadius: '6px',
                display: 'inline-flex',
                alignItems: 'center',
                gap: '5px',
                background: page.is_public ? 'rgba(16, 185, 129, 0.2)' : 'rgba(99, 102, 241, 0.25)',
                color: page.is_public ? '#6ee7b7' : '#a5b4fc',
                border: `1px solid ${page.is_public ? 'rgba(16, 185, 129, 0.4)' : 'rgba(99, 102, 241, 0.4)'}`,
              }}>
                {page.is_public ? <Globe size={13} /> : <Lock size={13} />}
                {page.is_public ? 'Publik' : 'Privat'}
              </span>
            </div>

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
            {currentUser && (
              <button className="btn btn-secondary" onClick={onEdit} title="Redigera denna wiki-sida">
                <Edit3 size={16} />
                <span>Redigera</span>
              </button>
            )}
            <button className="btn btn-secondary" onClick={onViewHistory} title="Visa ändringshistorik">
              <History size={16} />
              <span>Historik</span>
            </button>
            {currentUser && (
              <button className="btn btn-danger" onClick={onDelete} title="Radera sida">
                <Trash2 size={16} />
              </button>
            )}
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
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          rehypePlugins={[rehypeHighlight]}
          urlTransform={customUrlTransform}
          components={{
            a: ({ href, children, ...props }) => {
              if (href && href.startsWith('#wikilink:')) {
                const targetTitle = decodeURIComponent(href.replace('#wikilink:', ''));
                const targetSlug = slugify(targetTitle);
                const existingPage = pages.find((p) => p.slug === targetSlug);

                if (existingPage) {
                  return (
                    <a
                      href={`/wiki/${targetSlug}`}
                      className="wikilink wikilink-exists"
                      title={`Gå till "${existingPage.title}"`}
                      onClick={(e) => {
                        e.preventDefault();
                        if (onSelectPage) onSelectPage(targetSlug);
                      }}
                    >
                      {children}
                    </a>
                  );
                } else {
                  return (
                    <a
                      href="#"
                      className="wikilink wikilink-missing"
                      title={`Sidan "${targetTitle}" finns inte ännu. Klicka för att skapa!`}
                      onClick={(e) => {
                        e.preventDefault();
                        if (onCreateMissingPage) onCreateMissingPage(targetTitle);
                      }}
                    >
                      {children} ➕
                    </a>
                  );
                }
              }
              return <a href={href} target="_blank" rel="noreferrer" {...props}>{children}</a>;
            },
          }}
        >
          {transformWikiLinks(page.content)}
        </ReactMarkdown>
      </div>

      {/* Attachments Section */}
      {page.attachments && page.attachments.length > 0 && (
        <div style={{ borderTop: '1px solid var(--border-color)', marginTop: '20px', paddingTop: '20px' }}>
          <h3 style={{ fontSize: '1.1rem', fontWeight: 700, color: '#fff', marginBottom: '14px', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Paperclip size={18} color="var(--primary)" />
            Bifogade Filer ({page.attachments.length})
          </h3>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: '14px' }}>
            {page.attachments.map((att) => {
              const isImg = att.mime_type.startsWith('image/');
              const fileUrl = `${apiHost}${att.file_path}`;

              return (
                <div key={att.id} className="glass-panel" style={{ padding: '14px', background: 'rgba(0,0,0,0.3)', display: 'flex', flexDirection: 'column', justifyContent: 'space-between', gap: '10px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                    {isImg ? <Image size={20} color="var(--accent)" /> : <File size={20} color="var(--primary)" />}
                    <div style={{ overflow: 'hidden' }}>
                      <div style={{ fontSize: '0.9rem', fontWeight: 600, color: '#fff', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }} title={att.original_name}>
                        {att.original_name}
                      </div>
                      <div style={{ fontSize: '0.75rem', color: 'var(--text-subtle)' }}>
                        {formatSize(att.file_size)}
                      </div>
                    </div>
                  </div>

                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '8px', borderTop: '1px solid rgba(255,255,255,0.06)', paddingTop: '8px' }}>
                    <a
                      href={fileUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="btn btn-secondary"
                      style={{ padding: '4px 8px', fontSize: '0.75rem' }}
                      title="Ladda ner fil"
                    >
                      <Download size={14} /> Öppna
                    </a>

                    <button
                      className="btn btn-secondary"
                      style={{ padding: '4px 8px', fontSize: '0.75rem' }}
                      onClick={() => handleCopyMarkdown(att)}
                      title="Kopiera Markdown-länk"
                    >
                      {copiedId === att.id ? <Check size={14} color="var(--success)" /> : <Copy size={14} />}
                      <span>Markdown</span>
                    </button>

                    <button
                      className="btn btn-danger"
                      style={{ padding: '4px 8px', fontSize: '0.75rem' }}
                      onClick={() => handleDeleteAttachment(att.id)}
                      title="Ta bort bilaga"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Backlinks Section */}
      {backlinks && backlinks.length > 0 && (
        <div style={{ borderTop: '1px solid var(--border-color)', marginTop: '20px', paddingTop: '20px' }}>
          <h3 style={{ fontSize: '1.1rem', fontWeight: 700, color: '#fff', marginBottom: '14px', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Link2 size={18} color="var(--accent)" />
            Sidor som länkar hit (Backlinks) ({backlinks.length})
          </h3>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: '12px' }}>
            {backlinks.map((bPage) => (
              <div
                key={bPage.id}
                className="glass-panel"
                style={{
                  padding: '14px',
                  cursor: 'pointer',
                  transition: 'all 0.2s ease',
                  background: 'rgba(0,0,0,0.3)',
                  border: '1px solid var(--border-color)',
                }}
                onClick={() => onSelectPage && onSelectPage(bPage.slug)}
              >
                <div style={{ fontSize: '0.95rem', fontWeight: 700, color: '#fff', marginBottom: '4px' }}>
                  {bPage.title}
                </div>
                {bPage.summary && (
                  <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {bPage.summary}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </article>
  );
};
