import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useSelector } from 'react-redux';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import DashboardPage from './pages/DashboardPage';
import QuizEditPage from './pages/QuizEditPage';
import SessionPage from './pages/SessionPage';
import RoomPage from './pages/RoomPage';
import PlayPage from './pages/PlayPage';
import SettingsPage from './pages/SettingsPage';
import SessionHistoryPage from './pages/SessionHistoryPage';
import ProtectedRoute from './components/ProtectedRoute';
import './App.css';

export default function App() {
  const token = useSelector((s) => s.auth.token);

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={token ? <Navigate to="/dashboard" /> : <LoginPage />} />
        <Route path="/register" element={token ? <Navigate to="/dashboard" /> : <RegisterPage />} />
        <Route path="/play" element={<PlayPage />} />
        <Route element={<ProtectedRoute />}>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/quiz/:id" element={<QuizEditPage />} />
          <Route path="/session/:id" element={<SessionPage />} />
          <Route path="/room/:id" element={<RoomPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/history" element={<SessionHistoryPage />} />
        </Route>
        <Route path="*" element={<Navigate to={token ? '/dashboard' : '/login'} />} />
      </Routes>
    </BrowserRouter>
  );
}
