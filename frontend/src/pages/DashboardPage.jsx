import { useEffect, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import Header from '../components/Header';
import { loadQuizzes } from '../store/quizSlice';
import { createQuiz, deleteQuiz, checkAIStatus, generateQuiz } from '../api/quizzes';
import { createRoom } from '../api/rooms';
import { getSettings } from '../api/settings';
import './DashboardPage.css';

export default function DashboardPage() {
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const { list, loading } = useSelector((s) => s.quiz);
  const [creating, setCreating] = useState(false);
  const [newTitle, setNewTitle] = useState('');
  const [hasBotToken, setHasBotToken] = useState(null);
  const [aiAvailable, setAiAvailable] = useState(false);
  const [showAiModal, setShowAiModal] = useState(false);
  const [aiPrompt, setAiPrompt] = useState('');
  const [aiGenerating, setAiGenerating] = useState(false);
  const [aiError, setAiError] = useState('');

  useEffect(() => {
    dispatch(loadQuizzes());
    getSettings()
      .then(({ data }) => setHasBotToken(!!data.bot_token))
      .catch(() => setHasBotToken(false));
    checkAIStatus()
      .then(({ data }) => setAiAvailable(data.available))
      .catch(() => setAiAvailable(false));
  }, []);

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

  const handleLaunch = async (quizId, quizMode) => {
    try {
      const { data: room } = await createRoom(quizMode || 'web');
      navigate(`/room/${room.id}`, { state: { quizId } });
    } catch (err) {
      alert(err.response?.data?.error || 'Ошибка запуска');
    }
  };

  const handleAiGenerate = async () => {
    if (!aiPrompt.trim()) return;
    setAiGenerating(true);
    setAiError('');
    try {
      const { data } = await generateQuiz(aiPrompt.trim());
      const quizId = data.quiz?.id;
      setShowAiModal(false);
      setAiPrompt('');
      if (quizId) {
        navigate(`/quiz/${quizId}`);
      } else {
        dispatch(loadQuizzes());
      }
    } catch (err) {
      setAiError(err.response?.data?.error || 'Ошибка генерации');
    } finally {
      setAiGenerating(false);
    }
  };

  return (
    <>
      <Header />
      <div className="dashboard">
        <div className="dashboard-header">
          <h2>Мои квизы</h2>
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn btn-outline btn-sm" onClick={() => navigate('/history')}>История сессий</button>
            {aiAvailable && (
              <button className="btn btn-outline btn-sm ai-btn" onClick={() => setShowAiModal(true)}>
                AI Генерация
              </button>
            )}
            <button className="btn btn-primary btn-sm" onClick={() => setCreating(true)}>+ Создать квиз</button>
          </div>
        </div>

        {hasBotToken === false && (
          <div className="bot-token-warning">
            Для проведения квизов в боте необходимо добавить токен Telegram-бота в{' '}
            <a href="/settings" onClick={(e) => { e.preventDefault(); navigate('/settings'); }}>настройках</a>.
            Веб-квизы можно запускать без токена.
          </div>
        )}

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

        {loading ? (
          <div className="loading">Загрузка...</div>
        ) : list.length === 0 ? (
          <div className="empty-state">
            <h3>Пока нет квизов</h3>
            <p>Создайте свой первый квиз, чтобы начать</p>
          </div>
        ) : (
          <div className="quiz-grid">
            {list.map((q) => {
              const catCount = q.categories?.length || 0;
              const catQuestions = (q.categories || []).reduce((sum, c) => sum + (c.questions?.length || 0), 0);
              const orphanQuestions = q.questions?.length || 0;
              const totalQuestions = catQuestions + orphanQuestions;
              const isBot = q.mode === 'bot';
              const canLaunch = totalQuestions > 0 && (isBot ? hasBotToken : true);
              return (
              <div key={q.id} className="quiz-card">
                <h3>{q.title}</h3>
                <div className="quiz-meta">
                  {catCount} кат. &middot; {totalQuestions} вопр. &middot; {new Date(q.created_at).toLocaleDateString('ru')}
                </div>
                <div className="quiz-card-actions">
                  <button className="btn btn-outline btn-sm" onClick={() => navigate(`/quiz/${q.id}`)}>Редактировать</button>
                  <button
                    className="btn btn-success btn-sm"
                    onClick={() => handleLaunch(q.id, q.mode)}
                    disabled={!canLaunch}
                    title={isBot && !hasBotToken ? 'Добавьте токен бота в настройках' : !totalQuestions ? 'Добавьте хотя бы 1 вопрос' : ''}
                  >
                    Запустить
                  </button>
                  <button className="btn btn-danger btn-sm" onClick={() => handleDelete(q.id)}>Удалить</button>
                </div>
              </div>
              );
            })}
          </div>
        )}
      </div>

      {showAiModal && (
        <div className="ai-modal-overlay" onClick={() => !aiGenerating && setShowAiModal(false)}>
          <div className="ai-modal" onClick={(e) => e.stopPropagation()}>
            <h3>AI Генерация квиза</h3>
            <p className="ai-modal-hint">
              Опишите, какой квиз вы хотите создать. AI сгенерирует категории, вопросы, варианты ответов и цвета.
            </p>
            <textarea
              className="ai-prompt-input"
              value={aiPrompt}
              onChange={(e) => setAiPrompt(e.target.value)}
              placeholder="Например: Квиз про историю России, 3 категории по 5 вопросов, средняя сложность"
              rows={4}
              disabled={aiGenerating}
              autoFocus
            />
            {aiError && <div className="error-msg">{aiError}</div>}
            <div className="ai-modal-actions">
              <button
                className="btn btn-primary"
                onClick={handleAiGenerate}
                disabled={aiGenerating || !aiPrompt.trim()}
              >
                {aiGenerating ? 'Генерация...' : 'Сгенерировать'}
              </button>
              <button
                className="btn btn-outline"
                onClick={() => setShowAiModal(false)}
                disabled={aiGenerating}
              >
                Отмена
              </button>
            </div>
            {aiGenerating && (
              <div className="ai-loading">
                <div className="ai-spinner" />
                <span>AI генерирует квиз, это может занять до минуты...</span>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
}
