let isPrivate = false;
        function togglePrivacy() {
            const iconContainer = document.getElementById('privacy-toggle');
            const btn = document.getElementById('privacy-btn');
            isPrivate = !isPrivate;
            if (isPrivate) {
                iconContainer.innerHTML = '<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 10 C8 16, 16 16, 21 10"/><line x1="12" y1="14" x2="12" y2="18"/><line x1="8" y1="13" x2="6" y2="16"/><line x1="16" y1="13" x2="18" y2="16"/><line x1="10" y1="13.5" x2="9" y2="17"/><line x1="14" y1="13.5" x2="15" y2="17"/></svg>';
                btn.innerText = '私人內容';
            } else {
                iconContainer.innerHTML = '<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>';
                btn.innerText = '公開分享';
            }
        }

        function toggleEye(element) {
            const isOpen = element.innerHTML.includes('circle');
            if (isOpen) {
                // Change to closed eye
                element.innerHTML = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 10 C8 16, 16 16, 21 10"/><line x1="12" y1="14" x2="12" y2="18"/><line x1="8" y1="13" x2="6" y2="16"/><line x1="16" y1="13" x2="18" y2="16"/><line x1="10" y1="13.5" x2="9" y2="17"/><line x1="14" y1="13.5" x2="15" y2="17"/></svg>';
            } else {
                // Change to open eye
                element.innerHTML = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>';
            }
        }

        function toggleHeart(element) {
            const isLiked = element.getAttribute('fill') !== 'none';
            if (isLiked) {
                element.setAttribute('fill', 'none');
                element.setAttribute('stroke', 'currentColor');
                element.style.color = '';
            } else {
                element.setAttribute('fill', '#e74c3c');
                element.setAttribute('stroke', '#e74c3c');
                element.style.color = '#e74c3c';
            }
        }

        function toggleBookmark(element) {
            const isBookmarked = element.getAttribute('fill') !== 'none';
            if (isBookmarked) {
                element.setAttribute('fill', 'none');
                element.setAttribute('stroke', 'currentColor');
                element.style.color = '';
            } else {
                element.setAttribute('fill', '#4A3320');
                element.setAttribute('stroke', '#4A3320');
                element.style.color = '#4A3320';
            }
        }