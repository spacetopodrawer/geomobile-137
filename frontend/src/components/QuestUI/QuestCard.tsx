import React from 'react';
import { useDispatch } from 'react-redux';
import { useQuest } from '../../hooks/useQuest';
import { formatXP, formatDuration } from '../../utils/formatters';
import type { Quest } from '../../services/quest';

interface QuestCardProps {
  quest: Quest;
}

const QuestCard: React.FC<QuestCardProps> = ({ quest }) => {
  const dispatch = useDispatch();
  const { startNewQuest } = useQuest();

  const difficultyColor = {
    Easy: 'bg-green-100 text-green-800',
    Normal: 'bg-blue-100 text-blue-800',
    Hard: 'bg-orange-100 text-orange-800',
    Master: 'bg-purple-100 text-purple-800',
  } as Record<string, string>;

  const handleStartQuest = () => {
    startNewQuest(quest.quest_id);
  };

  return (
    <div className="bg-white rounded-lg shadow hover:shadow-lg transition-shadow">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-600 to-blue-700 text-white p-4 rounded-t-lg">
        <h3 className="text-lg font-bold">{quest.title}</h3>
        <p className="text-sm text-blue-100 mt-1">{quest.region}</p>
      </div>

      {/* Content */}
      <div className="p-4 space-y-3">
        {/* Description */}
        <p className="text-gray-700 text-sm line-clamp-2">{quest.description}</p>

        {/* Badges */}
        <div className="flex flex-wrap gap-2">
          <span className={`px-2 py-1 rounded text-xs font-semibold ${difficultyColor[quest.difficulty as keyof typeof difficultyColor] || 'bg-gray-100 text-gray-800'}`}>
            {quest.difficulty}
          </span>
          <span className="px-2 py-1 bg-purple-100 text-purple-800 rounded text-xs font-semibold">
            {quest.type}
          </span>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 gap-3 pt-2 border-t border-gray-200">
          <div>
            <p className="text-xs text-gray-500">XP Reward</p>
            <p className="text-sm font-bold text-green-600">{formatXP(quest.xp_reward)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-500">Duration</p>
            <p className="text-sm font-bold text-gray-700">
              {formatDuration(quest.estimated_duration * 60)}
            </p>
          </div>
          <div>
            <p className="text-xs text-gray-500">Min Tier</p>
            <p className="text-sm font-bold text-gray-700">Tier {quest.min_tier}</p>
          </div>
          <div>
            <p className="text-xs text-gray-500">Min XP</p>
            <p className="text-sm font-bold text-gray-700">{formatXP(quest.min_xp)}</p>
          </div>
        </div>
      </div>

      {/* Action */}
      <div className="px-4 py-3 bg-gray-50 rounded-b-lg border-t border-gray-100">
        <button
          onClick={handleStartQuest}
          className="w-full px-4 py-2 bg-blue-600 text-white font-semibold rounded hover:bg-blue-700 transition-colors"
        >
          Start Quest
        </button>
      </div>
    </div>
  );
};

export default QuestCard;
