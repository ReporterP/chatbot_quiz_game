import { useEffect, useState, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { playJoin, playReconnect, playAnswer, playGetState, playUpdateNickname } from '../api/play';
import useRoomWebSocket from '../hooks/useRoomWebSocket';
import './PlayPage.css';

const LS_KEY = 'quizgame_play';

function loadStorage() {
  try {
    return JSON.parse(localStorage.getItem(LS_KEY)) || {};
  } catch { return {}; }
}

function saveStorage(data) {
  localStorage.setItem(LS_KEY, JSON.stringify(data));
}

function clearStorage() {
  localStorage.removeItem(LS_KEY);
}

function generateToken() {
  return 'xxxx-xxxx-xxxx'.replace(/x/g, () => Math.random().toString(36)[2]);
}

export default function PlayPage() {
  const [searchParams] = useSearchParams();
  const initialCode = searchParams.get('code') || '';

  const [phase, setPhase] = useState('join');
  const [code, setCode] = useState(initialCode);
  const [nickname, setNickname] = useState('');
  const [token, setToken] = useState('');
  const [room, setRoom] = useState(null);
  const [member, setMember] = useState(null);
  const [members, setMembers] = useState([]);
  const [session, setSession] = useState(null);
  const [myResult, setMyResult] = useState(null);
  const [selectedOption, setSelectedOption] = useState(null);
  const [answered, setAnswered] = useState(false);
  const [error, setError] = useState('');
  const [editingNick, setEditingNick] = useState(false);
  const [newNick, setNewNick] = useState('');
  const [lightboxImg, setLightboxImg] = useState(null);
  useEffect(() => {
    const stored = loadStorage();
    if (stored.token && stored.roomCode) {
      setToken(stored.token);
      setCode(stored.roomCode);
      tryReconnect(stored.token, stored.roomCode);
    }
  }, []);

  const tryReconnect = async (t, c) => {
    try {
      const { data } = await playReconnect(t, c);
      enterRoom(data, t);
    } catch {
      clearStorage();
    }
  };

  const enterRoom = (data, t) => {
    setRoom(data.room);
    setMember(data.member);
    setMembers(data.members || data.room?.members || []);
    setToken(t);
    saveStorage({
      token: t,
      roomCode: data.room.code,
      roomId: data.room.id,
      memberId: data.member.id,
      nickname: data.member.nickname,
    });
    if (data.current_session) {
      setSession(data.current_session);
      const status = data.current_session.status;
      if (status === 'waiting') setPhase('lobby');
      else if (status === 'question') { setPhase('question'); resetAnswer(); }
      else if (status === 'revealed' || status === 'finished') setPhase(status);
      else setPhase('lobby');
    } else {
      setPhase('lobby');
    }
  };

  const resetAnswer = () => {
    setSelectedOption(null);
    setAnswered(false);
    setMyResult(null);
  };

  const handleJoin = async (e) => {
    e.preventDefault();
    setError('');
    const t = token || generateToken();
    try {
      const { data } = await playJoin(code, nickname, t);
      enterRoom(data, t);
    } catch (err) {
      setError(err.response?.data?.error || 'Ошибка подключения');
    }
  };

  const refreshState = useCallback(async () => {
    if (!token || !room?.code) return;
    try {
      const { data } = await playGetState(token, room.code);
      setRoom(data.room);
      setMember(data.member);
      setMembers(data.members || []);
      if (data.current_session) {
        const prev = session;
        setSession(data.current_session);
        const status = data.current_session.status;
        const newQ = data.current_session.current_question;

        if (status === 'question') {
          if (!prev || prev.current_question !== newQ || prev.status !== 'question') {
            resetAnswer();
          }
          setPhase('question');
        } else if (status === 'revealed') {
          setPhase('revealed');
          if (data.my_result) setMyResult(data.my_result);
        } else if (status === 'finished') {
          setPhase('finished');
        } else if (status === 'waiting') {
          setPhase('lobby');
        }
      } else {
        setSession(null);
        setPhase('lobby');
        resetAnswer();
      }
    } catch {
      clearStorage();
      setPhase('join');
    }
  }, [token, room?.code, session]);

  const onWsMessage = useCallback((msg) => {
    if (msg.type === 'room_closed') {
      clearStorage();
      setPhase('join');
      setRoom(null);
      return;
    }
    refreshState();
  }, [refreshState]);

  useRoomWebSocket(room?.code, onWsMessage);

  const handleAnswer = async (optionId) => {
    if (answered || !session || !member) return;
    setSelectedOption(optionId);
    setAnswered(true);
    try {
      await playAnswer(session.id, member.id, token, optionId);
    } catch { /* ignore */ }
  };

  const handleNicknameUpdate = async () => {
    if (!newNick.trim() || !member) return;
    try {
      await playUpdateNickname(token, room.code, newNick.trim());
      setMember({ ...member, nickname: newNick.trim() });
      setEditingNick(false);
      saveStorage({ ...loadStorage(), nickname: newNick.trim() });
    } catch { /* ignore */ }
  };

  const handleLeave = () => {
    clearStorage();
    setPhase('join');
    setRoom(null);
    setMember(null);
    setSession(null);
  };

  if (phase === 'join') {
    return (
      <div className="play-page">
        <div className="play-container">
          <h1 className="play-title">Quiz Game</h1>
          <p className="play-subtitle">Введите код комнаты и ваш никнейм</p>
          <form onSubmit={handleJoin} className="join-form">
            <input className="play-input" value={code} onChange={(e) => setCode(e.target.value)} placeholder="Код комнаты" maxLength={6} required autoFocus />
            <input className="play-input" value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="Ваш никнейм" maxLength={100} required />
            {error && <div className="play-error">{error}</div>}
            <button type="submit" className="play-btn">Подключиться</button>
          </form>
        </div>
      </div>
    );
  }

  if (phase === 'lobby') {
    return (
      <div className="play-page">
        <div className="play-container">
          <h2 className="play-title">Вы в комнате!</h2>
          <div className="play-nick">
            {editingNick ? (
              <div className="nick-edit">
                <input className="play-input" value={newNick} onChange={(e) => setNewNick(e.target.value)} placeholder="Новый никнейм" maxLength={100} autoFocus />
                <button className="play-btn play-btn-sm" onClick={handleNicknameUpdate}>Сохранить</button>
                <button className="play-btn play-btn-sm play-btn-outline" onClick={() => setEditingNick(false)}>Отмена</button>
              </div>
            ) : (
              <div className="nick-display">
                <span>Никнейм: <strong>{member?.nickname}</strong></span>
                <button className="play-btn play-btn-sm play-btn-outline" onClick={() => { setNewNick(member?.nickname || ''); setEditingNick(true); }}>Сменить</button>
              </div>
            )}
          </div>
          <div className="play-members">
            <h3>Участники ({members.length})</h3>
            <div className="play-member-list">
              {members.map((m) => (
                <span key={m.id} className={`play-member-chip${m.id === member?.id ? ' me' : ''}`}>{m.nickname}</span>
              ))}
            </div>
          </div>
          <p className="play-waiting">Ожидайте начала квиза...</p>
          <button className="play-btn play-btn-outline play-btn-leave" onClick={handleLeave}>Выйти</button>
        </div>
      </div>
    );
  }

  const question = session?.current_question_data;
  const current = session?.current_question || 0;
  const total = session?.total_questions || 0;

  if (phase === 'question' && question) {
    return (
      <div className="play-page">
        <div className="play-container play-game">
          {question.category_name && <div className="play-category">{question.category_name}</div>}
          <div className="play-counter">Вопрос {current} из {total}</div>
          <div className="play-question">{question.text}</div>

          {question.images?.length > 0 && (
            <div className="play-images">
              {question.images.map((img) => (
                <img key={img.id} src={img.url} alt="" className="play-thumb" onClick={() => setLightboxImg(img.url)} />
              ))}
            </div>
          )}

          <div className="play-options">
            {question.options.map((opt) => {
              const isSelected = selectedOption === opt.id;
              return (
                <button
                  key={opt.id}
                  className={`play-option${isSelected ? ' selected' : ''}${answered && !isSelected ? ' dimmed' : ''}`}
                  style={{ background: opt.color || '#444' }}
                  onClick={() => handleAnswer(opt.id)}
                  disabled={answered}
                >
                  {opt.text}
                </button>
              );
            })}
          </div>

          {answered && <div className="play-answered-msg">Ответ принят! Ожидайте...</div>}

          {lightboxImg && (
            <div className="lightbox" onClick={() => setLightboxImg(null)}>
              <img src={lightboxImg} alt="" />
            </div>
          )}
        </div>
      </div>
    );
  }

  if (phase === 'revealed' && question) {
    return (
      <div className="play-page">
        <div className="play-container play-game">
          <div className="play-counter">Вопрос {current} из {total}</div>
          <div className="play-question">{question.text}</div>

          <div className="play-options">
            {question.options.map((opt) => {
              let cls = 'play-option revealed';
              if (opt.is_correct) cls += ' correct';
              else cls += ' wrong';
              if (selectedOption === opt.id) cls += ' mine';
              return <div key={opt.id} className={cls}>{opt.text}</div>;
            })}
          </div>

          {myResult && (
            <div className={`play-result-box${myResult.is_correct ? ' correct' : ' wrong'}`}>
              {myResult.is_correct ? '✓ Правильно!' : '✗ Неправильно'}
              {myResult.answered && <span className="play-result-score">+{myResult.score} очков (Всего: {myResult.total_score})</span>}
            </div>
          )}
        </div>
      </div>
    );
  }

  if (phase === 'finished') {
    const participants = session?.participants || [];
    const sorted = [...participants].sort((a, b) => b.total_score - a.total_score);
    const myPos = sorted.findIndex(p => p.member_id === member?.id) + 1;

    return (
      <div className="play-page">
        <div className="play-container">
          <h2 className="play-title">Квиз завершён!</h2>
          {myPos > 0 && (
            <div className="play-my-position">
              Ваше место: <strong>{myPos}</strong> из {sorted.length}
            </div>
          )}
          <div className="play-leaderboard">
            {sorted.slice(0, 10).map((p, i) => (
              <div key={p.id} className={`play-lb-row${p.member_id === member?.id ? ' me' : ''}`}>
                <span className="play-lb-pos">{i + 1}</span>
                <span className="play-lb-name">{p.nickname}</span>
                <span className="play-lb-score">{p.total_score}</span>
              </div>
            ))}
          </div>
          <p className="play-waiting">Ожидайте следующий квиз или выйдите из комнаты</p>
          <button className="play-btn play-btn-outline play-btn-leave" onClick={handleLeave}>Выйти</button>
        </div>
      </div>
    );
  }

  return (
    <div className="play-page">
      <div className="play-container">
        <div className="loading" style={{ color: 'white' }}>Загрузка...</div>
      </div>
    </div>
  );
}
