import { useDispatch, useSelector } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import { logout } from '../store/authSlice';
import './Header.css';

export default function Header() {
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const username = useSelector((s) => s.auth.username);

  const handleLogout = () => {
    dispatch(logout());
    navigate('/login');
  };

  return (
    <header className="header">
      <div className="header-logo" onClick={() => navigate('/dashboard')} style={{ cursor: 'pointer' }}>
        Quiz Game
      </div>
      <div className="header-right">
        <button className="btn-settings" onClick={() => navigate('/settings')}>Настройки</button>
        <span className="header-user">{username}</span>
        <button className="btn-logout" onClick={handleLogout}>Выйти</button>
      </div>
    </header>
  );
}
