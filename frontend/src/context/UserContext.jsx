import React, { createContext, useState, useContext } from 'react';

// 1. 建立 Context
const UserContext = createContext();

// 2. 建立 Provider 元件來包覆整個 App 並提供狀態
export const UserProvider = ({ children }) => {
  // 這裡存放全域的使用者資料
  const [user, setUser] = useState({
    isLoggedIn: false,
    username: '訪客',
    handle: '@guest',
    bio: '這是一段預設的個人簡介。歡迎來到這個社交平台！',
  });

  // 登入方法 (現在接收後端完整的 userData)
  const login = (userData) => {
    setUser({
      isLoggedIn: true,
      username: userData.username || '新用戶',
      handle: userData.handle || `@${(userData.username || 'newuser').toLowerCase()}`,
      bio: userData.bio || '這是一位新加入的朋友！',
    });
  };

  // 登出方法
  const logout = () => {
    setUser({
      isLoggedIn: false,
      username: '訪客',
      handle: '@guest',
      bio: '請登入以查看更多資訊。',
    });
  };

  // 更新個人資料方法
  const updateProfile = (newData) => {
    setUser(prev => ({ ...prev, ...newData }));
  };

  return (
    <UserContext.Provider value={{ user, login, logout, updateProfile }}>
      {children}
    </UserContext.Provider>
  );
};

// 3. 建立一個自訂 Hook，讓其他元件可以輕鬆取得使用者狀態
export const useUser = () => {
  const context = useContext(UserContext);
  if (!context) {
    throw new Error('useUser 必須在 UserProvider 內使用');
  }
  return context;
};
