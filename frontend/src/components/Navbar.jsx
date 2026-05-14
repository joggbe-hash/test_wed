import React from 'react';
import { useNavigate } from 'react-router-dom';
import { useUser } from '../context/UserContext';

function Navbar() {
  const navigate = useNavigate();
  const { user } = useUser();

  return (
    <div className="navbar">
      <div className="nav-logo" onClick={() => navigate('/first')}></div>
      <div className="nav-actions">
        <div className="nav-square" onClick={() => navigate('/explore')}>
          <span className="material-symbols-outlined">explore</span>
        </div>
        <div 
          className="nav-icon" 
          onClick={() => navigate('/personal')}
          style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 'bold', fontSize: '18px' }}
          title={user?.username || '個人主頁'}
        >
          {user?.username ? user.username.charAt(0).toUpperCase() : ''}
        </div>
        <div className="nav-badge" onClick={() => navigate('/freq')}>999+</div>
      </div>
    </div>
  );
}

export default Navbar;
