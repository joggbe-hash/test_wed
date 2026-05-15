import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Layout from '../components/Layout';
import { useUser } from '../context/UserContext';

function PersonalPage() {
  const { user, logout } = useUser();
  const navigate = useNavigate();
  const [eyes, setEyes] = useState([true, true]);
  const [hearts, setHearts] = useState([false]);
  const [bookmarks, setBookmarks] = useState([false, false]);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const toggleEye = (index) => {
    const newEyes = [...eyes];
    newEyes[index] = !newEyes[index];
    setEyes(newEyes);
  };

  const toggleHeart = (index) => {
    const newHearts = [...hearts];
    newHearts[index] = !newHearts[index];
    setHearts(newHearts);
  };

  const toggleBookmark = (index) => {
    const newBookmarks = [...bookmarks];
    newBookmarks[index] = !newBookmarks[index];
    setBookmarks(newBookmarks);
  };

  return (
    <Layout sidebarType="personal">
      {/* Post 1: Profile details */}
      <div className="post-card">
        <div className="post-avatar"></div>
        <div className="post-body">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '15px' }}>
            <div style={{ fontSize: '20px', fontFamily: 'monospace', color: '#333', fontWeight: 'bold' }}>{user.handle}</div>
            <button 
              onClick={handleLogout}
              style={{ padding: '4px 12px', fontSize: '12px', borderRadius: '20px', border: '1px solid #ddd', backgroundColor: '#fff', cursor: 'pointer' }}
            >
              登出
            </button>
          </div>
          <div style={{ fontSize: '16px', color: '#333', lineHeight: '1.6', marginBottom: '30px' }}>
            {user.bio}
          </div>
          <div style={{ display: 'flex', gap: '15px' }}>
            <button className="profile-btn">編輯個人檔案</button>
            <button className="profile-btn">分享個人檔案</button>
          </div>
        </div>
      </div>

      {/* Post 2: Private Text */}
      <div className="post-card">
        <div className="post-avatar"></div>
        <div className="post-body">
          <div style={{ fontSize: '16px', color: '#333', lineHeight: '1.6', marginBottom: '20px' }}>
            個人內容
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderTop: '1px solid #EEEEEE', paddingTop: '15px', color: '#7A7A7A' }}>
            <div style={{ display: 'flex', gap: '15px' }}>
              <div onClick={() => toggleEye(0)} style={{ cursor: 'pointer', display: 'flex', alignItems: 'center' }}>
                {eyes[0] ? (
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>
                ) : (
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M3 10 C8 16, 16 16, 21 10" /><line x1="12" y1="14" x2="12" y2="18" /><line x1="8" y1="13" x2="6" y2="16" /><line x1="16" y1="13" x2="18" y2="16" /><line x1="10" y1="13.5" x2="9" y2="17" /><line x1="14" y1="13.5" x2="15" y2="17" /></svg>
                )}
              </div>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>
            </div>
            <div style={{ display: 'flex', gap: '15px' }}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>
              <svg onClick={() => toggleBookmark(0)} style={{ cursor: 'pointer', color: bookmarks[0] ? '#4A3320' : '' }} width="20" height="20" viewBox="0 0 24 24" fill={bookmarks[0] ? '#4A3320' : 'none'} stroke={bookmarks[0] ? '#4A3320' : 'currentColor'} strokeWidth="2"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"></path></svg>
            </div>
          </div>
        </div>
      </div>

      {/* Post 3: Public Text */}
      <div className="post-card">
        <div className="post-avatar"></div>
        <div className="post-body">
          <div style={{ fontSize: '16px', color: '#333', lineHeight: '1.6', marginBottom: '20px' }}>
            發佈內容
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderTop: '1px solid #EEEEEE', paddingTop: '15px', color: '#7A7A7A' }}>
            <div style={{ display: 'flex', gap: '15px' }}>
              <div onClick={() => toggleEye(1)} style={{ cursor: 'pointer', display: 'flex', alignItems: 'center' }}>
                {eyes[1] ? (
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>
                ) : (
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M3 10 C8 16, 16 16, 21 10" /><line x1="12" y1="14" x2="12" y2="18" /><line x1="8" y1="13" x2="6" y2="16" /><line x1="16" y1="13" x2="18" y2="16" /><line x1="10" y1="13.5" x2="9" y2="17" /><line x1="14" y1="13.5" x2="15" y2="17" /></svg>
                )}
              </div>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>
            </div>
            <div style={{ display: 'flex', gap: '15px' }}>
              <svg onClick={() => toggleHeart(0)} style={{ cursor: 'pointer', color: hearts[0] ? '#e74c3c' : '', transform: hearts[0] ? 'scale(1.1)' : 'scale(1)', transition: 'transform 0.2s' }} width="20" height="20" viewBox="0 0 24 24" fill={hearts[0] ? '#e74c3c' : 'none'} stroke={hearts[0] ? '#e74c3c' : 'currentColor'} strokeWidth="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path></svg>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg>
              <svg onClick={() => toggleBookmark(1)} style={{ cursor: 'pointer', color: bookmarks[1] ? '#4A3320' : '', transform: bookmarks[1] ? 'scale(1.1)' : 'scale(1)', transition: 'transform 0.2s' }} width="20" height="20" viewBox="0 0 24 24" fill={bookmarks[1] ? '#4A3320' : 'none'} stroke={bookmarks[1] ? '#4A3320' : 'currentColor'} strokeWidth="2"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"></path></svg>
            </div>
          </div>
        </div>
      </div>

      {/* Post 4: Timer Post */}
      <div className="post-card">
        <div className="post-avatar black"></div>
        <div className="post-body">
          <div style={{ fontSize: '32px', fontWeight: 'bold', fontFamily: 'monospace', letterSpacing: '2px', marginBottom: '20px' }}>02:00:58</div>
          <div style={{ width: '100%', height: '250px', backgroundColor: '#A0A0A0', borderRadius: '4px' }}></div>
        </div>
      </div>
    </Layout>
  );
}

export default PersonalPage;
