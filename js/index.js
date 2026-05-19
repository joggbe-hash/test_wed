function toggleForm(target) {
            const slider = document.getElementById('slider');
            if (target === 'register') {
                slider.style.transform = 'translateX(-50%)';
            } else {
                slider.style.transform = 'translateX(0)';
            }
        }