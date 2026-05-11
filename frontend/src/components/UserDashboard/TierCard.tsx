import React from 'react';
import { formatDate } from '../../utils/formatters';

interface TierCardProps {
  tier: number;
  tierName: string;
  expiration: string | null;
}

const TierCard: React.FC<TierCardProps> = ({ tier, tierName, expiration }) => {
  const tierColors = [
    'bg-gray-100 border-gray-300',
    'bg-blue-100 border-blue-300',
    'bg-purple-100 border-purple-300',
    'bg-pink-100 border-pink-300',
    'bg-yellow-100 border-yellow-300',
    'bg-orange-100 border-orange-300',
  ];

  const tierIcons = ['🌟', '🎯', '⚡', '🔥', '💎', '👑'];

  return (
    <div className={`${tierColors[tier]} rounded-lg shadow p-6 border-2`}>
      <h3 className="text-lg font-bold text-gray-900 mb-4">Subscription Tier</h3>

      <div className="flex items-center justify-between mb-4">
        <div>
          <p className="text-5xl">{tierIcons[tier]}</p>
        </div>
        <div className="text-right">
          <p className="text-sm text-gray-600">Current</p>
          <p className="text-3xl font-bold text-gray-900">{tierName}</p>
        </div>
      </div>

      {/* Expiration */}
      {expiration ? (
        <div className="bg-white bg-opacity-70 rounded p-3">
          <p className="text-xs text-gray-600">Expires</p>
          <p className="text-sm font-semibold text-gray-900">{formatDate(expiration)}</p>
        </div>
      ) : (
        <div className="bg-white bg-opacity-70 rounded p-3">
          <p className="text-xs text-gray-600">Status</p>
          <p className="text-sm font-semibold text-gray-900">Free Tier</p>
        </div>
      )}

      {/* Upgrade Button */}
      <a
        href="/profile?upgrade=true"
        className="w-full mt-4 px-4 py-2 bg-gradient-to-r from-blue-600 to-purple-600 text-white rounded font-semibold hover:shadow-lg transition-all"
      >
        Upgrade Tier
      </a>
    </div>
  );
};

export default TierCard;
