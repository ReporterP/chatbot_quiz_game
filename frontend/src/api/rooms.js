import api from './axios';

export const createRoom = (mode = 'web') =>
  api.post('/rooms', { mode });

export const listActiveRooms = () =>
  api.get('/rooms');

export const fetchRoom = (id) =>
  api.get(`/rooms/${id}`);

export const closeRoom = (id) =>
  api.post(`/rooms/${id}/close`);

export const startQuizInRoom = (roomId, quizId) =>
  api.post(`/rooms/${roomId}/start`, { quiz_id: quizId });

export const roomReveal = (roomId) =>
  api.post(`/rooms/${roomId}/reveal`);

export const roomNext = (roomId) =>
  api.post(`/rooms/${roomId}/next`);

export const roomFinish = (roomId) =>
  api.post(`/rooms/${roomId}/finish`);

export const roomLeaderboard = (roomId) =>
  api.get(`/rooms/${roomId}/leaderboard`);

export const listRoomHistory = () =>
  api.get('/rooms/history');
