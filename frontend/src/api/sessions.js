import api from './axios';

export const createSession = (quizId) =>
  api.post('/sessions', { quiz_id: quizId });

export const fetchSession = (id) =>
  api.get(`/sessions/${id}`);

export const listSessions = () =>
  api.get('/sessions');

export const revealAnswer = (id) =>
  api.post(`/sessions/${id}/reveal`);

export const nextQuestion = (id) =>
  api.post(`/sessions/${id}/next`);

export const fetchLeaderboard = (id) =>
  api.get(`/sessions/${id}/leaderboard`);

export const forceFinishSession = (id) =>
  api.post(`/sessions/${id}/finish`);
