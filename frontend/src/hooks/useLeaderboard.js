import { useDispatch, useSelector } from 'react-redux';
import { useCallback } from 'react';
import { setGlobalLeaderboard, setRegionalLeaderboard, setWeeklyLeaderboard, setScope, setRegion, setLoading, setError, } from '../redux/slices/leaderboardSlice';
import { userService } from '../services/user';
export const useLeaderboard = () => {
    const dispatch = useDispatch();
    const { global, regional, weekly, scope, region, loading, error, lastUpdated } = useSelector((state) => state.leaderboard);
    const fetchLeaderboard = useCallback(async (selectedScope = 'global', selectedRegion, limit = 50) => {
        dispatch(setLoading(true));
        try {
            const response = await userService.getLeaderboard(selectedScope, selectedRegion, limit);
            if (selectedScope === 'global') {
                dispatch(setGlobalLeaderboard(response.rankings || []));
            }
            else if (selectedScope === 'regional') {
                dispatch(setRegionalLeaderboard(response.rankings || []));
            }
            else if (selectedScope === 'weekly') {
                dispatch(setWeeklyLeaderboard(response.rankings || []));
            }
            dispatch(setError(null));
        }
        catch (err) {
            dispatch(setError(err instanceof Error ? err.message : 'Failed to fetch leaderboard'));
        }
        finally {
            dispatch(setLoading(false));
        }
    }, [dispatch]);
    const changeScopeAction = useCallback((newScope) => {
        dispatch(setScope(newScope));
    }, [dispatch]);
    const changeRegionAction = useCallback((newRegion) => {
        dispatch(setRegion(newRegion));
    }, [dispatch]);
    const getCurrentLeaderboard = () => {
        if (scope === 'global')
            return global;
        if (scope === 'regional')
            return regional;
        if (scope === 'weekly')
            return weekly;
        return [];
    };
    return {
        currentLeaderboard: getCurrentLeaderboard(),
        scope,
        region,
        loading,
        error,
        lastUpdated,
        fetchLeaderboard,
        changeScopeAction,
        changeRegionAction,
    };
};
