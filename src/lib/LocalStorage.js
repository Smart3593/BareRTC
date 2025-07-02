// All the distinct localStorage keys used.
const keys = {
    'fontSizeClass': String,        // Text magnification
    'videoScale': String,           // Video magnification (CSS classnames)
    'messageStyle': String,         // Message display style (cards, compact, etc.)
    'imageDisplaySetting': String,  // Show/hide/expand image preference
    'scrollback': Number,           // Scrollback buffer (int)
    'preferredDeviceNames': Object, // Webcam/mic device names (object, keys video,audio)
    'whoSort': String,              // user's preferred sort order for the Who List
    'theme': String,                // light, dark, or auto theme

    // Webcam settings (booleans)
    'videoMutual': Boolean,
    'videoMutualOpen': Boolean,
    'videoAutoMute': Boolean,
    'videoVipOnly': Boolean,
    'videoExplicit': Boolean,  // whether the user turns explicit on by default
    'videoNonExplicit': Boolean,  // user prefers not to see explicit
    'rememberExpresslyClosed': Boolean,
    'autoMuteWebcams': Boolean, // automatically mute other peoples' webcam audio feeds
    'videoAutoShare': Boolean, // automatically share your webcam on page load
    'videoNoWatermark': Boolean, // user does not want watermark protection

    // Booleans
    'usePolling': Boolean,  // use the polling API instead of WebSocket
    'joinMessages': Boolean,
    'exitMessages': Boolean,
    'watchNotif': Boolean,
    'muteSounds': Boolean,
    'closeDMs': Boolean,      // close unsolicited DMs
    'debug': Boolean,        // Debug views enabled (admin only)

    // Don't Show Again on NSFW modals.
    'skip-nsfw-modal': Boolean,
}

// UserSettings centralizes browser settings for the chat room.
class UserSettings {
    constructor() {
        // Recall stored settings. Only set the keys that were
        // found in localStorage on page load.
        for (let key of Object.keys(keys)) {
            if (localStorage[key] != undefined) {
                try {
                    this[key] = JSON.parse(localStorage[key]);
                } catch(e) {
                    console.error(`LocalStorage: parsing key ${key}: ${e}`);
                    delete(this[key]);
                }
            }
        }

        console.log("LocalStorage: Loaded settings", this);
    }

    // Return all of the current settings where the user had actually
    // left a preference on them (was in localStorage).
    getSettings() {
        let result = {};
        for (let key of Object.keys(keys)) {
            if (this[key] != undefined) {
                result[key] = this[key];
            }
        }
        return result;
    }

    // Get a value from localStorage, if set.
    get(key) {
        return this[key];
    }

    // Generic setter.
    set(key, value) {
        if (keys[key] == undefined) {
            throw `${key}: not a supported localStorage setting`;
        }

        localStorage[key] = JSON.stringify(value);
        this[key] = value;
    }
}

// LocalStorage is a global singleton to access and update user settings.
const LocalStorage = new UserSettings();
export default LocalStorage;
