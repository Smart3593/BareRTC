import QrCode from 'qrcodejs';

// WatermarkImage outputs a QR code containing watermark data about the current user.
//
// To help detect when someone has screen recorded and shared it, and being able to know who/when/etc.
function WatermarkImage(username) {
    let now = new Date();
    let dateString = [
        now.getFullYear(),
        ('0' + (now.getMonth()+1)).slice(-2),
        ('0' + (now.getDate())).slice(-2),
    ].join('-');

    let fields = [
        window.location.hostname,
        username,
        dateString,
    ].join(' ');

    console.error("watermark message:", fields);

    const matrix = QrCode.generate(fields);
    const uri = QrCode.render('svg-uri', matrix);
    return uri;
}

export default WatermarkImage;
