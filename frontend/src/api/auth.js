import api from './axios';

export const registerHost = (username, password) =>
  api.post('/auth/register', { username, password });

export const loginHost = (username, password) =>
  api.post('/auth/login', { username, password });
