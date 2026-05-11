import React from 'react';
import { useQuest } from '../../hooks/useQuest';
import { formatXP } from '../../utils/formatters';

interface Objective {
  id: string;
  description: string;
  completed: boolean;
  xp_reward: number;
}

interface ObjectiveTrackerProps {
  objective: Objective;
  sessionId: string;
}

const ObjectiveTracker: React.FC<ObjectiveTrackerProps> = ({ objective, sessionId }) => {
  const { completeObjective } = useQuest();

  const handleComplete = async () => {
    if (!objective.completed) {
      await completeObjective(sessionId, objective.id);
    }
  };

  return (
    <div className="flex items-start gap-4 p-4 border border-gray-200 rounded-lg hover:border-blue-300 transition-colors">
      {/* Checkbox */}
      <button
        onClick={handleComplete}
        className={`flex-shrink-0 mt-1 w-6 h-6 rounded border-2 transition-all ${
          objective.completed
            ? 'bg-green-600 border-green-600'
            : 'border-gray-300 hover:border-blue-400'
        }`}
      >
        {objective.completed && (
          <svg className="w-4 h-4 text-white" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
          </svg>
        )}
      </button>

      {/* Content */}
      <div className="flex-1">
        <p className={`font-medium transition-all ${objective.completed ? 'text-gray-400 line-through' : 'text-gray-900'}`}>
          {objective.description}
        </p>
      </div>

      {/* XP Reward */}
      <div className="flex-shrink-0 text-right">
        <p className={`text-sm font-semibold ${objective.completed ? 'text-green-600' : 'text-gray-600'}`}>
          +{formatXP(objective.xp_reward)} XP
        </p>
      </div>
    </div>
  );
};

export default ObjectiveTracker;
