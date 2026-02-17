import { useState, useEffect } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { Link, useNavigate } from 'react-router-dom';
import { register, clearError } from '../store/authSlice';
import './Auth.css';

export default function RegisterPage() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [password2, setPassword2] = useState('');
  const [localErr, setLocalErr] = useState('');
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const { loading, error, token } = useSelector((s) => s.auth);

  useEffect(() => { dispatch(clearError()); }, []);
  useEffect(() => { if (token) navigate('/dashboard'); }, [token]);

  const handleSubmit = (e) => {
    e.preventDefault();
    setLocalErr('');
    if (password !== password2) { setLocalErr('Пароли не совпадают'); return; }
    if (password.length < 6) { setLocalErr('Пароль минимум 6 символов'); return; }
    if (username.length < 3) { setLocalErr('Логин минимум 3 символа'); return; }
    dispatch(register({ username, password }));
  };

  const displayError = localErr || error;

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1>Quiz Game</h1>
        <p className="subtitle">Создайте аккаунт ведущего</p>

        {displayError && <div className="error-msg">{displayError}</div>}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Логин</label>
            <input type="text" value={username} onChange={(e) => setUsername(e.target.value)} placeholder="Минимум 3 символа" required />
          </div>
          <div className="form-group">
            <label>Пароль</label>
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Минимум 6 символов" required />
          </div>
          <div className="form-group">
            <label>Повторите пароль</label>
            <input type="password" value={password2} onChange={(e) => setPassword2(e.target.value)} placeholder="Повторите пароль" required />
          </div>
          <button type="submit" className="btn btn-primary" disabled={loading}>
            {loading ? 'Регистрация...' : 'Зарегистрироваться'}
          </button>
        </form>

        <div className="auth-footer">
          Уже есть аккаунт? <Link to="/login">Войти</Link>
        </div>
      </div>
    </div>
  );
}
