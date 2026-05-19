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
            const isFilled = element.getAttribute('fill') === '#e74c3c';
            if (isFilled) {
                element.setAttribute('fill', 'none');
                element.setAttribute('stroke', 'currentColor');
                element.style.transform = "scale(1)";
            } else {
                element.setAttribute('fill', '#e74c3c');
                element.setAttribute('stroke', '#e74c3c');
                element.style.transform = "scale(1.1)";
            }
        }

        function toggleBookmark(element) {
            const isFilled = element.getAttribute('fill') === '#4A3320';
            if (isFilled) {
                element.setAttribute('fill', 'none');
                element.style.transform = "scale(1)";
            } else {
                element.setAttribute('fill', '#4A3320');
                element.style.transform = "scale(1.1)";
            }
        }