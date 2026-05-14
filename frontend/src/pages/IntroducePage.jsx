import React from 'react';
import { useNavigate } from 'react-router-dom';

function IntroducePage() {
  const navigate = useNavigate();

  return (
    <div style={{ position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000 }}>
      
      {/* Background to simulate the modal effect */}
      <div 
        className="modal-backdrop" 
        style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', backgroundColor: 'rgba(10, 15, 25, 0.6)', backdropFilter: 'blur(4px)', WebkitBackdropFilter: 'blur(4px)', zIndex: 1, cursor: 'pointer' }}
        onClick={() => navigate(-1)}
      ></div>

      {/* Profile Card */}
      <div className="profile-card">
        <div className="profile-cover"></div>
        
        <div className="profile-avatar-container">
            <div className="profile-avatar-intro"></div>
        </div>

        <div className="profile-info">
            <div className="profile-username">@哈哈</div>
            <div className="profile-link">
                <svg viewBox="0 0 24 24" style={{ width: '14px', height: '14px', fill: 'currentColor' }}><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/></svg>
               haha.tww.com
            </div>

            <div className="profile-actions">
                <button className="action-btn">
                    <svg className="action-icon" viewBox="0 0 24 24"><path d="M20 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 4l-8 5-8-5V6l8 5 8-5v2z"/></svg>
                </button>
                <button className="action-btn">
                    <svg className="action-icon" viewBox="0 0 24 24"><path d="M15 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm-9-2V7H4v3H1v2h3v3h2v-3h3v-2H6zm9 4c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/></svg>
                </button>
                <button className="action-btn">
                    <svg className="action-icon" viewBox="0 0 24 24"><path d="M6 10c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm12 0c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm-6 0c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/></svg>
                </button>
            </div>
            
            <div className="expand-arrow" onClick={() => navigate(-1)}>
                <svg viewBox="0 0 24 24"><path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z"/></svg>
            </div>
        </div>
      </div>
    </div>
  );
}

export default IntroducePage;
