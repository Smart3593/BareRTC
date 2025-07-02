// Available status options.
const StatusOptions = [
    {
        category: "Status",
        options: [
            {
                name: "online",
                label: "Active",
                emoji: "☀️",
                icon: "fa fa-clock"
            },
            {
                name: "away",
                label: "Away",
                emoji: "🕒",
                icon: "fa fa-clock"
            },
            {
                name: "brb",
                label: "Be right back",
                emoji: "⏰",
                icon: "fa fa-stopwatch-20"
            },
            {
                name: "afk",
                label: "Away from keyboard",
                emoji: "⌨️",
                icon: "fa fa-keyboard who-status-wide-icon-1"
            },
            {
                name: "lunch",
                label: "Out to lunch",
                emoji: "🍴",
                icon: "fa fa-utensils"
            },
            {
                name: "call",
                label: "On the phone",
                emoji: "📞",
                icon: "fa fa-phone-volume"
            },
            {
                name: "busy",
                label: "Working",
                emoji: "💼",
                icon: "fa fa-briefcase"
            },
            {
                name: "book",
                label: "Studying",
                emoji: "📖",
                icon: "fa fa-book"
            },
            {
                name: "gaming",
                label: "Gaming",
                emoji: "🎮",
                icon: "fa fa-gamepad who-status-wide-icon-2"
            },
            {
                name: "movie",
                label: "Watching a movie",
                emoji: "🎞️",
                icon: "fa fa-film"
            },
            {
                name: "workout",
                label: "Working out",
                emoji: "🏋️",
                icon: "fa-solid fa-dumbbell who-status-wide-icon-2",
            },
            {
                name: "travel",
                label: "Traveling",
                emoji: "✈️",
                icon: "fa fa-plane"
            },

            // Hidden/special statuses
            {
                name: "idle",
                label: "Idle",
                emoji: "🕒",
                icon: "fa-regular fa-moon",
                hidden: true
            },
            {
                name: "hidden",
                label: "Hidden",
                emoji: "🕵️",
                icon: "",
                adminOnly: true
            },
        ],
    },
    {
        category: "Mood",
        options: [
            {
                name: "chatty",
                label: "Chatty and sociable",
                emoji: "🗨️",
                icon: "fa fa-comment"
            },
            {
                name: "introverted",
                label: "Introverted and quiet",
                emoji: "🥄",
                icon: "fa fa-spoon"
            },

            // If NSFW enabled
            {
                name: "horny",
                label: "Horny",
                emoji: "🔥",
                icon: "fa fa-fire",
                nsfw: true,
            },
            {
                name: "exhibitionist",
                label: "Watch me",
                emoji: "👀",
                icon: "fa-regular fa-eye who-status-wide-icon-1",
                nsfw: true,
            }
        ]
    }
];

// Flatten the set of all status options.
const StatusFlattened = (function() {
    let result = [];
    for (let category of StatusOptions) {
        for (let option of category.options) {
            result.push(option);
        }
    }
    return result;
})();

// Hash map lookup of status by name.
const StatusByName = (function() {
    let result = {};
    for (let item of StatusFlattened) {
        result[item.name] = item;
    }
    return result;
})();

// An API surface layer of functions.
class StatusMessageController {
    // The caller configures:
    // - nsfw (bool): the BareRTC PermitNSFW setting, which controls some status options.
    // - isAdmin (func): return a boolean if the current user is operator.
    // - currentStatus (func): return the name of the user's current status.
    constructor() {
        this.nsfw = false;
        this.isAdmin = function() { return false };
        this.currentStatus = function() { return StatusFlattened[0] };
    }

    // Iterate the category <optgroup> for the Status dropdown menu.
    iterSelectOptGroups() {
        return StatusOptions;
    }

    // Iterate the <option> for a category of statuses.
    iterSelectOptions(category) {
        let current = this.currentStatus(),
            isAdmin = this.isAdmin();

        for (let group of StatusOptions) {
            if (group.category === category) {
                // Return the filtered options.
                let result = group.options.filter(option => {
                    if ((option.hidden && current !== option.name) ||
                        (option.adminOnly && !isAdmin) ||
                        (option.nsfw && !this.nsfw)) {
                        return false;
                    }
                    return true;
                });
                return result;
            }
        }

        return [];
    }

    // Get details on a status message.
    getStatus(name) {
        if (StatusByName[name] != undefined) {
            return StatusByName[name];
        }

        // Return a dummy status object.
        return {
            name: name,
            label: name,
            icon: "fa fa-clock",
            emoji: "🕒"
        };
    }

    // Offline status.
    offline() {
        return {
            name: "offline",
            label: "Offline",
            icon: "fa fa-house-circle-xmark",
            emoji: "🌜",
        }
    }
}

const StatusMessage = new StatusMessageController();
export default StatusMessage;
