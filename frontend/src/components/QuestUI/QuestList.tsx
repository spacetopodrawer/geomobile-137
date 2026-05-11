import React, { useEffect, useState } from 'react';
import { useQuest } from '../../hooks/useQuest';
import { QUEST_DIFFICULTIES, QUEST_TYPES } from '../../utils/constants';
import { formatXP } from '../../utils/formatters';
import Loading from '../Common/Loading';
import QuestCard from './QuestCard';

const QuestList: React.FC = () => {
  const { availableQuests, loading, error, fetchAvailableQuests } = useQuest();
  const [filteredQuests, setFilteredQuests] = useState(availableQuests);
  const [selectedDifficulty, setSelectedDifficulty] = useState<string>('');
  const [selectedType, setSelectedType] = useState<string>('');
  const [selectedRegion, setSelectedRegion] = useState<string>('');

  useEffect(() => {
    fetchAvailableQuests(50);
  }, []);

  useEffect(() => {
    let filtered = availableQuests;

    if (selectedDifficulty) {
      filtered = filtered.filter(q => q.difficulty === selectedDifficulty);
    }
    if (selectedType) {
      filtered = filtered.filter(q => q.type === selectedType);
    }
    if (selectedRegion) {
      filtered = filtered.filter(q => q.region === selectedRegion);
    }

    setFilteredQuests(filtered);
  }, [availableQuests, selectedDifficulty, selectedType, selectedRegion]);

  if (loading) return <Loading message="Loading quests..." />;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-white rounded-lg shadow p-6">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">Available Quests</h1>
        <p className="text-gray-600">Choose a quest to begin your adventure</p>
      </div>

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {error}
        </div>
      )}

      {/* Filters */}
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-4">Filters</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Difficulty Filter */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Difficulty
            </label>
            <select
              value={selectedDifficulty}
              onChange={(e) => setSelectedDifficulty(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500"
            >
              <option value="">All Difficulties</option>
              {QUEST_DIFFICULTIES.map(diff => (
                <option key={diff} value={diff}>{diff}</option>
              ))}
            </select>
          </div>

          {/* Type Filter */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Type
            </label>
            <select
              value={selectedType}
              onChange={(e) => setSelectedType(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500"
            >
              <option value="">All Types</option>
              {QUEST_TYPES.map(type => (
                <option key={type} value={type}>{type}</option>
              ))}
            </select>
          </div>

          {/* Region Filter */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Region
            </label>
            <select
              value={selectedRegion}
              onChange={(e) => setSelectedRegion(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500"
            >
              <option value="">All Regions</option>
              <option value="Lékié">Lékié</option>
              <option value="Douala">Douala</option>
              <option value="Yaoundé">Yaoundé</option>
            </select>
          </div>
        </div>

        {/* Clear Filters */}
        {(selectedDifficulty || selectedType || selectedRegion) && (
          <button
            onClick={() => {
              setSelectedDifficulty('');
              setSelectedType('');
              setSelectedRegion('');
            }}
            className="mt-4 px-4 py-2 text-sm text-blue-600 hover:text-blue-700 font-medium"
          >
            Clear Filters
          </button>
        )}
      </div>

      {/* Results */}
      <div className="space-y-2">
        <p className="text-sm text-gray-600">
          Showing {filteredQuests.length} of {availableQuests.length} quests
        </p>

        {filteredQuests.length === 0 ? (
          <div className="bg-white rounded-lg shadow p-12 text-center">
            <p className="text-gray-500">No quests match your filters. Try adjusting them!</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredQuests.map(quest => (
              <QuestCard key={quest.quest_id} quest={quest} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default QuestList;
