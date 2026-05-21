import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect } from 'react';
import { useSelector } from 'react-redux';
import { useUser } from '../../hooks/useUser';
import { TIERS } from '../../utils/constants';
import { calculateProgress } from '../../utils/formatters';
import ProgressCard from './ProgressCard';
import TierCard from './TierCard';
import StatsPanel from './StatsPanel';
const UserDashboard = () => {
    const { fetchUserProgress } = useUser();
    const user = useSelector((state) => state.user);
    useEffect(() => {
        fetchUserProgress();
    }, []);
    const tierName = TIERS[user.tier_level] || 'Free';
    const progressPercent = calculateProgress(user.total_xp);
    return (_jsxs("div", { className: "space-y-6 py-8", children: [_jsx("div", { className: "bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg shadow p-6", children: _jsxs("div", { className: "flex justify-between items-start", children: [_jsxs("div", { children: [_jsx("h1", { className: "text-3xl font-bold mb-2", children: "Profile" }), _jsx("p", { className: "text-indigo-100", children: "Welcome back!" })] }), _jsx("div", { className: "text-right", children: _jsx("div", { className: "w-16 h-16 rounded-full bg-white bg-opacity-20 flex items-center justify-center text-3xl", children: "\uD83D\uDC64" }) })] }) }), _jsxs("div", { className: "grid grid-cols-1 md:grid-cols-2 gap-6", children: [_jsx(ProgressCard, { level: user.level, currentXP: user.total_xp, nextLevelXP: (user.level * 1000), progressPercent: progressPercent }), _jsx(TierCard, { tier: user.tier_level, tierName: tierName, expiration: user.tier_expiration })] }), _jsx(StatsPanel, { globalRank: user.global_rank, regionRank: user.region_rank, region: user.region, completedQuests: user.completed_quests, badgesEarned: user.badges_earned.length }), user.badges_earned.length > 0 && (_jsxs("div", { className: "bg-white rounded-lg shadow p-6", children: [_jsx("h3", { className: "text-lg font-bold text-gray-900 mb-4", children: "Badges Earned" }), _jsx("div", { className: "grid grid-cols-2 md:grid-cols-4 gap-4", children: user.badges_earned.map((badge, idx) => (_jsxs("div", { className: "text-center p-4 bg-yellow-50 rounded-lg border-2 border-yellow-200", children: [_jsx("p", { className: "text-3xl mb-2", children: "\u2B50" }), _jsx("p", { className: "text-sm font-semibold text-gray-900", children: badge })] }, idx))) })] })), _jsxs("div", { className: "grid grid-cols-1 md:grid-cols-2 gap-4", children: [_jsxs("a", { href: "/quests", className: "bg-blue-600 text-white rounded-lg shadow p-4 hover:bg-blue-700 transition-colors", children: [_jsx("p", { className: "font-bold", children: "Start a Quest" }), _jsx("p", { className: "text-sm text-blue-100", children: "Continue your adventure" })] }), _jsxs("a", { href: "/shop", className: "bg-purple-600 text-white rounded-lg shadow p-4 hover:bg-purple-700 transition-colors", children: [_jsx("p", { className: "font-bold", children: "Cosmetics Shop" }), _jsx("p", { className: "text-sm text-purple-100", children: "Upgrade your appearance" })] })] })] }));
};
export default UserDashboard;
