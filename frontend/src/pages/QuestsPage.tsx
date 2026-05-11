import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../redux/store';
import QuestList from '../components/QuestUI/QuestList';
import QuestSession from '../components/QuestUI/QuestSession';

const QuestsPage: React.FC = () => {
  const { activeSession } = useSelector((state: RootState) => state.quest);

  return (
    <div className="space-y-8 py-8">
      {activeSession ? (
        <>
          <div className="flex gap-6">
            <div className="flex-1">
              <QuestSession />
            </div>
          </div>
        </>
      ) : (
        <QuestList />
      )}
    </div>
  );
};

export default QuestsPage;
