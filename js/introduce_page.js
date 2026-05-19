const referrer = document.referrer;
        if (referrer && referrer.includes('html')) {
            document.getElementById('bg-iframe').src = referrer;
        } else {
            document.getElementById('bg-iframe').src = 'first_page.html';
        }