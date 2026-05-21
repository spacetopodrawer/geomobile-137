export const formatXP = (xp) => {
    if (xp >= 1000000) {
        return `${(xp / 1000000).toFixed(1)}M`;
    }
    if (xp >= 1000) {
        return `${(xp / 1000).toFixed(1)}K`;
    }
    return xp.toString();
};
export const formatCurrency = (amount, currency = 'XAF') => {
    return `${currency} ${amount.toLocaleString('fr-FR')}`;
};
export const formatDuration = (seconds) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
        return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
};
export const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('fr-FR', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
    });
};
export const formatTime = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleTimeString('fr-FR', {
        hour: '2-digit',
        minute: '2-digit',
    });
};
export const calculateXPToNextLevel = (currentXP) => {
    const currentLevel = Math.floor(currentXP / 1000);
    const nextLevelXP = (currentLevel + 1) * 1000;
    return nextLevelXP - currentXP;
};
export const calculateLevel = (totalXP) => {
    return Math.floor(totalXP / 1000) + 1;
};
export const calculateProgress = (currentXP) => {
    const currentLevel = Math.floor(currentXP / 1000);
    const levelStartXP = currentLevel * 1000;
    const levelEndXP = (currentLevel + 1) * 1000;
    const progressXP = currentXP - levelStartXP;
    return (progressXP / 1000) * 100;
};
