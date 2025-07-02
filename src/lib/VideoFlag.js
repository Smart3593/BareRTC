// Video flag constants (sync with values in messages.go)
const VideoFlag = {
    Active: 1 << 0,
    NSFW: 1 << 1,
    Muted: 1 << 2,
    NonExplicit: 1 << 3,
    MutualRequired: 1 << 4,
    MutualOpen: 1 << 5,
    VipOnly: 1 << 6,
    Invited: 1 << 7,
    NoWatermark: 1 << 8,
};

export default VideoFlag;
