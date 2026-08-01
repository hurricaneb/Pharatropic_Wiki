import React, { useState, useRef } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import type { Page, CreatePageInput, UpdatePageInput } from '../types';
import { Save, X, Eye, Edit3, MessageSquare, Paperclip } from 'lucide-react';
import { wikiAPI } from '../api';

interface PageEditorProps {
  initialPage?: Page | null;
  onSave: (data: CreatePageInput | UpdatePageInput) => Promise<void>;
  onCancel: () => void;
}

export const PageEditor: React.FC<PageEditorProps> = ({
  initialPage,
  onSave,
  onCancel,
}) => {
  const [title, setTitle] = useState(initialPage?.title || '');
  const [summary, setSummary] = useState(initialPage?.summary || '');
  const [content, setContent] = useState(initialPage?.content || '');
  const [tagsInput, setTagsInput] = useState(
    initialPage?.tags ? initialPage.tags.map((t) => t.name).join(', ') : ''
  );
  const [comment, setComment] = useState('');
  const [activeTab, setActiveTab] = useState<'editor' | 'split' | 'preview'>('split');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');
  
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;

    if (!initialPage) {
      alert('Spara sidan först innan du laddar upp filbilagor till den.');
      return;
    }

    const file = files[0];
    setIsUploading(true);
    setErrorMsg('');

    try {
      const result = await wikiAPI.uploadAttachment(initialPage.slug, file);
      // Append markdown snippet to content
      const snippet = `\n\n${result.markdown}\n`;
      setContent((prev) => prev + snippet);
    } catch (err: any) {
      setErrorMsg(err.message || 'Kunde inte ladda upp filen.');
    } finally {
      setIsUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || !content.trim()) {
      setErrorMsg('Titel och innehåll kan inte vara tomma.');
      return;
    }

    setIsSubmitting(true);
    setErrorMsg('');

    const tags = tagsInput
      .split(',')
      .map((t) => t.trim())
      .filter((t) => t.length > 0);

    try {
      await onSave({
        title: title.trim(),
        summary: summary.trim(),
        content,
        tags,
        comment: comment.trim() || (initialPage ? 'Sida uppdaterad' : 'Ny sida skapad'),
      });
    } catch (err: any) {
      setErrorMsg(err.message || 'Misslyckades att spara sidan.');
      setIsSubmitting(false);
    }
  };

  return (
    <div className="glass-panel" style={{ padding: '32px' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '24px' }}>
        <h2 style={{ fontSize: '1.6rem', fontWeight: 700, color: '#fff' }}>
          {initialPage ? `Redigera: ${initialPage.title}` : 'Skapa Ny Wiki-sida'}
        </h2>

        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', background: 'rgba(0,0,0,0.3)', padding: '4px', borderRadius: 'var(--radius-md)' }}>
          <button
            type="button"
            className={`btn ${activeTab === 'editor' ? 'btn-primary' : 'btn-secondary'}`}
            style={{ padding: '6px 12px', fontSize: '0.8rem' }}
            onClick={() => setActiveTab('editor')}
          >
            <Edit3 size={14} /> Redigerare
          </button>
          <button
            type="button"
            className={`btn ${activeTab === 'split' ? 'btn-primary' : 'btn-secondary'}`}
            style={{ padding: '6px 12px', fontSize: '0.8rem' }}
            onClick={() => setActiveTab('split')}
          >
            Delad Vy
          </button>
          <button
            type="button"
            className={`btn ${activeTab === 'preview' ? 'btn-primary' : 'btn-secondary'}`}
            style={{ padding: '6px 12px', fontSize: '0.8rem' }}
            onClick={() => setActiveTab('preview')}
          >
            <Eye size={14} /> Förhandsgranska
          </button>
        </div>
      </div>

      {errorMsg && (
        <div style={{ padding: '12px 16px', background: 'rgba(239, 68, 68, 0.15)', border: '1px solid rgba(239, 68, 68, 0.3)', borderRadius: 'var(--radius-md)', color: '#fca5a5', marginBottom: '20px', fontSize: '0.9rem' }}>
          {errorMsg}
        </div>
      )}

      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
        {/* Title */}
        <div>
          <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '6px' }}>
            Sidsida-Titel
          </label>
          <input
            type="text"
            className="input-field"
            placeholder="T.ex. Projektarkitektur eller API-guide"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
          />
        </div>

        {/* Summary & Tags */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
          <div>
            <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '6px' }}>
              Kort Sammanfattning
            </label>
            <input
              type="text"
              className="input-field"
              placeholder="Kort beskrivning av innehållet..."
              value={summary}
              onChange={(e) => setSummary(e.target.value)}
            />
          </div>

          <div>
            <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '6px' }}>
              Taggar (Kommaseparerade)
            </label>
            <input
              type="text"
              className="input-field"
              placeholder="T.ex. api, guide, go, react"
              value={tagsInput}
              onChange={(e) => setTagsInput(e.target.value)}
            />
          </div>
        </div>

        {/* Markdown Content Area + File Upload Bar */}
        <div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '6px' }}>
            <label style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-muted)' }}>
              Innehåll (Markdown-format)
            </label>

            {initialPage && (
              <div>
                <input
                  type="file"
                  ref={fileInputRef}
                  onChange={handleFileUpload}
                  style={{ display: 'none' }}
                />
                <button
                  type="button"
                  className="btn btn-secondary"
                  style={{ padding: '4px 10px', fontSize: '0.8rem' }}
                  onClick={() => fileInputRef.current?.click()}
                  disabled={isUploading}
                >
                  <Paperclip size={14} />
                  <span>{isUploading ? 'Laddar upp...' : 'Bifoga fil / bild'}</span>
                </button>
              </div>
            )}
          </div>

          <div style={{ 
            display: 'grid', 
            gridTemplateColumns: activeTab === 'split' ? '1fr 1fr' : '1fr', 
            gap: '16px', 
            minHeight: '400px' 
          }}>
            {(activeTab === 'editor' || activeTab === 'split') && (
              <textarea
                className="textarea-field"
                placeholder="# Skriv din markdown här..."
                value={content}
                onChange={(e) => setContent(e.target.value)}
                required
              />
            )}

            {(activeTab === 'preview' || activeTab === 'split') && (
              <div className="glass-panel markdown-body" style={{ padding: '20px', overflowY: 'auto', maxHeight: '500px', background: 'rgba(0,0,0,0.4)' }}>
                <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]}>
                  {content || '*Ingen förhandsgranskning än...*'}
                </ReactMarkdown>
              </div>
            )}
          </div>
        </div>

        {/* Change Comment */}
        <div>
          <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '6px' }}>
            <MessageSquare size={14} /> Ändringskommentar (sparades i revisionen)
          </label>
          <input
            type="text"
            className="input-field"
            placeholder={initialPage ? 'T.ex. Uppdaterade avsnitt om API-anrop' : 'Första versionen av sidan'}
            value={comment}
            onChange={(e) => setComment(e.target.value)}
          />
        </div>

        {/* Actions */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: '12px', marginTop: '12px' }}>
          <button type="button" className="btn btn-secondary" onClick={onCancel} disabled={isSubmitting}>
            <X size={16} /> Avbryt
          </button>
          <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
            <Save size={16} />
            <span>{isSubmitting ? 'Sparar...' : 'Spara Wiki-sida'}</span>
          </button>
        </div>
      </form>
    </div>
  );
};
