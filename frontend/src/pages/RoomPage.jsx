import { useEffect, useState, useCallback } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { QRCodeSVG } from 'qrcode.react';
import { fetchRoom, closeRoom, startQuizInRoom, roomReveal, roomNext, roomFinish, roomLeaderboard } from '../api/rooms';
import { fetchQuizzes } from '../api/quizzes';
import { getSettings } from '../api/settings';
import useRoomWebSocket from '../hooks/useRoomWebSocket';
import './SessionPage.css';
import './RoomPage.css';

export default function RoomPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const initialQuizId = location.state?.quizId;

  const [room, setRoom] = useState(null);
  const [session, setSession] = useState(null);
  const [leaderboard, setLeaderboard] = useState([]);
  const [quizzes, setQuizzes] = useState([]);
  const [selectedQuizId, setSelectedQuizId] = useState(initialQuizId || '');
  const [botLink, setBotLink] = useState('');
  const [loading, setLoading] = useState(true);
  const [lightboxImg, setLightboxImg] = useState(null);

  const loadRoom = useCallback(async () => {
    try {
      const { data } = await fetchRoom(id);
      setRoom(data.room);
      setSession(data.current_session);
      if (data.current_session?.status === 'finished') {
        const { data: lb } = await roomLeaderboard(id);
        setLeaderboard(lb);
      }
    } catch {
      navigate('/dashboard');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadRoom();
    fetchQuizzes().then(({ data }) => setQuizzes(data));
    getSettings().then(({ data }) => setBotLink(data.bot_link || '')).catch(() => {});
  }, [id]);

  const onWsMessage = useCallback((msg) => {
    if (msg.type === 'room_closed') {
      navigate('/dashboard');
      return;
    }
    loadRoom();
  }, [loadRoom]);

  useRoomWebSocket(room?.code, onWsMessage);

  const handleStartQuiz = async () => {
    if (!selectedQuizId) return;
    try {
      const { data } = await startQuizInRoom(id, Number(selectedQuizId));
      setSession(data);
    } catch (err) {
      alert(err.response?.data?.error || 'Ошибка запуска');
    }
  };

  const handleReveal = async () => {
    try {
      const { data } = await roomReveal(id);
      setSession(data);
    } catch (err) {
      alert(err.response?.data?.error || 'Ошибка');
    }
  };

  const handleNext = async () => {
    try {
      const { data } = await roomNext(id);
      setSession(data);
      if (data.status === 'finished') {
        const { data: lb } = await roomLeaderboard(id);
        setLeaderboard(lb);
      }
    } catch (err) {
      alert(err.response?.data?.error || 'Ошибка');
    }
  };

  const handleFinish = async () => {
    try {
      const { data } = await roomFinish(id);
      setSession(data);
      const { data: lb } = await roomLeaderboard(id);
      setLeaderboard(lb);
    } catch (err) {
      alert(err.response?.data?.error || 'Ошибка');
    }
  };

  const handleCloseRoom = async () => {
    if (!confirm('Закрыть комнату? Все участники будут отключены.')) return;
    await closeRoom(id);
    navigate('/dashboard');
  };

  const handleNextQuiz = () => {
    setSession(null);
    setLeaderboard([]);
    setSelectedQuizId('');
  };

  if (loading) {
    return <div className="room-page"><div className="room-body"><div className="loading" style={{ color: 'white' }}>Загрузка...</div></div></div>;
  }

  if (!room) return null;

  const status = session?.status;
  const question = session?.current_question_data;
  const participants = session?.participants || [];
  const members = room.members || [];
  const total = session?.total_questions || 0;
  const current = session?.current_question || 0;

  const isWeb = room.mode === 'web';
  const playUrl = `${window.location.origin}/play?code=${room.code}`;
  const botQrUrl = botLink ? `${botLink}?start=${room.code}` : '';
  const qrUrl = isWeb ? playUrl : botQrUrl;

  return (
    <div className="room-page">
      <div className="room-header">
        <a className="back-link" href="#" onClick={(e) => { e.preventDefault(); navigate('/dashboard'); }}>← Вернуться</a>
        <div className="room-info">
          Комната: {room.code} &middot; {isWeb ? 'Веб' : 'Бот'}
        </div>
        <button className="btn-close-room" onClick={handleCloseRoom}>Закрыть комнату</button>
      </div>
      <div className="room-body">
        {!session && <RoomLobby room={room} members={members} qrUrl={qrUrl} playUrl={playUrl} isWeb={isWeb} quizzes={quizzes} selectedQuizId={selectedQuizId} onSelectQuiz={setSelectedQuizId} onStart={handleStartQuiz} />}

        {status === 'waiting' && (
          <WaitingScreen session={session} members={members} participants={participants} qrUrl={qrUrl} playUrl={playUrl} isWeb={isWeb} onStart={handleNext} />
        )}

        {(status === 'question' || status === 'revealed') && question && (
          <GameScreen question={question} current={current} total={total} status={status} answerCount={session.answer_count} participantCount={participants.length} onReveal={handleReveal} onNext={handleNext} onFinish={handleFinish} isLast={current >= total} lightboxImg={lightboxImg} setLightboxImg={setLightboxImg} />
        )}

        {status === 'finished' && (
          <FinishedScreen leaderboard={leaderboard} onNextQuiz={handleNextQuiz} onCloseRoom={handleCloseRoom} />
        )}
      </div>
    </div>
  );
}

