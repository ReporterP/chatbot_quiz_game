import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import * as sessionsApi from '../api/sessions';

export const loadSession = createAsyncThunk('session/load', async (id, { rejectWithValue }) => {
  try {
    const { data } = await sessionsApi.fetchSession(id);
    return data;
  } catch (err) {
    return rejectWithValue(err.response?.data?.error || 'Failed to load session');
  }
});

export const doReveal = createAsyncThunk('session/reveal', async (id, { rejectWithValue }) => {
  try {
    const { data } = await sessionsApi.revealAnswer(id);
    return data;
  } catch (err) {
    return rejectWithValue(err.response?.data?.error || 'Failed to reveal');
  }
});

export const doNext = createAsyncThunk('session/next', async (id, { rejectWithValue }) => {
  try {
    const { data } = await sessionsApi.nextQuestion(id);
    return data;
  } catch (err) {
    return rejectWithValue(err.response?.data?.error || 'Failed to advance');
  }
});

export const loadLeaderboard = createAsyncThunk('session/leaderboard', async (id, { rejectWithValue }) => {
  try {
    const { data } = await sessionsApi.fetchLeaderboard(id);
    return data;
  } catch (err) {
    return rejectWithValue(err.response?.data?.error || 'Failed to load leaderboard');
  }
});

const sessionSlice = createSlice({
  name: 'session',
  initialState: {
    data: null,
    leaderboard: [],
    loading: false,
    error: null,
  },
  reducers: {
    clearSession(state) {
      state.data = null;
      state.leaderboard = [];
    },
    setSessionData(state, action) {
      state.data = action.payload;
    },
  },
  extraReducers: (builder) => {
    const applySession = (s, a) => { s.loading = false; s.data = a.payload; };
    builder
      .addCase(loadSession.pending, (s) => { s.loading = true; })
      .addCase(loadSession.fulfilled, applySession)
      .addCase(loadSession.rejected, (s, a) => { s.loading = false; s.error = a.payload; })
      .addCase(doReveal.fulfilled, applySession)
      .addCase(doNext.fulfilled, applySession)
      .addCase(loadLeaderboard.fulfilled, (s, a) => { s.leaderboard = a.payload || []; });
  },
});

export const { clearSession, setSessionData } = sessionSlice.actions;
export default sessionSlice.reducer;
