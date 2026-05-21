export const TIERS = {
    0: 'Free',
    1: 'Casual',
    2: 'Player',
    3: 'Expert',
    4: 'Pro',
    5: 'PRO ATELIER',
};
export const TIER_PRICES = {
    1: { monthly: 2000, quarterly: 5400, semiannual: 10200, annual: 19200 },
    2: { monthly: 5000, quarterly: 13500, semiannual: 25500, annual: 48000 },
    3: { monthly: 15000, quarterly: 40500, semiannual: 76500, annual: 144000 },
    4: { monthly: 50000, quarterly: 135000, semiannual: 255000, annual: 480000 },
    5: { monthly: 150000, quarterly: 405000, semiannual: 765000, annual: 1440000 },
};
export const COSMETIC_DISCOUNTS = {
    0: 0,
    1: 0.1,
    2: 0.2,
    3: 0.3,
    4: 0.5,
    5: 1.0,
};
export const QUEST_DIFFICULTIES = ['Easy', 'Normal', 'Hard', 'Master'];
export const QUEST_TYPES = ['Timeline', 'POI Hunt', 'Parcel Challenge', 'Building Detective', 'Temporal Glitch', 'Owner Quiz'];
export const COSMETIC_CATEGORIES = ['All Items', 'Avatars', 'Emotes', 'Borders', 'Titles', 'Effects'];
export const XP_PER_LEVEL = 1000;
export const REGIONS = [
    'Lékié',
    'Douala',
    'Yaoundé',
    'Buea',
    'Bafoussam',
    'Limbe',
    'Bamenda',
    'Kinshasa',
];
export const API_ENDPOINTS = {
    QUESTS: '/quest',
    USER: '/user',
    LEADERBOARD: '/leaderboard',
    PAYMENT: '/payment',
    COSMETICS: '/cosmetics',
};
export const WEBSOCKET_EVENTS = {
    QUEST_OBJECTIVE_COMPLETE: 'quest:objective_complete',
    USER_XP_GAINED: 'user:xp_gained',
    LEADERBOARD_RANK_UPDATED: 'leaderboard:rank_updated',
    PAYMENT_COMPLETED: 'payment:completed',
};
export const HTTP_STATUS = {
    OK: 200,
    CREATED: 201,
    BAD_REQUEST: 400,
    NOT_FOUND: 404,
    INTERNAL_ERROR: 500,
};
