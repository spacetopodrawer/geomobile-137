import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { formatXP } from '../../utils/formatters';
import RankCard from './RankCard';

interface RankingEntry {
  rank: number;
  user_id: string;
  username: string;
  total_xp: number;
  tier_level: number;
  completed_quests: number;
}

interface RankingTableProps {
  rankings: RankingEntry[];
}

const RankingTable: React.FC<RankingTableProps> = ({ rankings }) => {
  const { userId } = useSelector((state: RootState) => state.auth);

  if (rankings.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow p-12 text-center">
        <p className="text-gray-500">No rankings available yet.</p>
      </div>
    );
  }

  // Separate current user from others
  const userRank = rankings.find(r => r.user_id === userId);
  const topRanks = rankings.slice(0, 10);

  return (
    <div className="space-y-4">
      {/* Top Rankings */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-100 border-b border-gray-200">
            <tr>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">Rank</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">Player</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">XP</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">Tier</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">Quests</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {topRanks.map((entry, idx) => (
              <RankCard
                key={entry.user_id}
                rank={entry.rank}
                username={entry.username}
                xp={entry.total_xp}
                tier={entry.tier_level}
                quests={entry.completed_quests}
                isHighlighted={entry.user_id === userId}
                medal={idx === 0 ? '🥇' : idx === 1 ? '🥈' : idx === 2 ? '🥉' : undefined}
              />
            ))}
          </tbody>
        </table>
      </div>

      {/* User's Rank (if not in top 10) */}
      {userRank && userRank.rank > 10 && (
        <div className="bg-blue-50 border-2 border-blue-400 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-blue-100 border-b border-blue-400">
              <tr>
                <th className="px-6 py-3 text-left text-sm font-semibold text-blue-900">Rank</th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-blue-900">Player</th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-blue-900">XP</th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-blue-900">Tier</th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-blue-900">Quests</th>
              </tr>
            </thead>
            <tbody>
              <RankCard
                rank={userRank.rank}
                username={userRank.username}
                xp={userRank.total_xp}
                tier={userRank.tier_level}
                quests={userRank.completed_quests}
                isHighlighted={true}
                medal="👤"
              />
            </tbody>
          </table>
        </div>
      )}

      {/* Stats Summary */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-600">Total Players</p>
          <p className="text-2xl font-bold text-gray-900">{rankings.length}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-600">Top XP</p>
          <p className="text-2xl font-bold text-green-600">{formatXP(rankings[0]?.total_xp || 0)}</p>
        </div>
        {userRank && (
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-sm text-gray-600">Your XP</p>
            <p className="text-2xl font-bold text-blue-600">{formatXP(userRank.total_xp)}</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default RankingTable;
