import React from 'react';
import Navbar from './Navbar';
import Sidebar from './Sidebar';

function Layout({ children, showSidebar = true, showFab = true, sidebarType = 'first', customStyles = {}, feedStyles = {}, fabIcon = '+' }) {
  return (
    <div className="app-container" style={customStyles}>
      <Navbar />
      <div className="main-layout">
        {showSidebar && <Sidebar type={sidebarType} />}
        <div className="feed-content" style={{ ...(sidebarType === 'explore' ? { maxWidth: '1400px', overflowX: 'hidden' } : {}), ...feedStyles }}>
          {children}
        </div>
      </div>
      {showFab && <div className="fab" style={{ paddingBottom: fabIcon === '+' ? '8px' : '0' }}>{fabIcon}</div>}
    </div>
  );
}

export default Layout;
