import { useEffect, useCallback, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useDispatch, useSelector } from 'react-redux';
import { QRCodeSVG } from 'qrcode.react';
import { loadSession, doReveal, doNext, loadLeaderboard, clearSession } from '../store/sessionSlice';
import { getSettings } from '../api/settings';
import useWebSocket from '../hooks/useWebSocket';
import './SessionPage.css';

export default function SessionPage() {
  const { id } = useParams();
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const { data: session, leaderboard, loading } = useSelector((s) => s.session);
  const [botLink, setBotLink] = useState('');

  useEffect(() => {
    dispatch(loadSession(id));
    getSettings().then(({ data }) => setBotLink(data.bot_link || '')).catch(() => {});
    return () => dispatch(clearSession());
  }, [id]);

  useEffect(() => {
    if (session?.status === 'finished') dispatch(loadLeaderboard(id));
  }, [session?.status]);

  const onWsMessage = useCallback(() => { dispatch(loadSession(id)); }, [id]);
  useWebSocket(id, onWsMessage);

  const handleReveal = () => dispatch(doReveal(id));
  const handleNext = () => dispatch(doNext(id));

  if (loading && !session) {
    return <div className="session-page"><div className="session-body"><div className="loading" style={{ color: 'white' }}>Загрузка...</div></div></div>;
  }
  if (!session) return null;

  const status = session.status;
  const question = session.current_question_data;
  const participants = session.participants || [];
  const total = session.total_questions;
  const current = session.current_question;

  return (
    <div className="session-page">
      <div className="session-header">
        <a className="back-link" href="#" onClick={(e) => { e.preventDefault(); navigate('/dashboard'); }}>← Вернуться</a>
        <div className="session-info">
          {session.quiz?.title} &middot; Код: {session.code}
        </div>
      </div>
      <div className="session-body">
        {status === 'waiting' && <Lobby session={session} participants={participants} onStart={handleNext} botLink={botLink} />}
        {(status === 'question' || status === 'revealed') && question && (
          <GameScreen question={question} current={current} total={total} status={status} answerCount={session.answer_count} participantCount={participants.length} onReveal={handleReveal} onNext={handleNext} isLast={current >= total} />
        )}
        {status === 'finished' && <Leaderboard entries={leaderboard} onBack={() => navigate('/dashboard')} />}
      </div>
    </div>
  );
}

function Lobby({ session, participants, onStart, botLink }) {
  const qrUrl = botLink ? `${botLink}?start=${session.code}` : '';
  return (
    <div className="lobby">
      <h2>Ожидание участников</h2>
      <p className="lobby-subtitle">Попросите участников подключиться через Telegram-бота</p>
      <div className="lobby-code-block">
        <div className="lobby-code-label">Код для подключения</div>
        <div className="lobby-code">{session.code}</div>
      </div>
      {qrUrl ? (
        <div className="lobby-qr"><QRCodeSVG value={qrUrl} size={180} /></div>
      ) : (
        <div className="lobby-qr-hint">Укажите ссылку на бота в <a href="/settings" style={{ color: '#4fc3f7' }}>настройках</a></div>
      )}
      <div className="lobby-participants">
        <h3>Участники ({participants.length})</h3>
        {participants.length === 0 ? (
          <p style={{ color: 'rgba(255,255,255,0.3)', fontSize: 14 }}>Пока никто не подключился...</p>
        ) : (
          <div className="participant-chips">
            {participants.map((p) => <span key={p.id} className="participant-chip">{p.nickname}</span>)}
          </div>
        )}
      </div>
      <button className="btn-game btn-next" onClick={onStart} disabled={participants.length === 0}>Начать квиз</button>
    </div>
  );
}

function GameScreen({ question, current, total, status, answerCount, participantCount, onReveal, onNext, isLast }) {
  const isRevealed = status === 'revealed';
  const [lightboxImg, setLightboxImg] = useState(null);

  return (
    <div className="game-screen">
      {question.category_name && <div className="category-badge">{question.category_name}</div>}
      <div className="question-counter">Вопрос {current} из {total}</div>
      <div className="question-text">{question.text}</div>

      {question.images?.length > 0 && (
        <div className="question-images-game">
          {question.images.map((img) => (
            <img key={img.id} src={img.url} alt="" className="game-thumb" onClick={() => setLightboxImg(img.url)} />
          ))}
        </div>
      )}

      <div className="game-options">
        {question.options.map((opt) => {
          let cls = 'game-option';
          if (isRevealed && opt.is_correct) cls += ' correct';
          if (isRevealed && !opt.is_correct) cls += ' wrong';
          const bg = opt.color || '#444';
          return (
            <div key={opt.id} className={cls} style={{ background: isRevealed ? undefined : bg }}>
              {opt.text}
            </div>
          );
        })}
      </div>

      <div className="game-controls">
        <span className="answer-count">Ответили: {answerCount} / {participantCount}</span>
        {status === 'question' && <button className="btn-game btn-reveal" onClick={onReveal}>Показать ответ</button>}
        {status === 'revealed' && (
          <button className="btn-game btn-next" onClick={onNext}>
            {isLast ? 'Показать результаты' : 'Следующий вопрос →'}
          </button>
        )}
      </div>

      {lightboxImg && (
        <div className="lightbox" onClick={() => setLightboxImg(null)}>
          <img src={lightboxImg} alt="" />
        </div>
      )}
    </div>
  );
}

function Leaderboard({ entries, onBack }) {
  const top3 = entries.slice(0, 3);
  const rest = entries.slice(3);
  const order = [1, 0, 2];

  return (
    <div className="leaderboard">
      <h2>Таблица лидеров</h2>

      {top3.length > 0 && (
        <div className="podium">
          {order.map((idx) => {
            const e = top3[idx];
            if (!e) return <div key={idx} className="podium-item" />;
            const cls = idx === 0 ? 'first' : idx === 1 ? 'second' : 'third';
            return (
              <div key={idx} className="podium-item">
                <div className={`podium-bar ${cls}`}>
                  <div className="podium-place">{e.position}</div>
                  <div className="podium-name">{e.nickname}</div>
                  <div className="podium-score">{e.total_score} очков</div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {rest.length > 0 && (
        <div className="leaderboard-table">
          {rest.map((e) => (
            <div key={e.position} className="leaderboard-row">
              <span className="pos">{e.position}</span>
              <span className="name">{e.nickname}</span>
              <span className="score">{e.total_score} очков</span>
            </div>
          ))}
        </div>
      )}

      {entries.length === 0 && <p style={{ color: 'rgba(255,255,255,0.5)', textAlign: 'center' }}>Нет участников</p>}

      <button className="btn-game btn-next" style={{ marginTop: 32 }} onClick={onBack}>Вернуться в кабинет</button>
    </div>
  );
}
