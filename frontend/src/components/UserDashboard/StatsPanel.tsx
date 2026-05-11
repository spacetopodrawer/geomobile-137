import React from 'react';

interface StatsPanelProps {
  globalRank: number;
  regionRank: number;
  region: string;
  completedQuests: number;
  badgesEarned: number;
}

const StatsPanel: React.FC<StatsPanelProps> = ({
  globalRank,
  regionRank,
  region,
  completedQuests,
  badgesEarned,
}) => {
  return (
    <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
      {/* Global Rank */}
      <div className="bg-white rounded-lg shadow p-4 text-center">
        <p className="text-3xl mb-2">🌍</p>
        <p className="text-xs text-gray-600">Global Rank</p>
        <p className="text-2xl font-bold text-gray-900">#{globalRank}</p>
      </div>

      {/* Region Rank */}
      <div className="bg-white rounded-lg shadow p-4 text-center">
        <p className="text-3xl mb-2">📍</p>
        <p className="text-xs text-gray-600">{region || 'Region'}</p>
        <p className="text-2xl font-bold text-gray-900">#{regionRank}</p>
      </div>

      {/* Completed Quests */}
      <div className="bg-white rounded-lg shadow p-4 text-center">
        <p className="text-3xl mb-2">✅</p>
        <p className="text-xs text-gray-600">Quests Done</p>
        <p className="text-2xl font-bold text-gray-900">{completedQuests}</p>
      </div>

      {/* Badges */}
      <div className="bg-white rounded-lg shadow p-4 text-center">
        <p className="text-3xl mb-2">⭐</p>
        <p className="text-xs text-gray-600">Badges</p>
        <p className="text-2xl font-bold text-gray-900">{badgesEarned}</p>
      </div>

      {/* Achievements */}
      <div className="bg-white rounded-lg shadow p-4 text-center">
        <p className="text-3xl mb-2">🏆</p>
        <p className="text-xs text-gray-600">Achievements</p>
        <p className="text-2xl font-bold text-gray-900">{badgesEarned * 2}</p>
      </div>
    </div>
  );
};

export default StatsPanel;
