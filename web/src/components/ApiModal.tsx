import React, { useState } from 'react';
import { X, Copy, Check, Terminal, Key } from 'lucide-react';
import type { Page } from '../types';

interface ApiModalProps {
  page?: Page | null;
  onClose: () => void;
}

export const ApiModal: React.FC<ApiModalProps> = ({ page, onClose }) => {
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);

  const apiHost = window.location.hostname === 'localhost' ? 'http://localhost:8080' : window.location.origin;
  const currentSlug = page ? page.slug : 'valkommen-till-wikin';

  const snippets = [
    {
      title: 'Hämta alla publika wikisidor (Öppen åtkomst)',
      method: 'GET',
      endpoint: '/api/v1/pages',
      cmd: `curl ${apiHost}/api/v1/pages`,
    },
    {
      title: `Hämta specifik sida (${currentSlug})`,
      method: 'GET',
      endpoint: `/api/v1/pages/${currentSlug}`,
      cmd: `curl -H "X-API-Key: ptc_key_din_api_nyckel" ${apiHost}/api/v1/pages/${currentSlug}`,
    },
    {
      title: 'Skapa ny wikisida via API',
      method: 'POST',
      endpoint: '/api/v1/pages',
      cmd: `curl -X POST ${apiHost}/api/v1/pages \\\n  -H "Content-Type: application/json" \\\n  -H "X-API-Key: ptc_key_din_api_nyckel" \\\n  -d '{\n    "title": "Ny Sida Från API",\n    "content": "# Rubrik\\nDetta skapades automatiskt via REST API:t.",\n    "is_public": false,\n    "tags": ["api", "automation"]\n  }'`,
    },
    {
      title: `Hämta ändringshistorik för sida`,
      method: 'GET',
      endpoint: `/api/v1/pages/${currentSlug}/revisions`,
      cmd: `curl -H "X-API-Key: ptc_key_din_api_nyckel" ${apiHost}/api/v1/pages/${currentSlug}/revisions`,
    },
    {
      title: 'Ladda upp fil / bild till sida',
      method: 'POST',
      endpoint: `/api/v1/pages/${currentSlug}/attachments`,
      cmd: `curl -X POST ${apiHost}/api/v1/pages/${currentSlug}/attachments \\\n  -H "X-API-Key: ptc_key_din_api_nyckel" \\\n  -F "file=@/sökväg/till/bild.png"`,
    },
    {
      title: 'Återställ sida till en tidigare revision',
      method: 'POST',
      endpoint: `/api/v1/pages/${currentSlug}/revert/1`,
      cmd: `curl -X POST ${apiHost}/api/v1/pages/${currentSlug}/revert/1 \\\n  -H "X-API-Key: ptc_key_din_api_nyckel"`,
    },
    {
      title: 'Sök i alla sidor',
      method: 'GET',
      endpoint: '/api/v1/search?q=start',
      cmd: `curl -H "X-API-Key: ptc_key_din_api_nyckel" "${apiHost}/api/v1/search?q=start"`,
    },
  ];

  const handleCopy = (text: string, index: number) => {
    navigator.clipboard.writeText(text);
    setCopiedIndex(index);
    setTimeout(() => setCopiedIndex(null), 2000);
  };

  return (
    <div style={{
      position: 'fixed',
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      backgroundColor: 'rgba(0, 0, 0, 0.8)',
      backdropFilter: 'blur(8px)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 1000,
      padding: '20px',
    }}>
      <div className="glass-panel" style={{ width: '100%', maxWidth: '780px', maxHeight: '90vh', overflowY: 'auto', padding: '28px', border: '1px solid var(--border-glow)' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '20px', borderBottom: '1px solid var(--border-color)', paddingBottom: '14px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Terminal size={24} color="var(--primary)" />
            <div>
              <h2 style={{ fontSize: '1.4rem', fontWeight: 700, color: '#fff' }}>
                REST API Interaktiv Guide
              </h2>
              <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                Base URL: <code style={{ color: 'var(--accent)' }}>{apiHost}/api/v1</code>
              </span>
            </div>
          </div>
          <button className="btn btn-secondary" onClick={onClose} style={{ padding: '6px' }}>
            <X size={18} />
          </button>
        </div>

        <div style={{ padding: '12px 16px', background: 'rgba(99, 102, 241, 0.15)', border: '1px solid rgba(99, 102, 241, 0.3)', borderRadius: 'var(--radius-md)', marginBottom: '20px', fontSize: '0.85rem', color: '#a5b4fc', display: 'flex', alignItems: 'center', gap: '10px' }}>
          <Key size={18} />
          <span>
            För anrop som kräver behörighet, skapa en personlig API-nyckel under <strong>Mina API-nycklar</strong> i menyn och skicka den som <code style={{ background: 'rgba(0,0,0,0.3)', padding: '2px 6px', borderRadius: '4px' }}>X-API-Key: ptc_key_...</code> eller Bearer token.
          </span>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
          {snippets.map((s, idx) => (
            <div key={idx} style={{ background: 'rgba(0,0,0,0.4)', borderRadius: 'var(--radius-md)', padding: '16px', border: '1px solid var(--border-color)' }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
                <span style={{ fontSize: '0.9rem', fontWeight: 600, color: '#fff' }}>
                  {s.title}
                </span>
                <span style={{ 
                  fontSize: '0.75rem', 
                  fontWeight: 700, 
                  padding: '2px 8px', 
                  borderRadius: '4px',
                  background: s.method === 'GET' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(99, 102, 241, 0.2)',
                  color: s.method === 'GET' ? '#6ee7b7' : '#a5b4fc',
                }}>
                  {s.method} {s.endpoint}
                </span>
              </div>

              <pre style={{ position: 'relative', background: '#080c14', padding: '14px', borderRadius: 'var(--radius-sm)', overflowX: 'auto', margin: 0 }}>
                <code style={{ fontFamily: 'var(--font-mono)', fontSize: '0.85rem', color: '#38bdf8' }}>
                  {s.cmd}
                </code>
                <button
                  onClick={() => handleCopy(s.cmd, idx)}
                  style={{
                    position: 'absolute',
                    top: '10px',
                    right: '10px',
                    background: 'rgba(255,255,255,0.1)',
                    border: 'none',
                    color: '#fff',
                    padding: '6px',
                    borderRadius: '4px',
                    cursor: 'pointer',
                  }}
                  title="Kopiera cURL-kommando"
                >
                  {copiedIndex === idx ? <Check size={14} color="var(--success)" /> : <Copy size={14} />}
                </button>
              </pre>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
