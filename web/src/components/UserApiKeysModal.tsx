import React, { useState, useEffect } from 'react';
import { X, Key, Plus, Trash2, Copy, Check, Clock, Calendar, ShieldCheck } from 'lucide-react';
import { wikiAPI } from '../api';
import type { UserApiKey } from '../types';

interface UserApiKeysModalProps {
  onClose: () => void;
}

export const UserApiKeysModal: React.FC<UserApiKeysModalProps> = ({ onClose }) => {
  const [keys, setKeys] = useState<UserApiKey[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  
  // Create Key Form State
  const [keyName, setKeyName] = useState('');
  const [expiryOption, setExpiryOption] = useState('30d');
  const [isCreating, setIsCreating] = useState(false);
  const [newSecretKey, setNewSecretKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const loadKeys = async () => {
    try {
      const fetchedKeys = await wikiAPI.listUserApiKeys();
      setKeys(fetchedKeys);
    } catch (err: any) {
      setError(err.message || 'Kunde inte hämta API-nycklar');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadKeys();
  }, []);

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!keyName.trim()) return;

    setIsCreating(true);
    setError('');

    try {
      const res = await wikiAPI.createUserApiKey(keyName, expiryOption);
      setNewSecretKey(res.key);
      setKeyName('');
      await loadKeys();
    } catch (err: any) {
      setError(err.message || 'Kunde inte skapa API-nyckel');
    } finally {
      setIsCreating(false);
    }
  };

  const handleRevokeKey = async (id: number) => {
    if (!window.confirm('Är du säker på att du vill återkalla denna API-nyckel? Alla system som använder nyckeln kommer att förlora åtkomst.')) {
      return;
    }

    try {
      await wikiAPI.revokeUserApiKey(id);
      await loadKeys();
    } catch (err: any) {
      alert(err.message || 'Misslyckades att återkalla nyckeln');
    }
  };

  const handleCopySecret = () => {
    if (!newSecretKey) return;
    navigator.clipboard.writeText(newSecretKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div style={{
      position: 'fixed',
      inset: 0,
      background: 'rgba(0,0,0,0.75)',
      backdropFilter: 'blur(8px)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 1000,
      padding: '20px',
    }}>
      <div className="glass-panel modal-glass-container" style={{ width: '100%', maxWidth: '640px', padding: '28px', position: 'relative', maxHeight: '90vh', overflowY: 'auto' }}>
        <button
          onClick={onClose}
          style={{ position: 'absolute', right: '18px', top: '18px', background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer' }}
        >
          <X size={20} />
        </button>

        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '20px' }}>
          <div style={{ padding: '8px', borderRadius: '10px', background: 'rgba(99, 102, 241, 0.2)', color: 'var(--primary)' }}>
            <Key size={22} />
          </div>
          <div>
            <h2 style={{ fontSize: '1.4rem', fontWeight: 800, color: '#fff' }}>Mina API-nycklar</h2>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>Hantera dina personliga API-nycklar för programmatisk åtkomst</p>
          </div>
        </div>

        {error && (
          <div style={{ padding: '10px 14px', background: 'rgba(239, 68, 68, 0.15)', border: '1px solid rgba(239, 68, 68, 0.3)', borderRadius: 'var(--radius-md)', color: '#fca5a5', marginBottom: '16px', fontSize: '0.85rem' }}>
            {error}
          </div>
        )}

        {/* 1-Time Secret Key Reveal Modal */}
        {newSecretKey && (
          <div style={{ padding: '16px', background: 'rgba(16, 185, 129, 0.15)', border: '1px solid rgba(16, 185, 129, 0.4)', borderRadius: 'var(--radius-md)', marginBottom: '20px' }}>
            <div style={{ fontWeight: 700, color: '#6ee7b7', marginBottom: '6px', display: 'flex', alignItems: 'center', gap: '6px' }}>
              <ShieldCheck size={18} /> API-nyckel Skapad! Kopiera nyckeln nu.
            </div>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '12px' }}>
              Av säkerhetsskäl visas denna nyckel <strong>endast en gång</strong>. Spara den på ett säkert ställe!
            </p>
            <div style={{ display: 'flex', gap: '8px' }}>
              <input
                type="text"
                className="input-field"
                readOnly
                value={newSecretKey}
                style={{ fontFamily: 'var(--font-mono)', fontSize: '0.85rem' }}
              />
              <button className="btn btn-primary" onClick={handleCopySecret} style={{ padding: '8px 14px', whiteSpace: 'nowrap' }}>
                {copied ? <Check size={16} /> : <Copy size={16} />}
                <span>{copied ? 'Kopierad!' : 'Kopiera'}</span>
              </button>
            </div>
            <button
              className="btn btn-secondary"
              onClick={() => setNewSecretKey(null)}
              style={{ marginTop: '10px', fontSize: '0.75rem', padding: '4px 10px' }}
            >
              Stäng hemlig visning
            </button>
          </div>
        )}

        {/* Create API Key Form */}
        <form onSubmit={handleCreateKey} style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '20px', marginBottom: '20px' }}>
          <h3 style={{ fontSize: '0.95rem', fontWeight: 700, color: '#fff', marginBottom: '12px' }}>
            Skapa Ny API-nyckel
          </h3>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 140px auto', gap: '10px' }}>
            <input
              type="text"
              className="input-field"
              placeholder="Ex: CI/CD Deployment Key"
              value={keyName}
              onChange={(e) => setKeyName(e.target.value)}
              required
            />
            <select
              className="input-field"
              value={expiryOption}
              onChange={(e) => setExpiryOption(e.target.value)}
            >
              <option value="7d">7 Dagar</option>
              <option value="30d">30 Dagar</option>
              <option value="90d">90 Dagar</option>
              <option value="1y">1 År</option>
              <option value="never">Aldrig</option>
            </select>

            <button type="submit" className="btn btn-primary" disabled={isCreating} style={{ whiteSpace: 'nowrap' }}>
              <Plus size={16} />
              <span>{isCreating ? 'Skapar...' : 'Skapa'}</span>
            </button>
          </div>
        </form>

        {/* API Keys List */}
        <div>
          <h3 style={{ fontSize: '0.95rem', fontWeight: 700, color: '#fff', marginBottom: '12px' }}>
            Aktiva API-nycklar ({keys.length})
          </h3>

          {isLoading ? (
            <div style={{ padding: '20px', textAlign: 'center', color: 'var(--text-muted)' }}>Laddar nycklar...</div>
          ) : keys.length === 0 ? (
            <div style={{ padding: '20px', textAlign: 'center', color: 'var(--text-subtle)', background: 'rgba(0,0,0,0.2)', borderRadius: 'var(--radius-md)' }}>
              Inga API-nycklar skapade än.
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              {keys.map((k) => {
                const isExpired = k.expires_at && new Date(k.expires_at) < new Date();
                const expDateStr = k.expires_at
                  ? new Date(k.expires_at).toLocaleDateString('sv-SE')
                  : 'Aldrig';

                return (
                  <div
                    key={k.id}
                    className="glass-panel"
                    style={{ padding: '14px', background: 'rgba(0,0,0,0.3)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px' }}
                  >
                    <div>
                      <div style={{ fontSize: '0.95rem', fontWeight: 700, color: '#fff', display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <span>{k.name}</span>
                        {isExpired && (
                          <span style={{ fontSize: '0.7rem', padding: '2px 6px', borderRadius: '4px', background: 'var(--danger)', color: '#fff' }}>
                            Utgången
                          </span>
                        )}
                      </div>
                      <div style={{ fontSize: '0.8rem', fontFamily: 'var(--font-mono)', color: 'var(--accent)', marginTop: '2px' }}>
                        {k.prefix}
                      </div>
                      <div style={{ fontSize: '0.75rem', color: 'var(--text-subtle)', marginTop: '4px', display: 'flex', gap: '14px' }}>
                        <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                          <Calendar size={12} /> Utgår: {expDateStr}
                        </span>
                        <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                          <Clock size={12} /> Senast använd: {k.last_used_at ? new Date(k.last_used_at).toLocaleDateString('sv-SE') : 'Aldrig'}
                        </span>
                      </div>
                    </div>

                    <button
                      className="btn btn-danger"
                      onClick={() => handleRevokeKey(k.id)}
                      style={{ padding: '6px 10px', fontSize: '0.8rem' }}
                      title="Återkalla / Ta bort nyckel"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
