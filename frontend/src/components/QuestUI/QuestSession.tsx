import React, { useState, useEffect } from 'react';
import { useSelector } from 'react-redux';
import { useQuest } from '../../hooks/useQuest';
import { RootState } from '../../redux/store';
import { formatXP } from '../../utils/formatters';
import ObjectiveTracker from './ObjectiveTracker';

const QuestSession: React.FC = () => {
  const { activeSession, finishQuest, abandonCurrentQuest } = useQuest();
  const [elapsedTime, setElapsedTime] = useState<number>(0);
  const [showAbandonment, setShowAbandonment] = useState(false);

  useEffect(() => {
    if (!activeSession) return;

    const timer = setInterval(() => {
      setElapsedTime(prev => prev + 1);
    }, 1000);

    return () => clearInterval(timer);
  }, [activeSession]);

  if (!activeSession) {
    return (
      <div className="bg-white rounded-lg shadow p-8 text-center">
        <p className="text-gray-500">No active quest. Start one from the quest list!</p>
      </div>
    );
  }

  const completedObjectives = activeSession.objectives.filter(obj => obj.completed).length;
  const totalObjectives = activeSession.objectives.length;
  const progressPercent = (completedObjectives / totalObjectives) * 100;

  const handleComplete = async () => {
    if (completedObjectives === totalObjectives) {
      await finishQuest(activeSession.session_id);
    }
  };

  const handleAbandon = () => {
    abandonCurrentQuest(activeSession.session_id);
    setShowAbandonment(false);
  };

  return (
    <div className="space-y-6">
      {/* Active Quest Header */}
      <div className="bg-gradient-to-r from-green-600 to-green-700 text-white rounded-lg shadow p-6">
        <div className="flex justify-between items-start mb-4">
          <div>
            <h1 className="text-3xl font-bold">Active Quest</h1>
            <p className="text-green-100 mt-1">Session ID: {activeSession.session_id.substring(0, 8)}...</p>
          </div>
          <div className="text-right">
            <p className="text-sm text-green-100">Elapsed Time</p>
            <p className="text-2xl font-bold">{Math.floor(elapsedTime / 60)}:{(elapsedTime % 60).toString().padStart(2, '0')}</p>
          </div>
        </div>
      </div>

      {/* Progress Bar */}
      <div className="bg-white rounded-lg shadow p-6">
        <div className="flex justify-between items-center mb-2">
          <h3 className="text-lg font-semibold">Progress</h3>
          <span className="text-sm text-gray-600">
            {completedObjectives} / {totalObjectives} objectives complete
          </span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-3">
          <div
            className="bg-green-600 h-3 rounded-full transition-all duration-500"
            style={{ width: `${progressPercent}%` }}
          />
        </div>
      </div>

      {/* Objectives */}
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4">Objectives</h3>
        <div className="space-y-3">
          {activeSession.objectives.map(objective => (
            <ObjectiveTracker
              key={objective.id}
              objective={objective}
              sessionId={activeSession.session_id}
            />
          ))}
        </div>
      </div>

      {/* XP Display */}
      <div className="bg-white rounded-lg shadow p-6">
        <div className="flex justify-between items-center">
          <div>
            <p className="text-sm text-gray-600">Current XP Earned</p>
            <p className="text-3xl font-bold text-green-600">{formatXP(activeSession.xp_earned)}</p>
          </div>
          <div className="text-right">
            <p className="text-xs text-gray-500 mb-2">Progress towards completion</p>
            <div className="text-xl font-semibold text-gray-700">
              {Math.round(progressPercent)}%
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="bg-white rounded-lg shadow p-6 flex gap-3">
        <button
          onClick={handleComplete}
          disabled={completedObjectives !== totalObjectives}
          className={`flex-1 px-6 py-3 rounded-lg font-semibold transition-colors ${
            completedObjectives === totalObjectives
              ? 'bg-green-600 text-white hover:bg-green-700'
              : 'bg-gray-200 text-gray-500 cursor-not-allowed'
          }`}
        >
          Complete Quest
        </button>
        <button
          onClick={() => setShowAbandonment(true)}
          className="flex-1 px-6 py-3 bg-red-600 text-white rounded-lg font-semibold hover:bg-red-700 transition-colors"
        >
          Abandon
        </button>
      </div>

      {/* Abandonment Confirmation */}
      {showAbandonment && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-lg p-6 max-w-sm">
            <h3 className="text-lg font-bold text-gray-900 mb-3">Abandon Quest?</h3>
            <p className="text-gray-600 mb-6">
              You will lose all progress and earned XP for this quest. This action cannot be undone.
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setShowAbandonment(false)}
                className="flex-1 px-4 py-2 bg-gray-200 text-gray-900 rounded hover:bg-gray-300 font-semibold"
              >
                Cancel
              </button>
              <button
                onClick={handleAbandon}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 font-semibold"
              >
                Confirm Abandon
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default QuestSession;
