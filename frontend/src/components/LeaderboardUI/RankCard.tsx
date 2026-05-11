import React from 'react';
import { TIERS, COSMETIC_DISCOUNTS } from '../../utils/constants';
import { formatXP } from '../../utils/formatters';

interface RankCardProps {
  rank: number;
  username: string;
  xp: number;
  tier: number;
  quests: number;
  isHighlighted?: boolean;
  medal?: string;
}

const RankCard: React.FC<RankCardProps> = ({
  rank,
  username,
  xp,
  tier,
  quests,
  isHighlighted = false,
  medal,
}) => {
  const tierName = TIERS[tier as keyof typeof TIERS] || 'Free';
  const discount = COSMETIC_DISCOUNTS[tier as keyof typeof COSMETIC_DISCOUNTS] || 0;

  return (
    <tr className={isHighlighted ? 'bg-yellow-50' : 'hover:bg-gray-50'}>
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="flex items-center gap-2">
          <span className="text-2xl">{medal || ''}</span>
          <span className="font-bold text-gray-900">#{rank}</span>
        </div>
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-400 to-purple-500 flex items-center justify-center text-white font-bold">
            {username.charAt(0).toUpperCase()}
          </div>
          <div>
            <p className="font-semibold text-gray-900">{username}</p>
            <p className="text-xs text-gray-500">{tierName}</p>
          </div>
        </div>
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <p className="font-bold text-green-600">{formatXP(xp)}</p>
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <span className={`px-3 py-1 rounded-full text-sm font-semibold ${
          tier === 0 ? 'bg-gray-100 text-gray-800' :
          tier <= 2 ? 'bg-blue-100 text-blue-800' :
          tier <= 4 ? 'bg-purple-100 text-purple-800' :
          'bg-gold-100 text-gold-800'
        }`}>
          {tierName}
        </span>
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <p className="text-gray-900 font-medium">{quests}</p>
      </td>
    </tr>
  );
};

export default RankCard;
