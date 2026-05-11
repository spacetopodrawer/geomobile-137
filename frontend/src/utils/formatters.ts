export const formatXP = (xp: number): string => {
  if (xp >= 1000000) {
    return `${(xp / 1000000).toFixed(1)}M`;
  }
  if (xp >= 1000) {
    return `${(xp / 1000).toFixed(1)}K`;
  }
  return xp.toString();
};

export const formatCurrency = (amount: number, currency: string = 'XAF'): string => {
  return `${currency} ${amount.toLocaleString('fr-FR')}`;
};

export const formatDuration = (seconds: number): string => {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
};

export const formatDate = (dateString: string): string => {
  const date = new Date(dateString);
  return date.toLocaleDateString('fr-FR', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
};

export const formatTime = (dateString: string): string => {
  const date = new Date(dateString);
  return date.toLocaleTimeString('fr-FR', {
    hour: '2-digit',
    minute: '2-digit',
  });
};

export const calculateXPToNextLevel = (currentXP: number): number => {
  const currentLevel = Math.floor(currentXP / 1000);
  const nextLevelXP = (currentLevel + 1) * 1000;
  return nextLevelXP - currentXP;
};

export const calculateLevel = (totalXP: number): number => {
  return Math.floor(totalXP / 1000) + 1;
};

export const calculateProgress = (currentXP: number): number => {
  const currentLevel = Math.floor(currentXP / 1000);
  const levelStartXP = currentLevel * 1000;
  const levelEndXP = (currentLevel + 1) * 1000;
  const progressXP = currentXP - levelStartXP;
  return (progressXP / 1000) * 100;
};
