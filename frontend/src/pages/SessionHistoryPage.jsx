import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Header from '../components/Header';
import { listSessions, forceFinishSession, fetchLeaderboard } from '../api/sessions';
import './SessionHistoryPage.css';

export default function SessionHistoryPage() {
  const navigate = useNavigate();
  const [sessions, setSessions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [expandedId, setExpandedId] = useState(null);
  const [leaderboards, setLeaderboards] = useState({});
  const [page, setPage] = useState(0);
  const perPage = 6;

  const load = async () => {
    try {
      const { data } = await listSessions();
      setSessions(data || []);
    } catch { /* ignore */ }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const handleForceFinish = async (e, id) => {
    e.stopPropagation();
    if (!confirm('Досрочно завершить сессию?')) return;
    try {
      await forceFinishSession(id);
      load();
    } catch (err) {
      alert(err.response?.data?.error || 'Ошибка');
    }
  };

  const toggleLeaderboard = async (e, id) => {
    e.stopPropagation();
    if (expandedId === id) {
      setExpandedId(null);
      return;
    }
    if (!leaderboards[id]) {
      try {
        const { data } = await fetchLeaderboard(id);
        setLeaderboards((prev) => ({ ...prev, [id]: data }));
      } catch { /* ignore */ }
    }
    setExpandedId(id);
  };

  const statusLabel = (s) => {
    const map = { waiting: 'Ожидание', question: 'Вопрос', revealed: 'Ответ', finished: 'Завершён' };
    return map[s] || s;
  };

  const isActive = (status) => status !== 'finished';

  return (
    <>
      <Header />
      <div className="dashboard">
        <div className="dashboard-header">
          <h2>История сессий</h2>
          <button className="btn btn-outline btn-sm" onClick={() => navigate('/dashboard')}>← К квизам</button>
        </div>

        {loading ? (
          <div className="loading">Загрузка...</div>
        ) : sessions.length === 0 ? (
          <div className="empty-state">
            <h3>Нет сессий</h3>
            <p>Запустите квиз, чтобы здесь появилась история</p>
          </div>
        ) : (() => {
          const totalPages = Math.ceil(sessions.length / perPage);
          const paged = sessions.slice(page * perPage, (page + 1) * perPage);
          return (
            <>
              <div className="history-list">
                {paged.map((s) => (
                  <div key={s.id} className="history-card-full">
                    <div className="history-card-row" onClick={() => navigate(`/session/${s.id}`)}>
                      <div className="history-card-left">
                        <div className="history-title">{s.quiz_title}</div>
                        <div className="history-meta">
                          <span className={`status-badge status-${s.status}`}>{statusLabel(s.status)}</span>
                          <span>{s.participant_count} участн.</span>
                          <span>{new Date(s.created_at).toLocaleString('ru')}</span>
                          <span className="history-code">Код: {s.code}</span>
                        </div>
                      </div>
                      <div className="history-card-actions">
                        {s.status === 'finished' && (
                          <button className="btn btn-outline btn-sm" onClick={(e) => toggleLeaderboard(e, s.id)}>
                            {expandedId === s.id ? 'Скрыть' : 'Результаты'}
                          </button>
                        )}
                        {isActive(s.status) && (
                          <>
                            <button className="btn btn-primary btn-sm" onClick={(e) => { e.stopPropagation(); navigate(`/session/${s.id}`); }}>
                              Открыть
                            </button>
                            <button className="btn btn-danger btn-sm" onClick={(e) => handleForceFinish(e, s.id)}>
                              Завершить
                            </button>
                          </>
                        )}
                      </div>
                    </div>

                    {expandedId === s.id && leaderboards[s.id] && (
                      <div className="history-leaderboard">
                        {leaderboards[s.id].length === 0 ? (
                          <p className="no-data">Нет участников</p>
                        ) : (
                          <table className="lb-table">
                            <thead>
                              <tr><th>#</th><th>Участник</th><th>Очки</th></tr>
                            </thead>
                            <tbody>
                              {leaderboards[s.id].map((e) => (
                                <tr key={e.position} className={e.position <= 3 ? `top-${e.position}` : ''}>
                                  <td>{e.position <= 3 ? ['🥇','🥈','🥉'][e.position - 1] : e.position}</td>
                                  <td>{e.nickname}</td>
                                  <td>{e.total_score}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>

              {totalPages > 1 && (
                <div className="pagination">
                  <button className="btn btn-outline btn-sm" disabled={page === 0} onClick={() => setPage(page - 1)}>← Назад</button>
                  <span className="pagination-info">{page + 1} / {totalPages}</span>
                  <button className="btn btn-outline btn-sm" disabled={page >= totalPages - 1} onClick={() => setPage(page + 1)}>Вперёд →</button>
                </div>
              )}
            </>
          );
        })()}
      </div>
    </>
  );
}
