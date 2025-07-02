// Try and detect whether the user is on an Apple Safari browser, which has
// special nuances in their WebRTC video sharing support. This is intended to
// detect: iPads, iPhones, and Safari on macOS.
function isAppleWebkit() {
    const ua = navigator.userAgent;

    // By User-Agent: Apple mobiles.
    if (/iPad|iPhone|iPod/.test(ua)) {
        return true;
    }

    // Safari browser: claims to be Safari but not Chrome
    // (Google Chrome claims to be both)
    if (/Safari/i.test(ua) && !/Chrome/i.test(ua)) {
        return true;
    }

    // By (deprecated) navigator.platform.
    if (navigator.platform === 'iPad' || navigator.platform === 'iPhone' || navigator.platform === 'iPod') {
        return true;
    }

    return false;
}

export { isAppleWebkit };
