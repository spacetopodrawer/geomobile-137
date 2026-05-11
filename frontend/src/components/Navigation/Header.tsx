import React from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { logout } from '../../redux/slices/authSlice';
import { RootState } from '../../redux/store';
import { TIERS } from '../../utils/constants';

const Header: React.FC = () => {
  const dispatch = useDispatch();
  const { userId } = useSelector((state: RootState) => state.auth);
  const { tier_level, level, total_xp } = useSelector((state: RootState) => state.user);

  const handleLogout = () => {
    dispatch(logout());
    window.location.href = '/';
  };

  return (
    <header className="bg-white shadow">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-8">
            <h1 className="text-2xl font-bold text-blue-600">Geo-Mobile137</h1>
            <nav className="hidden md:flex gap-6">
              <a href="/quests" className="hover:text-blue-600">
                Quests
              </a>
              <a href="/map" className="hover:text-blue-600">
                Map
              </a>
              <a href="/shop" className="hover:text-blue-600">
                Shop
              </a>
              <a href="/leaderboard" className="hover:text-blue-600">
                Leaderboard
              </a>
            </nav>
          </div>

          <div className="flex items-center gap-4">
            <div className="text-right hidden sm:block">
              <p className="text-sm font-semibold">Level {level}</p>
              <p className="text-xs text-gray-500">{TIERS[tier_level as keyof typeof TIERS]}</p>
            </div>
            <button
              onClick={handleLogout}
              className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
            >
              Logout
            </button>
          </div>
        </div>
      </div>
    </header>
  );
};

export default Header;
