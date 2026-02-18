import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Header from '../components/Header';
import { listRoomHistory, closeRoom } from '../api/rooms';
import { fetchLeaderboard } from '../api/sessions';
import './SessionHistoryPage.css';

export default function SessionHistoryPage() {
  const navigate = useNavigate();
  const [rooms, setRooms] = useState([]);
  const [loading, setLoading] = useState(true);
  const [expandedSession, setExpandedSession] = useState(null);
  const [leaderboards, setLeaderboards] = useState({});
  const [page, setPage] = useState(0);
  const perPage = 6;

  const load = async () => {
    try {
      const { data } = await listRoomHistory();
      setRooms(data || []);
    } catch { /* ignore */ }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const handleCloseRoom = async (e, roomId) => {
    e.stopPropagation();
    if (!confirm('Закрыть комнату?')) return;
    try {
      await closeRoom(roomId);
      load();
    } catch (err) {
      alert(err.response?.data?.error || 'Ошибка');
    }
  };

  const toggleLeaderboard = async (e, sessionId) => {
    e.stopPropagation();
    if (expandedSession === sessionId) {
      setExpandedSession(null);
      return;
    }
    if (!leaderboards[sessionId]) {
      try {
        const { data } = await fetchLeaderboard(sessionId);
        setLeaderboards((prev) => ({ ...prev, [sessionId]: data }));
      } catch { /* ignore */ }
    }
    setExpandedSession(sessionId);
  };

  const statusLabel = (s) => {
    const map = { active: 'Активна', closed: 'Закрыта', waiting: 'Ожидание', question: 'Вопрос', revealed: 'Ответ', finished: 'Завершён' };
    return map[s] || s;
  };

  return (
    <>
      <Header />
      <div className="dashboard">
        <div className="dashboard-header">
          <h2>История комнат</h2>
          <button className="btn btn-outline btn-sm" onClick={() => navigate('/dashboard')}>← К квизам</button>
        </div>

        {loading ? (
          <div className="loading">Загрузка...</div>
        ) : rooms.length === 0 ? (
          <div className="empty-state">
            <h3>Нет комнат</h3>
            <p>Создайте комнату, чтобы здесь появилась история</p>
          </div>
        ) : (() => {
          const totalPages = Math.ceil(rooms.length / perPage);
          const paged = rooms.slice(page * perPage, (page + 1) * perPage);
          return (
            <>
              <div className="history-list">
                {paged.map((r) => (
                  <div key={r.id} className="history-card-full">
                    <div className="history-card-row">
                      <div className="history-card-left">
                        <div className="history-title">
                          Комната {r.code}
                          <span className="history-mode-badge">{r.mode === 'web' ? 'Веб' : 'Бот'}</span>
                        </div>
                        <div className="history-meta">
                          <span className={`status-badge status-${r.status}`}>{statusLabel(r.status)}</span>
                          <span>{r.member_count} участн.</span>
                          <span>{new Date(r.created_at).toLocaleString('ru')}</span>
                        </div>
                      </div>
                      <div className="history-card-actions">
                        {r.status === 'active' && (
                          <>
                            <button className="btn btn-primary btn-sm" onClick={() => navigate(`/room/${r.id}`)}>
                              Открыть
                            </button>
                            <button className="btn btn-danger btn-sm" onClick={(e) => handleCloseRoom(e, r.id)}>
                              Закрыть
                            </button>
                          </>
                        )}
                      </div>
                    </div>

                    {r.sessions && r.sessions.length > 0 && (
                      <div className="room-sessions-list">
                        {r.sessions.map((sess) => (
                          <div key={sess.id} className="room-session-item">
                            <div className="room-session-row">
                              <span className="room-session-quiz">{sess.quiz_title}</span>
                              <span className={`status-badge status-${sess.status}`}>{statusLabel(sess.status)}</span>
                              <span className="room-session-count">{sess.participant_count} уч.</span>
                              <span className="room-session-date">{new Date(sess.created_at).toLocaleString('ru')}</span>
                              {sess.status === 'finished' && (
                                <button className="btn btn-outline btn-sm" onClick={(e) => toggleLeaderboard(e, sess.id)}>
                                  {expandedSession === sess.id ? 'Скрыть' : 'Результаты'}
                                </button>
                              )}
                            </div>

                            {expandedSession === sess.id && leaderboards[sess.id] && (
                              <div className="history-leaderboard">
                                {leaderboards[sess.id].length === 0 ? (
                                  <p className="no-data">Нет участников</p>
                                ) : (
                                  <table className="lb-table">
                                    <thead>
                                      <tr><th>#</th><th>Участник</th><th>Очки</th></tr>
                                    </thead>
                                    <tbody>
                                      {leaderboards[sess.id].map((entry) => (
                                        <tr key={entry.position} className={entry.position <= 3 ? `top-${entry.position}` : ''}>
                                          <td>{entry.position <= 3 ? ['🥇','🥈','🥉'][entry.position - 1] : entry.position}</td>
                                          <td>{entry.nickname}</td>
                                          <td>{entry.total_score}</td>
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
