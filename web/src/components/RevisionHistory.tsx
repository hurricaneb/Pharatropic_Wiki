import React, { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import type { Page, Revision } from '../types';
import { ArrowLeft, Clock, MessageSquare, RotateCcw } from 'lucide-react';
import { getApiHost } from '../api';

interface RevisionHistoryProps {
  page: Page;
  revisions: Revision[];
  onBack: () => void;
  onRevertRevision?: (revisionId: number) => Promise<void>;
}

export const RevisionHistory: React.FC<RevisionHistoryProps> = ({
  page,
  revisions,
  onBack,
  onRevertRevision,
}) => {
  const [selectedRevision, setSelectedRevision] = useState<Revision | null>(
    revisions.length > 0 ? revisions[0] : null
  );
  const [isReverting, setIsReverting] = useState(false);

  const handleRevert = async () => {
    if (!selectedRevision || !onRevertRevision) return;
    const selectedIdx = revisions.findIndex((r) => r.id === selectedRevision.id);
    const seqNo = selectedIdx !== -1 ? revisions.length - selectedIdx : selectedRevision.id;

    if (!window.confirm(`Är du säker på att du vill återställa wikisidan till revision #${seqNo}?`)) {
      return;
    }

    setIsReverting(true);
    try {
      await onRevertRevision(selectedRevision.id);
    } catch (err: any) {
      alert(err.message || 'Misslyckades att återställa revisionen.');
    } finally {
      setIsReverting(false);
    }
  };

  return (
    <div className="glass-panel" style={{ padding: '32px', display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--border-color)', paddingBottom: '18px' }}>
        <div>
          <button className="btn btn-secondary" onClick={onBack} style={{ marginBottom: '12px', padding: '6px 12px', fontSize: '0.85rem' }}>
            <ArrowLeft size={16} /> Tillbaka till sidan
          </button>
          <h2 style={{ fontSize: '1.8rem', fontWeight: 800, color: '#fff' }}>
            Ändringshistorik: {page.title}
          </h2>
          <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>
            Totalt {revisions.length} sparade revisioner
          </p>
        </div>
      </div>

      {/* Grid: Timeline vs Content preview */}
      <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 300px) minmax(0, 1fr)', gap: '24px' }}>
        {/* Revision Timeline List */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', minWidth: 0 }}>
          <h3 style={{ fontSize: '0.9rem', fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-muted)' }}>
            Revisioner
          </h3>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '550px', overflowY: 'auto' }}>
            {revisions.map((rev, index) => {
              const isSelected = selectedRevision?.id === rev.id;
              const dateStr = new Date(rev.created_at).toLocaleDateString('sv-SE', {
                year: 'numeric',
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit',
              });

              return (
                <div
                  key={rev.id}
                  onClick={() => setSelectedRevision(rev)}
                  style={{
                    padding: '14px',
                    borderRadius: 'var(--radius-md)',
                    border: '1px solid',
                    borderColor: isSelected ? 'var(--primary)' : 'var(--border-color)',
                    background: isSelected ? 'var(--primary-light)' : 'rgba(0,0,0,0.2)',
                    cursor: 'pointer',
                    transition: 'all 0.2s ease',
                    wordBreak: 'break-word',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '6px' }}>
                    <span style={{ fontSize: '0.85rem', fontWeight: 700, color: isSelected ? '#fff' : 'var(--text-main)' }}>
                      Revision #{revisions.length - index}
                    </span>
                    {index === 0 && (
                      <span style={{ fontSize: '0.7rem', padding: '2px 6px', borderRadius: '4px', background: 'var(--success)', color: '#fff', fontWeight: 700 }}>
                        Nuvarande
                      </span>
                    )}
                  </div>

                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.78rem', color: 'var(--text-subtle)', marginBottom: '6px' }}>
                    <Clock size={13} /> {dateStr}
                  </div>

                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.78rem', color: 'var(--text-muted)', overflowWrap: 'anywhere' }}>
                    <MessageSquare size={13} color="var(--primary)" style={{ flexShrink: 0 }} /> {rev.comment || 'Ingen kommentar'}
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Selected Revision Content Preview */}
        <div style={{ minWidth: 0 }}>
          {selectedRevision ? (
            <div className="glass-panel" style={{ padding: '24px', background: 'rgba(0,0,0,0.3)', minWidth: 0 }}>
              <div style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '14px', marginBottom: '18px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '12px' }}>
                <div>
                  <h3 style={{ fontSize: '1.3rem', fontWeight: 700, color: '#fff', marginBottom: '4px' }}>
                    {selectedRevision.title}
                  </h3>
                  <div style={{ fontSize: '0.85rem', color: 'var(--text-muted)', display: 'flex', gap: '16px' }}>
                    <span>Författare: {selectedRevision.author}</span>
                    <span>Kommentar: "{selectedRevision.comment}"</span>
                  </div>
                </div>

                {onRevertRevision && (
                  <button
                    className="btn btn-primary"
                    onClick={handleRevert}
                    disabled={isReverting}
                    style={{ padding: '6px 14px', fontSize: '0.85rem' }}
                    title="Skapa ny revision baserad på denna tidigare version"
                  >
                    <RotateCcw size={15} />
                    <span>{isReverting ? 'Återställer...' : 'Återställ till denna version'}</span>
                  </button>
                )}
              </div>

              <div className="markdown-body">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  rehypePlugins={[rehypeHighlight]}
                  urlTransform={(url) => {
                    const apiHost = getApiHost();
                    let finalUrl = url.startsWith('/uploads/') ? `${apiHost}${url}` : url;
                    return finalUrl.replace(/ /g, '%20');
                  }}
                >
                  {selectedRevision.content}
                </ReactMarkdown>
              </div>
            </div>
          ) : (
            <div style={{ padding: '40px', textAlign: 'center', color: 'var(--text-subtle)' }}>
              Välj en revision till vänster för att granska innehållet.
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
