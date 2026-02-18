import axios from 'axios';

const BASE = (window.location.origin.includes('localhost') || window.location.origin.includes('127.0.0.1'))
  ? 'http://localhost:8080/api/v1'
  : '/api/v1';

const playApi = axios.create({ baseURL: BASE });

export const playJoin = (code, nickname, token) =>
  playApi.post('/play/join', { code, nickname, token });

export const playReconnect = (token, code) =>
  playApi.get('/play/reconnect', { params: { token, code } });

export const playAnswer = (sessionId, memberId, token, optionId) =>
  playApi.post('/play/answer', {
    session_id: sessionId,
    member_id: memberId,
    token,
    option_id: optionId,
  });

export const playGetState = (token, code) =>
  playApi.get('/play/state', { params: { token, code } });

export const playUpdateNickname = (token, roomCode, nickname) =>
  playApi.put('/play/nickname', { token, room_code: roomCode, nickname });

export const playGetMyResult = (sessionId, memberId) =>
  playApi.get('/play/my-result', { params: { session_id: sessionId, member_id: memberId } });
