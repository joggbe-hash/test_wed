import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useUser } from '../context/UserContext';

function LogIn() {
  const navigate = useNavigate();
  const { login } = useUser();
  const [isRegister, setIsRegister] = useState(false);
  const [usernameInput, setUsernameInput] = useState('');

  const toggleForm = (target) => {
    setIsRegister(target === 'register');
  };

  const handleRegister = () => {
    alert('註冊成功！請登入您的帳號。');
    toggleForm('login');
  };

  const handleLogin = async () => {
    if (!usernameInput) {
      alert('請輸入帳號！');
      return;
    }

    try {
      // 呼叫我們剛剛建立的後端 API
      const response = await fetch('http://localhost:3000/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: usernameInput, password: 'password123' }) // 暫時寫死密碼方便測試
      });
      
      const data = await response.json();
      
      if (data.success) {
        // 登入成功，將後端回傳的 user 資料存入 Context
        login(data.user);
        navigate('/first');
      } else {
        alert(data.message);
      }
    } catch (error) {
      console.error("API 連線失敗:", error);
      alert("無法連線到後端 API，請確認 backend 伺服器是否啟動");
    }
  };

  return (
    <div className="auth-container">
      <div 
        className="form-slider" 
        style={{ transform: isRegister ? 'translateX(-50%)' : 'translateX(0)' }}
      >
        {/* 登入表單 */}
        <div className="form-section">
          <div className="login-image-placeholder">
            <div className="shape-circle"></div>
            <div className="shape-triangle"></div>
          </div>

          <div className="input-group">
            <div className="input-label">帳號</div>
            <input 
              type="text" 
              className="input-field" 
              placeholder="例如: s123456789 或 您的暱稱" 
              value={usernameInput}
              onChange={(e) => setUsernameInput(e.target.value)}
            />
          </div>

          <div className="input-group">
            <div className="input-label">密碼</div>
            <input type="password" className="input-field" placeholder="輸入密碼" />
          </div>

          <div className="forgot-password"><a href="#">忘記密碼</a></div>

          <div className="login-actions">
            <button className="login-btn" onClick={() => toggleForm('register')}>註冊</button>
            <button className="login-btn" onClick={handleLogin}>登入</button>
          </div>
        </div>

        {/* 註冊表單 */}
        <div className="form-section reg-bg">
          <div className="reg-group"><div className="reg-label">輸入電子郵件</div><input type="email" className="reg-input" /></div>
          <div className="reg-group"><div class="reg-label">輸入驗證碼</div><input type="text" className="reg-input" /></div>
          <div className="reg-group"><div className="reg-label">輸入密碼</div><input type="password" className="reg-input" /></div>
          <div className="reg-group"><div className="reg-label">再次輸入密碼</div><input type="password" className="reg-input" /></div>
          <div className="reg-group"><div className="reg-label">輸入使用者名稱</div><input type="text" className="reg-input" /></div>

          <button className="reg-btn" onClick={handleRegister}>確認註冊</button>
          
          <div style={{ marginTop: '20px' }}>
            <a href="#" style={{ color: '#A2826D', textDecoration: 'none' }} onClick={(e) => { e.preventDefault(); toggleForm('login'); }}>返回登入</a>
          </div>
        </div>
      </div>
    </div>
  );
}

export default LogIn;
