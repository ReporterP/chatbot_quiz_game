import api from './axios';

export const fetchQuizzes = () =>
  api.get('/quizzes');

export const fetchQuiz = (id) =>
  api.get(`/quizzes/${id}`);

export const createQuiz = (title) =>
  api.post('/quizzes', { title });

export const updateQuiz = (id, title) =>
  api.put(`/quizzes/${id}`, { title });

export const deleteQuiz = (id) =>
  api.delete(`/quizzes/${id}`);

export const createCategory = (quizId, title) =>
  api.post(`/quizzes/${quizId}/categories`, { title });

export const updateCategory = (catId, title) =>
  api.put(`/categories/${catId}`, { title });

export const deleteCategory = (catId) =>
  api.delete(`/categories/${catId}`);

export const reorderQuiz = (quizId, data) =>
  api.put(`/quizzes/${quizId}/reorder`, data);

export const createQuestion = (quizId, data) =>
  api.post(`/quizzes/${quizId}/questions`, data);

export const updateQuestion = (questionId, data) =>
  api.put(`/questions/${questionId}`, data);

export const deleteQuestion = (questionId) =>
  api.delete(`/questions/${questionId}`);

export const uploadImage = (file) => {
  const form = new FormData();
  form.append('file', file);
  return api.post('/upload', form, { headers: { 'Content-Type': 'multipart/form-data' } });
};

export const addQuestionImage = (questionId, url) =>
  api.post(`/questions/${questionId}/images`, { url });

export const deleteQuestionImage = (imageId) =>
  api.delete(`/images/${imageId}`);

export const exportQuiz = (quizId, format = 'json') =>
  api.get(`/quizzes/${quizId}/export?format=${format}`, { responseType: 'blob' });

export const importQuiz = (quizId, file) => {
  const form = new FormData();
  form.append('file', file);
  return api.post(`/quizzes/${quizId}/import`, form, { headers: { 'Content-Type': 'multipart/form-data' } });
};
