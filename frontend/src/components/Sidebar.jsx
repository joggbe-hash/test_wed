import React from 'react';
import { useNavigate, useLocation } from 'react-router-dom';

function Sidebar({ type = 'first' }) {
  const navigate = useNavigate();
  const location = useLocation();

  if (type === 'explore') {
    const icons = Array.from({ length: 20 });
    return (
      <div className="sidebar">
        <div className="sidebar-search">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#7A7A7A" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ marginRight: '10px' }}>
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
          <input type="text" className="sidebar-search-input" placeholder="" />
        </div>
        <div className="grid-icons">
          {icons.map((_, index) => (
            <div key={index} className="grid-item">
              <div className="grid-icon-circle"></div>
              <div style={{ fontSize: '14px', color: '#333', marginTop: '5px' }}>類別</div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (type === 'personal') {
    return (
      <div className="sidebar">
      </div>
    );
  }

  if (type === 'freq') {
    const cards = Array.from({ length: 3 });
    return (
      <div className="sidebar">
        {cards.map((_, i) => (
          <div key={i} className="freq-sidebar-card"></div>
        ))}
      </div>
    );
  }

  if (type === 'social') {
    const icons = Array.from({ length: 16 });
    return (
      <div className="sidebar">
        <div className="sidebar-card"></div>
        <div className="sidebar-card"></div>
        <div className="sidebar-card"></div>
        <div className="grid-icons">
          {icons.map((_, index) => (
            <div
              key={index}
              className="grid-icon-circle"
              onClick={() => navigate('/introduce', { state: { background: location } })}
            ></div>
          ))}
        </div>
      </div>
    );
  }

  // Default / First page sidebar
  const icons = Array.from({ length: 16 });
  return (
    <div className="sidebar">
      <div className="grid-icons">
        {icons.map((_, index) => (
          <div
            key={index}
            className="grid-icon-circle"
            onClick={() => navigate('/introduce', { state: { background: location } })}
          ></div>
        ))}
      </div>
      <div className="sidebar-card"></div>
      <div className="sidebar-card"></div>
      <div className="sidebar-card"></div>
    </div>
  );
}

export default Sidebar;
