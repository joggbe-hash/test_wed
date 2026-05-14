import React, { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Layout from '../components/Layout';

function SocialPage() {
  const navigate = useNavigate();
  const [visiblePosts, setVisiblePosts] = useState({});
  const observerRef = useRef(null);

  const posts = Array.from({ length: 12 }); // 4 posts duplicated 3 times = 12 total

  useEffect(() => {
    observerRef.current = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          setVisiblePosts((prev) => ({ ...prev, [entry.target.dataset.index]: true }));
        }
      });
    }, {
      threshold: 0.15
    });

    return () => {
      if (observerRef.current) observerRef.current.disconnect();
    };
  }, []);

  const observePost = (el, index) => {
    if (el && observerRef.current) {
      el.dataset.index = index;
      observerRef.current.observe(el);
    }
  };

  return (
    <Layout sidebarType="social" feedStyles={{ maxWidth: '1200px' }} fabIcon="+">
      <div className="theme-banner">
        <div className="theme-banner-header">
          <div style={{ display: 'flex', gap: '15px', alignItems: 'center' }}>
            <div style={{ width: '80px', height: '28px', backgroundColor: 'rgba(255,255,255,0.2)', backdropFilter: 'blur(4px)', borderRadius: '14px', display: 'flex', justifyContent: 'center', alignItems: 'center', fontSize: '12px' }}>🔥 趨勢</div>
            <div style={{ fontSize: '14px', color: 'rgba(255,255,255,0.8)' }}>2,451 則討論</div>
          </div>
          <div style={{ display: 'flex', gap: '15px', alignItems: 'center' }}>
            <div style={{ padding: '8px 20px', backgroundColor: '#FFFFFF', color: '#4A3320', borderRadius: '20px', fontWeight: 'bold', fontSize: '14px', cursor: 'pointer', boxShadow: '0 4px 10px rgba(0,0,0,0.1)' }}>進入話題</div>
            <div style={{ width: '35px', height: '35px', backgroundColor: 'rgba(255,255,255,0.2)', borderRadius: '50%', display: 'flex', justifyContent: 'center', alignItems: 'center', backdropFilter: 'blur(4px)', cursor: 'pointer' }}>...</div>
          </div>
        </div>
        <div className="theme-banner-bottom">
          <div style={{ fontSize: '32px', fontWeight: '800', textShadow: '0 2px 5px rgba(0,0,0,0.2)', letterSpacing: '1px' }}>今日熱門討論區</div>
          <div style={{ fontSize: '16px', color: 'rgba(255,255,255,0.9)', lineHeight: '1.5' }}>一起探索大家都在聊些什麼吧！<br/>這裡有最新的社群趨勢與精彩話題。</div>
        </div>
        <div style={{ position: 'absolute', bottom: '-30px', right: '40px', width: '180px', height: '180px', border: '25px solid rgba(255,255,255,0.08)', borderRadius: '50%' }}></div>
      </div>

      {posts.map((_, index) => {
        const type = index % 4; // 0, 1, 2, 3 pattern

        return (
          <div 
            key={index} 
            ref={(el) => observePost(el, index)}
            className={`post-card ${visiblePosts[index] ? 'show' : 'hidden'}`}
          >
            <div className="post-avatar" onClick={() => navigate('/personal')}></div>
            {type === 0 && (
              <div className="post-body">
                <div className="skeleton-line short"></div>
                <div style={{ display: 'flex', gap: '10px', marginBottom: '20px', marginTop: '20px' }}>
                  <div className="skeleton-line" style={{ width: '40px', marginBottom: '0' }}></div>
                  <div className="skeleton-line" style={{ width: '40px', marginBottom: '0' }}></div>
                </div>
                <button className="post-action-btn">分享</button>
              </div>
            )}
            {type === 1 && (
              <div style={{ display: 'flex', flexDirection: 'column', width: '100%' }}>
                <div className="post-body">
                  <div className="skeleton-line short"></div>
                  <div className="skeleton-line long"></div>
                  <div className="skeleton-line long"></div>
                </div>
                <div style={{ display: 'flex', gap: '10px', marginTop: '15px', paddingLeft: '5px' }}>
                  <div style={{ width: '25px', height: '25px', backgroundColor: '#CCCCCC', borderRadius: '4px' }}></div>
                  <div style={{ width: '25px', height: '25px', backgroundColor: '#CCCCCC', borderRadius: '4px' }}></div>
                  <div style={{ width: '25px', height: '25px', backgroundColor: '#CCCCCC', borderRadius: '4px' }}></div>
                </div>
              </div>
            )}
            {type === 2 && (
              <div className="post-body">
                <div style={{ width: '100%', height: '350px', backgroundColor: 'var(--hover-bg)', borderRadius: '8px' }}></div>
              </div>
            )}
            {type === 3 && (
               <div className="post-body">
                 <div className="skeleton-line short"></div>
                 <button className="post-action-btn">分享</button>
               </div>
            )}
          </div>
        );
      })}
    </Layout>
  );
}

export default SocialPage;
