import React from 'react'
import { BrowserRouter as Router, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import FirstPage from './pages/FirstPage'
import ExplorePage from './pages/ExplorePage'
import PersonalPage from './pages/PersonalPage'
import FreqPage from './pages/FreqPage'
import IntroducePage from './pages/IntroducePage'
import SocialPage from './pages/SocialPage'
import LogIn from './pages/LogIn'

import { UserProvider } from './context/UserContext'

function AppContent() {
  const location = useLocation();
  const background = location.state && location.state.background;

  return (
    <>
      <Routes location={background || location}>
        <Route path="/" element={<Navigate to="/login" replace />} />
        <Route path="/login" element={<LogIn />} />
        <Route path="/first" element={<FirstPage />} />
        <Route path="/explore" element={<ExplorePage />} />
        <Route path="/personal" element={<PersonalPage />} />
        <Route path="/freq" element={<FreqPage />} />
        <Route path="/social" element={<SocialPage />} />
        <Route path="/introduce" element={<FirstPage />} />
      </Routes>

      {location.pathname === '/introduce' && <IntroducePage />}
    </>
  );
}

function App() {
  return (
    <UserProvider>
      <Router>
        <AppContent />
      </Router>
    </UserProvider>
  )
}

export default App
