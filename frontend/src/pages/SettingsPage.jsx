import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Header from '../components/Header';
import { getSettings, updateSettings } from '../api/settings';
import './SettingsPage.css';

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
      const { data } = await updateSettings({ bot_token: botToken });
      setBotLink(data.bot_link || '');
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
              Создайте бота через <a href="https://t.me/BotFather" target="_blank" rel="noreferrer">@BotFather</a> в Telegram и вставьте токен ниже. Ссылка на бота определится автоматически.
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

            {botLink && (
              <div className="settings-bot-link">
                Ссылка на бота: <a href={botLink} target="_blank" rel="noreferrer">{botLink}</a>
              </div>
            )}
          </div>

          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {saving ? 'Проверка...' : 'Сохранить'}
            </button>
            {message && <span className={message.startsWith('Ошибка') ? 'text-error' : 'text-success'}>{message}</span>}
          </div>
        </form>
      </div>
    </>
  );
}
