import React from 'react';
import { formatXP } from '../../utils/formatters';

interface ProgressCardProps {
  level: number;
  currentXP: number;
  nextLevelXP: number;
  progressPercent: number;
}

const ProgressCard: React.FC<ProgressCardProps> = ({
  level,
  currentXP,
  nextLevelXP,
  progressPercent,
}) => {
  const xpToNext = nextLevelXP - (currentXP % 1000);

  return (
    <div className="bg-gradient-to-br from-blue-50 to-cyan-50 rounded-lg shadow p-6 border-2 border-blue-200">
      <h3 className="text-lg font-bold text-gray-900 mb-4">Level Progress</h3>

      <div className="flex items-center justify-between mb-4">
        <div>
          <p className="text-4xl font-bold text-blue-600">{level}</p>
          <p className="text-sm text-gray-600">Current Level</p>
        </div>
        <div className="text-right">
          <p className="text-sm text-gray-600">Next Level</p>
          <p className="text-2xl font-bold text-gray-900">{level + 1}</p>
        </div>
      </div>

      {/* Progress Bar */}
      <div className="mb-4">
        <div className="w-full bg-gray-200 rounded-full h-4 overflow-hidden">
          <div
            className="bg-gradient-to-r from-blue-500 to-cyan-500 h-4 transition-all duration-500"
            style={{ width: `${progressPercent}%` }}
          />
        </div>
        <p className="text-xs text-gray-600 mt-2">
          {formatXP(currentXP % 1000)} / 1,000 XP to next level
        </p>
      </div>

      {/* Total XP */}
      <div className="bg-white rounded p-3">
        <p className="text-xs text-gray-600">Total XP Earned</p>
        <p className="text-2xl font-bold text-blue-600">{formatXP(currentXP)}</p>
      </div>
    </div>
  );
};

export default ProgressCard;
