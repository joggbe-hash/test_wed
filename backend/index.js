import express from 'express';
import cors from 'cors';

const app = express();
const PORT = 3000;

// Middleware
app.use(cors()); // 允許跨域請求，讓前端 (Vite) 可以打 API 到這裡
app.use(express.json()); // 讓 Express 可以解析 JSON 格式的 Request Body

// ==========================================
// 模擬資料庫 (Mock Database)
// ==========================================
const users = [
  { id: 1, username: 'admin', password: 'password123', handle: '@admin', bio: '我是管理員' }
];

const posts = [
  { id: 1, userId: 1, author: 'admin', content: '這是一篇從 Node.js 後端 API 取得的貼文！', likes: 10, bookmarks: 2 },
  { id: 2, userId: 1, author: 'admin', content: '我們的 API 伺服器架設成功了！🚀', likes: 25, bookmarks: 5 }
];

// ==========================================
// API 路由 (Routes)
// ==========================================

// 1. 健康檢查 (用來測試伺服器有沒有活著)
// 網址: GET http://localhost:3000/api/health
app.get('/api/health', (req, res) => {
  res.json({ status: 'ok', message: '後端 API 伺服器正常運行中！' });
});

// 2. 登入 API
// 網址: POST http://localhost:3000/api/login
app.post('/api/login', (req, res) => {
  const { username, password } = req.body;
  
  if (!username) {
    return res.status(400).json({ success: false, message: '請輸入帳號' });
  }

  const inputName = username.trim().toLowerCase();

  // 尋找使用者 (不分大小寫)
  const user = users.find(u => u.username.toLowerCase() === inputName);
  
  // 驗證密碼
  if (user && user.password === password) {
    // 登入成功
    const { password: _, ...userData } = user;
    return res.json({ success: true, user: userData, token: 'fake-jwt-token' });
  } else if (!user) {
    // 如果找不到，自動生成測試帳號 (handle 也強制小寫)
    const newUser = { 
        id: Date.now(), 
        username: username.trim(), 
        handle: `@${inputName}`, 
        bio: '這是一段從後端傳來的新手簡介！' 
    };
    return res.json({ success: true, user: newUser, token: 'fake-jwt-token' });
  }

  return res.status(401).json({ success: false, message: '密碼錯誤' });
});

// 3. 取得所有貼文 API
// 網址: GET http://localhost:3000/api/posts
app.get('/api/posts', (req, res) => {
  res.json({ success: true, data: posts });
});

// ==========================================
// 啟動伺服器
// ==========================================
app.listen(PORT, () => {
  console.log(`Backend API Server is running on http://localhost:${PORT}`);
  console.log(`Test Health URL: http://localhost:${PORT}/api/health`);
});