function RoomLobby({ room, members, qrUrl, playUrl, isWeb, quizzes, selectedQuizId, onSelectQuiz, onStart }) {
  return (
    <div className="lobby">
      <h2>Лобби комнаты</h2>
      <p className="lobby-subtitle">{isWeb ? 'Участники подключаются через веб-страницу' : 'Участники подключаются через Telegram-бота'}</p>
      <div className="lobby-code-block">
        <div className="lobby-code-label">Код комнаты</div>
        <div className="lobby-code">{room.code}</div>
      </div>
      {qrUrl && <div className="lobby-qr"><QRCodeSVG value={qrUrl} size={180} /></div>}
      {isWeb && <div className="lobby-link"><a href={playUrl} target="_blank" rel="noreferrer">{playUrl}</a></div>}
      <div className="lobby-participants">
        <h3>Участники ({members.length})</h3>
        {members.length === 0 ? (
          <p style={{ color: 'rgba(255,255,255,0.3)', fontSize: 14 }}>Пока никто не подключился...</p>
        ) : (
          <div className="participant-chips">
            {members.map((m) => <span key={m.id} className="participant-chip">{m.nickname}</span>)}
          </div>
        )}
      </div>
      <div className="quiz-select-block">
        <h3>Выберите квиз</h3>
        <select value={selectedQuizId} onChange={(e) => onSelectQuiz(e.target.value)} className="quiz-select">
          <option value="">-- Выбрать квиз --</option>
          {quizzes.map((q) => <option key={q.id} value={q.id}>{q.title}</option>)}
        </select>
        <button className="btn-game btn-next" onClick={onStart} disabled={!selectedQuizId || members.length === 0}>Начать квиз</button>
      </div>
    </div>
  );
}

function WaitingScreen({ session, members, participants, qrUrl, playUrl, isWeb, onStart }) {
  return (
    <div className="lobby">
      <h2>Ожидание начала</h2>
      <div className="lobby-code-block">
        <div className="lobby-code-label">Код</div>
        <div className="lobby-code">{session.code}</div>
      </div>
      {qrUrl && <div className="lobby-qr"><QRCodeSVG value={qrUrl} size={140} /></div>}
      <div className="lobby-participants">
        <h3>Участники ({members.length})</h3>
        <div className="participant-chips">
          {members.map((m) => <span key={m.id} className="participant-chip">{m.nickname}</span>)}
        </div>
      </div>
      <button className="btn-game btn-next" onClick={onStart} disabled={participants.length === 0}>Начать</button>
    </div>
  );
}

function GameScreen({ question, current, total, status, answerCount, participantCount, onReveal, onNext, onFinish, isLast, lightboxImg, setLightboxImg }) {
  const isRevealed = status === 'revealed';

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
          return <div key={opt.id} className={cls} style={{ background: isRevealed ? undefined : bg }}>{opt.text}</div>;
        })}
      </div>

      <div className="game-controls">
        <span className="answer-count">Ответили: {answerCount} / {participantCount}</span>
        {status === 'question' && (
          <>
            <button className="btn-game btn-reveal" onClick={onReveal}>Показать ответ</button>
            <button className="btn-game btn-finish-early" onClick={onFinish}>Завершить досрочно</button>
          </>
        )}
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

function FinishedScreen({ leaderboard, onNextQuiz, onCloseRoom }) {
  const top3 = leaderboard.slice(0, 3);
  const rest = leaderboard.slice(3);
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

      {leaderboard.length === 0 && <p style={{ color: 'rgba(255,255,255,0.5)', textAlign: 'center' }}>Нет участников</p>}

      <div className="finish-actions">
        <button className="btn-game btn-next" onClick={onNextQuiz}>Следующий квиз</button>
        <button className="btn-game btn-finish-early" onClick={onCloseRoom}>Закрыть комнату</button>
      </div>
    </div>
  );
}
