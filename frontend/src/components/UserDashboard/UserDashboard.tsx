import React, { useEffect } from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { useUser } from '../../hooks/useUser';
import { TIERS } from '../../utils/constants';
import { formatXP, calculateProgress, formatDate } from '../../utils/formatters';
import ProgressCard from './ProgressCard';
import TierCard from './TierCard';
import StatsPanel from './StatsPanel';

const UserDashboard: React.FC = () => {
  const { fetchUserProgress } = useUser();
  const user = useSelector((state: RootState) => state.user);

  useEffect(() => {
    fetchUserProgress();
  }, []);

  const tierName = TIERS[user.tier_level as keyof typeof TIERS] || 'Free';
  const progressPercent = calculateProgress(user.total_xp);

  return (
    <div className="space-y-6 py-8">
      {/* Header with User Info */}
      <div className="bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg shadow p-6">
        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-3xl font-bold mb-2">Profile</h1>
            <p className="text-indigo-100">Welcome back!</p>
          </div>
          <div className="text-right">
            <div className="w-16 h-16 rounded-full bg-white bg-opacity-20 flex items-center justify-center text-3xl">
              👤
            </div>
          </div>
        </div>
      </div>

      {/* Top Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Progress */}
        <ProgressCard
          level={user.level}
          currentXP={user.total_xp}
          nextLevelXP={(user.level * 1000)}
          progressPercent={progressPercent}
        />

        {/* Tier */}
        <TierCard
          tier={user.tier_level}
          tierName={tierName}
          expiration={user.tier_expiration}
        />
      </div>

      {/* Main Stats Panel */}
      <StatsPanel
        globalRank={user.global_rank}
        regionRank={user.region_rank}
        region={user.region}
        completedQuests={user.completed_quests}
        badgesEarned={user.badges_earned.length}
      />

      {/* Badges Section */}
      {user.badges_earned.length > 0 && (
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-lg font-bold text-gray-900 mb-4">Badges Earned</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {user.badges_earned.map((badge, idx) => (
              <div key={idx} className="text-center p-4 bg-yellow-50 rounded-lg border-2 border-yellow-200">
                <p className="text-3xl mb-2">⭐</p>
                <p className="text-sm font-semibold text-gray-900">{badge}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <a
          href="/quests"
          className="bg-blue-600 text-white rounded-lg shadow p-4 hover:bg-blue-700 transition-colors"
        >
          <p className="font-bold">Start a Quest</p>
          <p className="text-sm text-blue-100">Continue your adventure</p>
        </a>
        <a
          href="/shop"
          className="bg-purple-600 text-white rounded-lg shadow p-4 hover:bg-purple-700 transition-colors"
        >
          <p className="font-bold">Cosmetics Shop</p>
          <p className="text-sm text-purple-100">Upgrade your appearance</p>
        </a>
      </div>
    </div>
  );
};

export default UserDashboard;
