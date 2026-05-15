import React, { useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import Layout from '../components/Layout';

function HorizontalScroller({ children, style }) {
  const sliderRef = useRef(null);
  let isDown = false;
  let startX;
  let scrollLeft;

  const handleMouseDown = (e) => {
    isDown = true;
    if (sliderRef.current) {
      sliderRef.current.style.cursor = 'grabbing';
      startX = e.pageX - sliderRef.current.offsetLeft;
      scrollLeft = sliderRef.current.scrollLeft;
    }
  };

  const handleMouseLeave = () => {
    isDown = false;
    if (sliderRef.current) sliderRef.current.style.cursor = 'grab';
  };

  const handleMouseUp = () => {
    isDown = false;
    if (sliderRef.current) sliderRef.current.style.cursor = 'grab';
  };

  const handleMouseMove = (e) => {
    if (!isDown || !sliderRef.current) return;
    e.preventDefault();
    const x = e.pageX - sliderRef.current.offsetLeft;
    const walk = (x - startX) * 1;
    sliderRef.current.scrollLeft = scrollLeft - walk;
  };

  const handleWheel = (e) => {
    if (sliderRef.current) {
      sliderRef.current.scrollLeft += e.deltaY;
    }
  };

  return (
    <div 
      className="horizontal-scroller" 
      ref={sliderRef}
      onMouseDown={handleMouseDown}
      onMouseLeave={handleMouseLeave}
      onMouseUp={handleMouseUp}
      onMouseMove={handleMouseMove}
      onWheel={handleWheel}
      style={style}
    >
      {children}
    </div>
  );
}

function ExplorePage() {
  const navigate = useNavigate();
  const rows = Array.from({ length: 3 });
  const cardsPerRow = Array.from({ length: 20 });

  return (
    <Layout sidebarType="explore" showFab={false}>
      {rows.map((_, rowIndex) => (
        <HorizontalScroller key={rowIndex} style={{ marginTop: rowIndex === 0 ? '0' : '20px' }}>
          {cardsPerRow.map((_, cardIndex) => (
            <div key={cardIndex} className="explore-card">
              <div className="explore-header">
                <div className="explore-avatar" onClick={() => navigate('/personal')}></div>
                <div className="explore-title-area">
                  <div className="explore-title">社群名稱</div>
                  <div className="explore-tags">#社群標籤 </div>
                </div>
              </div>
              <div className="explore-details">
                <div className="explore-desc">社群簡介</div>
                <div className="explore-members">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
                    <circle cx="9" cy="7" r="4"></circle>
                    <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
                    <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
                  </svg>
                  <span>555名成員</span>
                </div>
                <button className="post-action-btn-large" onClick={() => navigate('/social')}>探索</button>
              </div>
            </div>
          ))}
        </HorizontalScroller>
      ))}
    </Layout>
  );
}

export default ExplorePage;
