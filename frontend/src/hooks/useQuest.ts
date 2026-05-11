import { useDispatch, useSelector } from 'react-redux';
import { useCallback } from 'react';
import { RootState } from '../redux/store';
import {
  setAvailableQuests,
  startQuest,
  updateObjective,
  completeQuest,
  abandonQuest,
  setLoading,
  setError,
} from '../redux/slices/questSlice';
import { questService } from '../services/quest';

export const useQuest = () => {
  const dispatch = useDispatch();
  const { availableQuests, activeSession, loading, error } = useSelector(
    (state: RootState) => state.quest
  );

  const fetchAvailableQuests = useCallback(
    async (limit: number = 20) => {
      dispatch(setLoading(true));
      try {
        const quests = await questService.getAvailableQuests(limit);
        dispatch(setAvailableQuests(quests));
        dispatch(setError(null));
      } catch (err) {
        dispatch(setError(err instanceof Error ? err.message : 'Failed to fetch quests'));
      } finally {
        dispatch(setLoading(false));
      }
    },
    [dispatch]
  );

  const startNewQuest = useCallback(
    async (questId: string) => {
      dispatch(setLoading(true));
      try {
        const session = await questService.startQuest(questId);
        dispatch(startQuest(session));
        dispatch(setError(null));
      } catch (err) {
        dispatch(setError(err instanceof Error ? err.message : 'Failed to start quest'));
      } finally {
        dispatch(setLoading(false));
      }
    },
    [dispatch]
  );

  const completeObjective = useCallback(
    async (sessionId: string, objectiveId: string) => {
      try {
        await questService.completeObjective(sessionId, objectiveId);
        dispatch(updateObjective({ objectiveId, completed: true }));
        dispatch(setError(null));
      } catch (err) {
        dispatch(setError(err instanceof Error ? err.message : 'Failed to complete objective'));
      }
    },
    [dispatch]
  );

  const finishQuest = useCallback(
    async (sessionId: string) => {
      dispatch(setLoading(true));
      try {
        const result = await questService.completeQuest(sessionId);
        dispatch(completeQuest({ sessionId, xpEarned: result.xp_earned }));
        dispatch(setError(null));
        return result;
      } catch (err) {
        dispatch(setError(err instanceof Error ? err.message : 'Failed to complete quest'));
      } finally {
        dispatch(setLoading(false));
      }
    },
    [dispatch]
  );

  const abandonCurrentQuest = useCallback(
    async (sessionId: string) => {
      try {
        await questService.abandonQuest(sessionId);
        dispatch(abandonQuest());
        dispatch(setError(null));
      } catch (err) {
        dispatch(setError(err instanceof Error ? err.message : 'Failed to abandon quest'));
      }
    },
    [dispatch]
  );

  return {
    availableQuests,
    activeSession,
    loading,
    error,
    fetchAvailableQuests,
    startNewQuest,
    completeObjective,
    finishQuest,
    abandonCurrentQuest,
  };
};
