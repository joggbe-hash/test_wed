const sliders = document.querySelectorAll('.horizontal-scroller');
                    sliders.forEach(slider => {
                        let isDown = false;
                        let startX;
                        let scrollLeft;

                        slider.addEventListener('mousedown', (e) => {
                            isDown = true;
                            slider.style.cursor = 'grabbing';
                            startX = e.pageX - slider.offsetLeft;
                            scrollLeft = slider.scrollLeft;
                            // Prevent native drag/text selection
                            e.preventDefault();
                        });

                        window.addEventListener('mouseup', () => {
                            if(isDown) {
                                isDown = false;
                                slider.style.cursor = 'grab';
                            }
                        });

                        window.addEventListener('mousemove', (e) => {
                            if (!isDown) return;
                            e.preventDefault();
                            const x = e.pageX - slider.offsetLeft;
                            const walk = (x - startX) * 1; // 1:1 movement like mobile
                            slider.scrollLeft = scrollLeft - walk;
                        });

                        // Mouse Wheel to Scroll Horizontally
                        slider.addEventListener('wheel', (e) => {
                            e.preventDefault();
                            slider.scrollLeft += e.deltaY;
                        }, { passive: false });
                    });