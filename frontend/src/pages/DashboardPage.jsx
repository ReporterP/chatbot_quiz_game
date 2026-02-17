import { useEffect, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import Header from '../components/Header';
import { loadQuizzes } from '../store/quizSlice';
import { createQuiz, deleteQuiz } from '../api/quizzes';
import { createSession, listSessions } from '../api/sessions';

export default function DashboardPage() {
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const { list, loading } = useSelector((s) => s.quiz);
  const [creating, setCreating] = useState(false);
  const [newTitle, setNewTitle] = useState('');
  const [sessions, setSessions] = useState([]);
  const [showHistory, setShowHistory] = useState(false);

  useEffect(() => { dispatch(loadQuizzes()); }, []);

  const loadHistory = async () => {
    try {
      const { data } = await listSessions();
      setSessions(data || []);
      setShowHistory(true);
    } catch { /* ignore */ }
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    if (!newTitle.trim()) return;
    try {
      const { data } = await createQuiz(newTitle.trim());
      setNewTitle('');
      setCreating(false);
      navigate(`/quiz/${data.id}`);
    } catch { /* ignore */ }
  };

  const handleDelete = async (id) => {
    if (!confirm('Удалить квиз?')) return;
    await deleteQuiz(id);
    dispatch(loadQuizzes());
  };

  const handleLaunch = async (quizId) => {
    try {
      const { data } = await createSession(quizId);
      const sessionId = data.session?.id || data.id;
      navigate(`/session/${sessionId}`);
    } catch (err) {
      alert(err.response?.data?.error || 'Ошибка запуска');
    }
  };

  const statusLabel = (s) => {
    const map = { waiting: 'Ожидание', question: 'Вопрос', revealed: 'Ответ', finished: 'Завершён' };
    return map[s] || s;
  };

  return (
    <>
      <Header />
      <div className="dashboard">
        <div className="dashboard-header">
          <h2>Мои квизы</h2>
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn btn-outline btn-sm" onClick={loadHistory}>История сессий</button>
            <button className="btn btn-primary btn-sm" onClick={() => setCreating(true)}>+ Создать квиз</button>
          </div>
        </div>

        {creating && (
          <form onSubmit={handleCreate} style={{ marginBottom: 24, display: 'flex', gap: 12 }}>
            <input
              className="quiz-title-input"
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              placeholder="Название квиза..."
              autoFocus
            />
            <button type="submit" className="btn btn-success btn-sm">Создать</button>
            <button type="button" className="btn btn-outline btn-sm" onClick={() => setCreating(false)}>Отмена</button>
          </form>
        )}

        {showHistory && (
          <div className="session-history">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <h3>История сессий</h3>
              <button className="btn btn-outline btn-sm" onClick={() => setShowHistory(false)}>Скрыть</button>
            </div>
            {sessions.length === 0 ? (
              <p style={{ color: '#999', fontSize: 14 }}>Нет завершённых сессий</p>
            ) : (
              <div className="history-list">
                {sessions.map((s) => (
                  <div key={s.id} className="history-card" onClick={() => navigate(`/session/${s.id}`)}>
                    <div className="history-title">{s.quiz_title}</div>
                    <div className="history-meta">
                      <span className={`status-badge status-${s.status}`}>{statusLabel(s.status)}</span>
                      <span>{s.participant_count} участн.</span>
                      <span>{new Date(s.created_at).toLocaleString('ru')}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {loading ? (
          <div className="loading">Загрузка...</div>
        ) : list.length === 0 ? (
          <div className="empty-state">
            <h3>Пока нет квизов</h3>
            <p>Создайте свой первый квиз, чтобы начать</p>
          </div>
        ) : (
          <div className="quiz-grid">
            {list.map((q) => (
              <div key={q.id} className="quiz-card">
                <h3>{q.title}</h3>
                <div className="quiz-meta">
                  {q.questions?.length || 0} вопрос(ов) &middot; {new Date(q.created_at).toLocaleDateString('ru')}
                </div>
                <div className="quiz-card-actions">
                  <button className="btn btn-outline btn-sm" onClick={() => navigate(`/quiz/${q.id}`)}>Редактировать</button>
                  <button
                    className="btn btn-success btn-sm"
                    onClick={() => handleLaunch(q.id)}
                    disabled={!q.questions?.length}
                    title={!q.questions?.length ? 'Добавьте хотя бы 1 вопрос' : ''}
                  >
                    Запустить
                  </button>
                  <button className="btn btn-danger btn-sm" onClick={() => handleDelete(q.id)}>Удалить</button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  );
}
