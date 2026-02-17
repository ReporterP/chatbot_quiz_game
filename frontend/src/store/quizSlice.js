import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import * as quizzesApi from '../api/quizzes';

export const loadQuizzes = createAsyncThunk('quiz/loadAll', async (_, { rejectWithValue }) => {
  try {
    const { data } = await quizzesApi.fetchQuizzes();
    return data;
  } catch (err) {
    return rejectWithValue(err.response?.data?.error || 'Failed to load quizzes');
  }
});

export const loadQuiz = createAsyncThunk('quiz/loadOne', async (id, { rejectWithValue }) => {
  try {
    const { data } = await quizzesApi.fetchQuiz(id);
    return data;
  } catch (err) {
    return rejectWithValue(err.response?.data?.error || 'Failed to load quiz');
  }
});

const quizSlice = createSlice({
  name: 'quiz',
  initialState: {
    list: [],
    current: null,
    loading: false,
    error: null,
  },
  reducers: {
    clearCurrent(state) {
      state.current = null;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(loadQuizzes.pending, (s) => { s.loading = true; s.error = null; })
      .addCase(loadQuizzes.fulfilled, (s, a) => { s.loading = false; s.list = a.payload || []; })
      .addCase(loadQuizzes.rejected, (s, a) => { s.loading = false; s.error = a.payload; })
      .addCase(loadQuiz.pending, (s) => { s.loading = true; s.error = null; })
      .addCase(loadQuiz.fulfilled, (s, a) => { s.loading = false; s.current = a.payload; })
      .addCase(loadQuiz.rejected, (s, a) => { s.loading = false; s.error = a.payload; });
  },
});

export const { clearCurrent } = quizSlice.actions;
export default quizSlice.reducer;
