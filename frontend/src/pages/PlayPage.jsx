import { useEffect, useState, useCallback, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { playJoin, playReconnect, playAnswer, playGetState, playUpdateNickname, playLeave } from '../api/play';
import useRoomWebSocket from '../hooks/useRoomWebSocket';
import './PlayPage.css';

const LS_KEY = 'quizgame_play';
const LS_TOKEN_KEY = 'quizgame_device_token';

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

function getDeviceToken() {
  let t = localStorage.getItem(LS_TOKEN_KEY);
  if (!t) {
    t = 'xxxx-xxxx-xxxx-xxxx'.replace(/x/g, () => Math.random().toString(36)[2]);
    localStorage.setItem(LS_TOKEN_KEY, t);
  }
  return t;
}

export default function PlayPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const urlCode = searchParams.get('code') || '';

  const [phase, setPhase] = useState('join');
  const [code, setCode] = useState('');
  const [nickname, setNickname] = useState('');
  const [token] = useState(() => getDeviceToken());
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
  const [leaderboard, setLeaderboard] = useState([]);

  const sessionRef = useRef(null);
  const tokenRef = useRef(token);
  const roomCodeRef = useRef(null);

  useEffect(() => {
    const stored = loadStorage();
    if (stored.roomCode) {
      setCode(stored.roomCode);
      tryReconnect(token, stored.roomCode);
    } else if (urlCode) {
      setCode(urlCode);
    }
  }, []);

  const tryReconnect = async (t, c) => {
    try {
      const { data } = await playReconnect(t, c);
      enterRoom(data);
      if (data.leaderboard) setLeaderboard(data.leaderboard);
    } catch {
      clearStorage();
    }
  };

  const enterRoom = (data) => {
    setRoom(data.room);
    setMember(data.member);
    setMembers(data.members || data.room?.members || []);
    roomCodeRef.current = data.room.code;
    saveStorage({
      roomCode: data.room.code,
      roomId: data.room.id,
      memberId: data.member.id,
      nickname: data.member.nickname,
    });
    if (data.current_session) {
      setSession(data.current_session);
      sessionRef.current = data.current_session;
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
    try {
      const { data } = await playJoin(code, nickname, token);
      enterRoom(data);
    } catch (err) {
      setError(err.response?.data?.error || 'Ошибка подключения');
    }
  };

  const refreshState = useCallback(async () => {
    const t = tokenRef.current;
    const rc = roomCodeRef.current;
    if (!t || !rc) return;
    try {
      const { data } = await playGetState(t, rc);
      setRoom(data.room);
      setMember(data.member);
      setMembers(data.members || []);
      if (data.current_session) {
        const prev = sessionRef.current;
        setSession(data.current_session);
        sessionRef.current = data.current_session;
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
          if (data.leaderboard) setLeaderboard(data.leaderboard);
        } else if (status === 'waiting') {
          setPhase('lobby');
          resetAnswer();
          setLeaderboard([]);
        }
      } else {
        setSession(null);
        sessionRef.current = null;
        setPhase('lobby');
        resetAnswer();
        setLeaderboard([]);
      }
    } catch {
      clearStorage();
      setPhase('join');
    }
  }, []);

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
    if (!session || !member) return;
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

  const handleLeave = async () => {
    try {
      if (token && room?.code) {
        await playLeave(token, room.code);
      }
    } catch { /* ignore */ }
    clearStorage();
    setSearchParams({}, { replace: true });
    setPhase('join');
    setRoom(null);
    setMember(null);
    setSession(null);
    sessionRef.current = null;
    roomCodeRef.current = null;
    setCode('');
    setNickname('');
    setLeaderboard([]);
    resetAnswer();
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
                  className={`play-option${isSelected ? ' selected' : ''}`}
                  style={{ background: opt.color || '#444' }}
                  onClick={() => handleAnswer(opt.id)}
                >
                  {opt.text}
                </button>
              );
            })}
          </div>

          {answered && <div className="play-answered-msg">✅ Ответ принят!</div>}
          {session?.answer_count != null && (
            <div className="play-answer-count">Ответили: {session.answer_count} из {session.participants?.length || members.length}</div>
          )}

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
              const isMine = selectedOption === opt.id;
              let cls = 'play-option revealed';
              if (opt.is_correct) cls += ' correct';
              else if (isMine) cls += ' wrong';
              else cls += ' neutral';
              if (isMine) cls += ' mine';
              return <div key={opt.id} className={cls}>{opt.text}</div>;
            })}
          </div>

          {myResult && (
            <div className={`play-result-box${myResult.is_correct ? ' correct' : ' wrong'}`}>
              {myResult.is_correct ? '✓ Правильно!' : '✗ Неправильно'}
              {myResult.answered && <span className="play-result-score">+{myResult.score} очков (Всего: {myResult.total_score})</span>}
            </div>
          )}

          {lightboxImg && (
            <div className="lightbox" onClick={() => setLightboxImg(null)}>
              <img src={lightboxImg} alt="" />
            </div>
          )}
        </div>
      </div>
    );
  }

  if (phase === 'finished') {
    const myEntry = leaderboard.find(e => e.member_id === member?.id);
    const myPos = myEntry?.position || 0;

    return (
      <div className="play-page">
        <div className="play-container">
          <h2 className="play-title">Квиз завершён!</h2>
          {myPos > 0 && (
            <div className="play-my-position">
              Ваше место: <strong>{myPos}</strong> из {leaderboard.length}
            </div>
          )}
          <div className="play-leaderboard">
            {leaderboard.map((e) => (
              <div key={e.position} className={`play-lb-row${e.member_id === member?.id ? ' me' : ''}`}>
                <span className="play-lb-pos">
                  {e.position <= 3 ? ['🥇','🥈','🥉'][e.position - 1] : e.position}
                </span>
                <span className="play-lb-name">{e.nickname}</span>
                <span className="play-lb-score">{e.total_score}</span>
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
