import { useDispatch, useSelector } from 'react-redux';
import { useCallback } from 'react';
import { RootState } from '../redux/store';
import {
  setUserProgress,
  addXP,
  incrementQuestCompletion,
  updateTier,
  updateRanks,
  setLoading,
  setError,
} from '../redux/slices/userSlice';
import { userService } from '../services/user';

export const useUser = () => {
  const dispatch = useDispatch();
  const {
    userId,
    level,
    total_xp,
    tier_level,
    completed_quests,
    global_rank,
    region_rank,
    loading,
    error,
  } = useSelector((state: RootState) => state.user);

  const fetchUserProgress = useCallback(async () => {
    dispatch(setLoading(true));
    try {
      const progress = await userService.getUserProgress();
      dispatch(setUserProgress(progress));
      dispatch(setError(null));
    } catch (err) {
      dispatch(setError(err instanceof Error ? err.message : 'Failed to fetch user progress'));
    } finally {
      dispatch(setLoading(false));
    }
  }, [dispatch]);

  const gainXP = useCallback(
    (amount: number) => {
      dispatch(addXP(amount));
    },
    [dispatch]
  );

  const completeQuestAction = useCallback(() => {
    dispatch(incrementQuestCompletion());
  }, [dispatch]);

  const upgradeTierAction = useCallback(
    async (newTier: number, durationDays: number) => {
      dispatch(setLoading(true));
      try {
        const result = await userService.upgradeTier(newTier, durationDays);
        dispatch(
          updateTier({
            tier: newTier,
            expiration: result.expiration,
          })
        );
        dispatch(setError(null));
        return result;
      } catch (err) {
        dispatch(setError(err instanceof Error ? err.message : 'Failed to upgrade tier'));
      } finally {
        dispatch(setLoading(false));
      }
    },
    [dispatch]
  );

  const updateRanksAction = useCallback(
    (globalRank: number, regionRank: number) => {
      dispatch(updateRanks({ globalRank, regionRank }));
    },
    [dispatch]
  );

  return {
    userId,
    level,
    total_xp,
    tier_level,
    completed_quests,
    global_rank,
    region_rank,
    loading,
    error,
    fetchUserProgress,
    gainXP,
    completeQuestAction,
    upgradeTierAction,
    updateRanksAction,
  };
};
