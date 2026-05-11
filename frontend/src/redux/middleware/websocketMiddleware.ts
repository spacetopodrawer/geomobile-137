import { Middleware } from '@reduxjs/toolkit';
import { io, Socket } from 'socket.io-client';
import {
  updateObjective,
  completeQuest,
} from '../slices/questSlice';
import {
  addXP,
  updateRanks,
} from '../slices/userSlice';
import {
  updateRanking,
} from '../slices/leaderboardSlice';
import {
  completePayment,
} from '../slices/paymentSlice';

let socket: Socket | null = null;

const websocketMiddleware: Middleware = (store) => (next) => (action) => {
  const result = next(action);

  if (!socket) {
    const userId = localStorage.getItem('userId');
    if (userId) {
      socket = io(import.meta.env.VITE_WEBSOCKET_URL || 'http://localhost:8080', {
        query: { userId },
      });

      socket.on('quest:objective_complete', (data) => {
        store.dispatch(updateObjective({
          objectiveId: data.objective_id,
          completed: true,
        }));
      });

      socket.on('user:xp_gained', (data) => {
        store.dispatch(addXP(data.xp_amount));
        if (data.new_rank || data.region_rank) {
          store.dispatch(updateRanks({
            globalRank: data.new_rank || 0,
            regionRank: data.region_rank || 0,
          }));
        }
      });

      socket.on('leaderboard:rank_updated', (data) => {
        store.dispatch(updateRanking({
          rank: data.rank,
          user_id: data.user_id,
          username: data.username,
          total_xp: data.total_xp,
          tier_level: data.tier_level,
          completed_quests: data.completed_quests,
        }));
      });

      socket.on('payment:completed', (data) => {
        store.dispatch(completePayment());
      });
    }
  }

  return result;
};

export default websocketMiddleware;
