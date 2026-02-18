import { useEffect, useState, useCallback, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { DndContext, closestCenter, PointerSensor, TouchSensor, useSensor, useSensors } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy, useSortable, arrayMove } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { playJoin, playReconnect, playAnswer, playAnswerComplex, playGetState, playUpdateNickname, playLeave } from '../api/play';
import useRoomWebSocket from '../hooks/useRoomWebSocket';
import MediaViewer, { Lightbox } from '../components/MediaViewer';
import './PlayPage.css';

const LS_KEY = 'quizgame_play';
const LS_TOKEN_KEY = 'quizgame_device_token';

function loadStorage() { try { return JSON.parse(localStorage.getItem(LS_KEY)) || {}; } catch { return {}; } }
function saveStorage(data) { localStorage.setItem(LS_KEY, JSON.stringify(data)); }
function clearStorage() { localStorage.removeItem(LS_KEY); }
function getDeviceToken() {
  let t = localStorage.getItem(LS_TOKEN_KEY);
  if (!t) { t = 'xxxx-xxxx-xxxx-xxxx'.replace(/x/g, () => Math.random().toString(36)[2]); localStorage.setItem(LS_TOKEN_KEY, t); }
  return t;
}

function SortablePlayItem({ id, children }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });
  const style = { transform: CSS.Transform.toString(transform), transition, opacity: isDragging ? 0.5 : 1 };
  return (
    <div ref={setNodeRef} style={style} className="play-sortable-item" {...attributes} {...listeners}>
      {children}
    </div>
  );
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
  const [selectedOptions, setSelectedOptions] = useState([]);
  const [orderItems, setOrderItems] = useState([]);
  const [matchPairs, setMatchPairs] = useState({});
  const [numericValue, setNumericValue] = useState('');
  const [answered, setAnswered] = useState(false);
  const [error, setError] = useState('');
  const [editingNick, setEditingNick] = useState(false);
  const [newNick, setNewNick] = useState('');
  const [lightboxImg, setLightboxImg] = useState(null);
  const [leaderboard, setLeaderboard] = useState([]);

  const sessionRef = useRef(null);
  const tokenRef = useRef(token);
  const roomCodeRef = useRef(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 6 } }),
  );

  useEffect(() => {
    const stored = loadStorage();
    if (stored.roomCode) { setCode(stored.roomCode); tryReconnect(token, stored.roomCode); }
    else if (urlCode) setCode(urlCode);
  }, []);

  const tryReconnect = async (t, c) => {
    try {
      const { data } = await playReconnect(t, c);
      enterRoom(data);
      if (data.leaderboard) setLeaderboard(data.leaderboard);
    } catch { clearStorage(); }
  };

  const enterRoom = (data) => {
    setRoom(data.room);
    setMember(data.member);
    setMembers(data.members || data.room?.members || []);
    roomCodeRef.current = data.room.code;
    saveStorage({ roomCode: data.room.code, roomId: data.room.id, memberId: data.member.id, nickname: data.member.nickname });
    if (data.current_session) {
      setSession(data.current_session);
      sessionRef.current = data.current_session;
      const status = data.current_session.status;
      if (status === 'waiting') setPhase('lobby');
      else if (status === 'question') { setPhase('question'); resetAnswer(data.current_session); }
      else if (status === 'revealed' || status === 'finished') setPhase(status);
      else setPhase('lobby');
    } else setPhase('lobby');
  };

  const resetAnswer = (sess) => {
    setSelectedOption(null);
    setSelectedOptions([]);
    setNumericValue('');
    setAnswered(false);
    setMyResult(null);
    if (sess?.current_question_data) {
      const q = sess.current_question_data;
      if (q.type === 'ordering' && q.options) {
        const shuffled = [...q.options].sort(() => Math.random() - 0.5);
        setOrderItems(shuffled.map(o => o.id));
      }
      if (q.type === 'matching' && q.options) {
        setMatchPairs({});
      }
    }
  };

  const handleJoin = async (e) => {
    e.preventDefault();
    setError('');
    try { const { data } = await playJoin(code, nickname, token); enterRoom(data); }
    catch (err) { setError(err.response?.data?.error || 'Ошибка подключения'); }
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
          if (!prev || prev.current_question !== newQ || prev.status !== 'question') resetAnswer(data.current_session);
          setPhase('question');
        } else if (status === 'revealed') {
          setPhase('revealed');
          if (data.my_result) setMyResult(data.my_result);
        } else if (status === 'finished') {
          setPhase('finished');
          if (data.leaderboard) setLeaderboard(data.leaderboard);
        } else if (status === 'waiting') {
          setPhase('lobby');
          resetAnswer(null);
          setLeaderboard([]);
        }
      } else {
        setSession(null); sessionRef.current = null;
        setPhase('lobby'); resetAnswer(null); setLeaderboard([]);
      }
    } catch { clearStorage(); setPhase('join'); }
  }, []);

  const onWsMessage = useCallback((msg) => {
    if (msg.type === 'room_closed') { clearStorage(); setPhase('join'); setRoom(null); return; }
    refreshState();
  }, [refreshState]);

  useRoomWebSocket(room?.code, onWsMessage);

  const handleSingleAnswer = async (optionId) => {
    if (!session || !member) return;
    setSelectedOption(optionId);
    setAnswered(true);
    try { await playAnswer(session.id, member.id, token, optionId); } catch {}
  };

  const handleMultiToggle = (optionId) => {
    setSelectedOptions(prev => prev.includes(optionId) ? prev.filter(id => id !== optionId) : [...prev, optionId]);
  };

  const submitMultiAnswer = async () => {
    if (!session || !member || selectedOptions.length === 0) return;
    setAnswered(true);
    try { await playAnswerComplex(session.id, member.id, token, { option_ids: selectedOptions }); } catch {}
  };

  const handleOrderDragEnd = (event) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    setOrderItems(prev => {
      const oldIdx = prev.indexOf(active.id);
      const newIdx = prev.indexOf(over.id);
      return arrayMove(prev, oldIdx, newIdx);
    });
  };

  const submitOrderAnswer = async () => {
    if (!session || !member) return;
    setAnswered(true);
    try { await playAnswerComplex(session.id, member.id, token, { order: orderItems }); } catch {}
  };

  const handleMatchSelect = (leftId, rightText) => {
    setMatchPairs(prev => ({ ...prev, [String(leftId)]: rightText }));
  };

  const submitMatchAnswer = async () => {
    if (!session || !member) return;
    setAnswered(true);
    try { await playAnswerComplex(session.id, member.id, token, { pairs: matchPairs }); } catch {}
  };

  const submitNumericAnswer = async () => {
    if (!session || !member || numericValue === '') return;
    setAnswered(true);
    try { await playAnswerComplex(session.id, member.id, token, { value: parseFloat(numericValue) }); } catch {}
  };

  const handleNicknameUpdate = async () => {
    if (!newNick.trim() || !member) return;
    try {
      await playUpdateNickname(token, room.code, newNick.trim());
      setMember({ ...member, nickname: newNick.trim() });
      setEditingNick(false);
      saveStorage({ ...loadStorage(), nickname: newNick.trim() });
    } catch {}
  };

  const handleLeave = async () => {
    try { if (token && room?.code) await playLeave(token, room.code); } catch {}
    clearStorage(); setSearchParams({}, { replace: true });
    setPhase('join'); setRoom(null); setMember(null); setSession(null);
    sessionRef.current = null; roomCodeRef.current = null;
    setCode(''); setNickname(''); setLeaderboard([]); resetAnswer(null);
  };

  // --- RENDER ---
  if (phase === 'join') return (
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

  if (phase === 'lobby') return (
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
        <div className="play-members"><h3>Участники ({members.length})</h3>
          <div className="play-member-list">{members.map((m) => (
            <span key={m.id} className={`play-member-chip${m.id === member?.id ? ' me' : ''}`}>{m.nickname}</span>
          ))}</div>
        </div>
        <p className="play-waiting">Ожидайте начала квиза...</p>
        <button className="play-btn play-btn-outline play-btn-leave" onClick={handleLeave}>Выйти</button>
      </div>
    </div>
  );

  const question = session?.current_question_data;
  const current = session?.current_question || 0;
  const total = session?.total_questions || 0;
  const qType = question?.type || 'single_choice';

  const renderMedia = () => (
    <MediaViewer items={question?.images} onLightbox={setLightboxImg} compact />
  );

  if (phase === 'question' && question) {
    return (
      <div className="play-page">
        <div className="play-container play-game">
          {question.category_name && <div className="play-category">{question.category_name}</div>}
          <div className="play-counter">Вопрос {current} из {total}</div>
          <div className="play-question">{question.text}</div>
          {renderMedia()}

          {qType === 'single_choice' && (
            <div className="play-options">
              {question.options.map((opt) => (
                <button key={opt.id} className={`play-option${selectedOption === opt.id ? ' selected' : ''}`} style={{ background: opt.color || '#444' }} onClick={() => handleSingleAnswer(opt.id)}>{opt.text}</button>
              ))}
            </div>
          )}

          {qType === 'multiple_choice' && (
            <>
              <div className="play-options">
                {question.options.map((opt) => (
                  <button key={opt.id} className={`play-option play-option-multi${selectedOptions.includes(opt.id) ? ' selected' : ''}`} style={{ background: opt.color || '#444' }} onClick={() => !answered && handleMultiToggle(opt.id)}>
                    <span className="multi-check">{selectedOptions.includes(opt.id) ? '☑' : '☐'}</span> {opt.text}
                  </button>
                ))}
              </div>
              {!answered && selectedOptions.length > 0 && (
                <button className="play-btn play-confirm-btn" onClick={submitMultiAnswer}>Подтвердить ({selectedOptions.length})</button>
              )}
            </>
          )}

          {qType === 'ordering' && (
            <>
              <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleOrderDragEnd}>
                <SortableContext items={orderItems} strategy={verticalListSortingStrategy}>
                  <div className="play-ordering-list">
                    {orderItems.map((optId, idx) => {
                      const opt = question.options.find(o => o.id === optId);
                      if (!opt) return null;
                      return (
                        <SortablePlayItem key={optId} id={optId}>
                          <div className="play-order-item">
                            <span className="play-order-num">{idx + 1}</span>
                            <span className="play-order-handle">⠿</span>
                            <span className="play-order-text">{opt.text}</span>
                          </div>
                        </SortablePlayItem>
                      );
                    })}
                  </div>
                </SortableContext>
              </DndContext>
              {!answered && <button className="play-btn play-confirm-btn" onClick={submitOrderAnswer}>Подтвердить порядок</button>}
            </>
          )}

          {qType === 'matching' && (
            <>
              <div className="play-matching">
                {question.options.map((opt) => {
                  const allMatchTexts = question.options.map(o => o.match_text);
                  return (
                    <div key={opt.id} className="play-match-row">
                      <div className="play-match-left">{opt.text}</div>
                      <span className="play-match-arrow">→</span>
                      <select className="play-match-select" value={matchPairs[String(opt.id)] || ''} onChange={(e) => handleMatchSelect(opt.id, e.target.value)} disabled={answered}>
                        <option value="">Выберите...</option>
                        {allMatchTexts.map((mt, i) => <option key={i} value={mt}>{mt}</option>)}
                      </select>
                    </div>
                  );
                })}
              </div>
              {!answered && Object.keys(matchPairs).length === question.options.length && (
                <button className="play-btn play-confirm-btn" onClick={submitMatchAnswer}>Подтвердить</button>
              )}
            </>
          )}

          {qType === 'numeric' && (
            <div className="play-numeric">
              <input type="number" step="any" inputMode="decimal" className="play-numeric-input" value={numericValue} onChange={(e) => setNumericValue(e.target.value)} placeholder="Введите число..." disabled={answered} autoFocus />
              {!answered && numericValue !== '' && (
                <button className="play-btn play-confirm-btn" onClick={submitNumericAnswer}>Ответить</button>
              )}
            </div>
          )}

          {answered && <div className="play-answered-msg">✅ Ответ принят!</div>}
          {session?.answer_count != null && (
            <div className="play-answer-count">Ответили: {session.answer_count} из {session.participants?.length || members.length}</div>
          )}
          <Lightbox src={lightboxImg} onClose={() => setLightboxImg(null)} />
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
          {renderMedia()}

          {(qType === 'single_choice' || qType === 'multiple_choice') && (
            <div className="play-options">
              {question.options.map((opt) => {
                const isMine = qType === 'single_choice' ? selectedOption === opt.id : selectedOptions.includes(opt.id);
                let cls = 'play-option revealed';
                if (opt.is_correct) cls += ' correct';
                else if (isMine) cls += ' wrong';
                else cls += ' neutral';
                if (isMine) cls += ' mine';
                return <div key={opt.id} className={cls}>{qType === 'multiple_choice' && <span className="multi-check">{opt.is_correct ? '✓' : '✗'}</span>} {opt.text}</div>;
              })}
            </div>
          )}

          {qType === 'ordering' && (
            <div className="play-ordering-list revealed">
              {[...(question.options || [])].sort((a, b) => (a.correct_position || 0) - (b.correct_position || 0)).map((opt, idx) => {
                const userIdx = orderItems.indexOf(opt.id);
                const isCorrectPos = userIdx === idx;
                return (
                  <div key={opt.id} className={`play-order-item revealed${isCorrectPos ? ' correct' : ' wrong'}`}>
                    <span className="play-order-num">{idx + 1}</span>
                    <span className="play-order-text">{opt.text}</span>
                    {!isCorrectPos && userIdx >= 0 && <span className="play-order-your">(вы: {userIdx + 1})</span>}
                  </div>
                );
              })}
            </div>
          )}

          {qType === 'matching' && (
            <div className="play-matching revealed">
              {question.options.map((opt) => {
                const userAnswer = matchPairs[String(opt.id)];
                const isCorrect = userAnswer === opt.match_text;
                return (
                  <div key={opt.id} className={`play-match-row revealed${isCorrect ? ' correct' : ' wrong'}`}>
                    <div className="play-match-left">{opt.text}</div>
                    <span className="play-match-arrow">→</span>
                    <div className="play-match-right">
                      <div className={`play-match-answer${isCorrect ? '' : ' wrong-text'}`}>{userAnswer || '—'}</div>
                      {!isCorrect && <div className="play-match-correct">{opt.match_text}</div>}
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {qType === 'numeric' && (
            <div className="play-numeric-result">
              <div className="play-numeric-your">Ваш ответ: <strong>{numericValue || '—'}</strong></div>
              <div className="play-numeric-correct">Правильный: <strong>{question.correct_number}</strong>{question.tolerance ? ` (±${question.tolerance})` : ''}</div>
            </div>
          )}

          {myResult && (
            <div className={`play-result-box${myResult.is_correct ? ' correct' : ' wrong'}`}>
              {myResult.is_correct ? '✓ Правильно!' : '✗ Неправильно'}
              {myResult.answered && <span className="play-result-score">+{myResult.score} очков (Всего: {myResult.total_score})</span>}
            </div>
          )}
          <Lightbox src={lightboxImg} onClose={() => setLightboxImg(null)} />
        </div>
      </div>
    );
  }

  if (phase === 'finished') {
    const myEntry = leaderboard.find(e => e.member_id === member?.id);
    return (
      <div className="play-page">
        <div className="play-container">
          <h2 className="play-title">Квиз завершён!</h2>
          {myEntry && <div className="play-my-position">Ваше место: <strong>{myEntry.position}</strong> из {leaderboard.length}</div>}
          <div className="play-leaderboard">
            {leaderboard.map((e) => (
              <div key={e.position} className={`play-lb-row${e.member_id === member?.id ? ' me' : ''}`}>
                <span className="play-lb-pos">{e.position <= 3 ? ['🥇','🥈','🥉'][e.position - 1] : e.position}</span>
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

  return <div className="play-page"><div className="play-container"><div className="loading" style={{ color: 'white' }}>Загрузка...</div></div></div>;
}
