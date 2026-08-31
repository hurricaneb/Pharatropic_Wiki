import React, { useState } from 'react';
import { ShieldAlert, Lock } from 'lucide-react';
import { wikiAPI } from '../api';

interface ForcePasswordChangeModalProps {
  onSuccess: () => void;
}

export const ForcePasswordChangeModal: React.FC<ForcePasswordChangeModalProps> = ({ onSuccess }) => {
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (newPassword.length < 8) {
      setError('Det nya lösenordet måste vara minst 8 tecken.');
      return;
    }
    if (newPassword !== confirmPassword) {
      setError('Lösenorden matchar inte.');
      return;
    }

    setIsSaving(true);
    try {
      await wikiAPI.changePassword(currentPassword, newPassword);
      onSuccess();
    } catch (err: any) {
      setError(err.message || 'Kunde inte byta lösenord');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div style={{
      position: 'fixed',
      inset: 0,
      background: 'rgba(0,0,0,0.85)',
      backdropFilter: 'blur(8px)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 2000,
      padding: '20px',
    }}>
      <div className="glass-panel modal-glass-container" style={{ width: '100%', maxWidth: '440px', padding: '28px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '10px' }}>
          <div style={{ padding: '8px', borderRadius: '10px', background: 'rgba(239, 68, 68, 0.15)', color: '#fca5a5' }}>
            <ShieldAlert size={22} />
          </div>
          <div>
            <h2 style={{ fontSize: '1.3rem', fontWeight: 800, color: '#fff' }}>Byt lösenord för att fortsätta</h2>
          </div>
        </div>

        <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)', marginBottom: '20px' }}>
          Det här kontot använder fortfarande sitt ursprungliga lösenord. Av säkerhetsskäl måste du sätta ett nytt lösenord innan du kan använda wikin.
        </p>

        {error && (
          <div style={{ padding: '10px 14px', background: 'rgba(239, 68, 68, 0.15)', border: '1px solid rgba(239, 68, 68, 0.3)', borderRadius: 'var(--radius-md)', color: '#fca5a5', marginBottom: '16px', fontSize: '0.85rem' }}>
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div>
            <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '6px' }}>
              Nuvarande lösenord
            </label>
            <div style={{ position: 'relative' }}>
              <Lock size={16} color="var(--text-muted)" style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)' }} />
              <input
                type="password"
                className="input-field"
                style={{ paddingLeft: '38px' }}
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                required
                autoFocus
              />
            </div>
          </div>

          <div>
            <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '6px' }}>
              Nytt lösenord (minst 8 tecken)
            </label>
            <div style={{ position: 'relative' }}>
              <Lock size={16} color="var(--text-muted)" style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)' }} />
              <input
                type="password"
                className="input-field"
                style={{ paddingLeft: '38px' }}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
                minLength={8}
              />
            </div>
          </div>

          <div>
            <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '6px' }}>
              Bekräfta nytt lösenord
            </label>
            <div style={{ position: 'relative' }}>
              <Lock size={16} color="var(--text-muted)" style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)' }} />
              <input
                type="password"
                className="input-field"
                style={{ paddingLeft: '38px' }}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                minLength={8}
              />
            </div>
          </div>

          <button
            type="submit"
            className="btn btn-primary"
            disabled={isSaving}
            style={{ width: '100%', marginTop: '8px', padding: '10px' }}
          >
            {isSaving ? 'Byter lösenord...' : 'Byt lösenord'}
          </button>
        </form>
      </div>
    </div>
  );
};
