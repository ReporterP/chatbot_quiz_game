import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Header from '../components/Header';
import { getSettings, updateSettings } from '../api/settings';

export default function SettingsPage() {
  const navigate = useNavigate();
  const [botToken, setBotToken] = useState('');
  const [botLink, setBotLink] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    getSettings()
      .then(({ data }) => {
        setBotToken(data.bot_token || '');
        setBotLink(data.bot_link || '');
      })
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    setMessage('');
    try {
      await updateSettings({ bot_token: botToken, bot_link: botLink });
      setMessage('Настройки сохранены');
      setTimeout(() => setMessage(''), 3000);
    } catch (err) {
      setMessage('Ошибка: ' + (err.response?.data?.error || 'Неизвестная ошибка'));
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <><Header /><div className="dashboard"><div className="loading">Загрузка...</div></div></>;

  return (
    <>
      <Header />
      <div className="dashboard">
        <div className="dashboard-header">
          <h2>Настройки</h2>
          <button className="btn btn-outline btn-sm" onClick={() => navigate('/dashboard')}>← Назад</button>
        </div>

        <form onSubmit={handleSave} className="settings-form">
          <div className="settings-section">
            <h3>Telegram-бот</h3>
            <p className="settings-hint">
              Создайте бота через <a href="https://t.me/BotFather" target="_blank" rel="noreferrer">@BotFather</a> в Telegram и вставьте токен и ссылку ниже.
            </p>

            <label className="settings-label">
              Токен бота
              <input
                type="text"
                className="settings-input"
                value={botToken}
                onChange={(e) => setBotToken(e.target.value)}
                placeholder="123456789:AAHk..."
              />
            </label>

            <label className="settings-label">
              Ссылка на бота
              <input
                type="text"
                className="settings-input"
                value={botLink}
                onChange={(e) => setBotLink(e.target.value)}
                placeholder="https://t.me/my_quiz_bot"
              />
            </label>
          </div>

          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {saving ? 'Сохранение...' : 'Сохранить'}
            </button>
            {message && <span className={message.startsWith('Ошибка') ? 'text-error' : 'text-success'}>{message}</span>}
          </div>
        </form>
      </div>
    </>
  );
}
