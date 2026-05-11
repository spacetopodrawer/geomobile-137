import React, { useEffect } from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { useLeaderboard } from '../../hooks/useLeaderboard';
import { REGIONS } from '../../utils/constants';
import Loading from '../Common/Loading';
import RankingTable from './RankingTable';

const LeaderboardTabs: React.FC = () => {
  const { currentLeaderboard, scope, region, loading, lastUpdated, fetchLeaderboard, changeScopeAction, changeRegionAction } = useLeaderboard();
  const { global_rank } = useSelector((state: RootState) => state.user);

  useEffect(() => {
    fetchLeaderboard(scope, region);
  }, [scope, region]);

  if (loading) return <Loading message="Loading leaderboard..." />;

  return (
    <div className="space-y-6 py-8">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-600 to-cyan-600 text-white rounded-lg shadow p-6">
        <h1 className="text-3xl font-bold mb-2">Leaderboard</h1>
        <p className="text-blue-100">Compete with players worldwide</p>
      </div>

      {/* User Rank Highlight */}
      <div className="bg-gradient-to-r from-yellow-400 to-orange-400 rounded-lg shadow p-6 text-white">
        <div className="flex justify-between items-center">
          <div>
            <p className="text-sm opacity-90">Your Rank</p>
            <p className="text-4xl font-bold">#{global_rank}</p>
          </div>
          <div className="text-5xl">🏆</div>
        </div>
      </div>

      {/* Scope Tabs */}
      <div className="bg-white rounded-lg shadow">
        <div className="flex flex-wrap border-b border-gray-200">
          {(['global', 'regional', 'weekly'] as const).map(s => (
            <button
              key={s}
              onClick={() => changeScopeAction(s)}
              className={`px-6 py-4 font-medium border-b-2 transition-colors capitalize ${
                scope === s
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-600 hover:text-gray-900'
              }`}
            >
              {s === 'regional' ? 'Regional' : s === 'weekly' ? 'Weekly' : 'Global'} Rankings
            </button>
          ))}
        </div>

        {/* Regional Filter */}
        {scope === 'regional' && (
          <div className="px-6 py-4 bg-gray-50 border-b border-gray-200">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Select Region
            </label>
            <select
              value={region}
              onChange={(e) => changeRegionAction(e.target.value)}
              className="w-full md:w-48 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500"
            >
              <option value="">All Regions</option>
              {REGIONS.map(r => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
          </div>
        )}
      </div>

      {/* Last Updated */}
      {lastUpdated && (
        <p className="text-sm text-gray-600 text-center">
          Last updated: {new Date(lastUpdated).toLocaleTimeString()}
        </p>
      )}

      {/* Leaderboard Table */}
      <RankingTable rankings={currentLeaderboard} />
    </div>
  );
};

export default LeaderboardTabs;
