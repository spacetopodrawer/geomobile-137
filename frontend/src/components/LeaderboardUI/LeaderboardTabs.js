import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect } from 'react';
import { useSelector } from 'react-redux';
import { useLeaderboard } from '../../hooks/useLeaderboard';
import { REGIONS } from '../../utils/constants';
import Loading from '../Common/Loading';
import RankingTable from './RankingTable';
const LeaderboardTabs = () => {
    const { currentLeaderboard, scope, region, loading, lastUpdated, fetchLeaderboard, changeScopeAction, changeRegionAction } = useLeaderboard();
    const { global_rank } = useSelector((state) => state.user);
    useEffect(() => {
        fetchLeaderboard(scope, region);
    }, [scope, region]);
    if (loading)
        return _jsx(Loading, { message: "Loading leaderboard..." });
    return (_jsxs("div", { className: "space-y-6 py-8", children: [_jsxs("div", { className: "bg-gradient-to-r from-blue-600 to-cyan-600 text-white rounded-lg shadow p-6", children: [_jsx("h1", { className: "text-3xl font-bold mb-2", children: "Leaderboard" }), _jsx("p", { className: "text-blue-100", children: "Compete with players worldwide" })] }), _jsx("div", { className: "bg-gradient-to-r from-yellow-400 to-orange-400 rounded-lg shadow p-6 text-white", children: _jsxs("div", { className: "flex justify-between items-center", children: [_jsxs("div", { children: [_jsx("p", { className: "text-sm opacity-90", children: "Your Rank" }), _jsxs("p", { className: "text-4xl font-bold", children: ["#", global_rank] })] }), _jsx("div", { className: "text-5xl", children: "\uD83C\uDFC6" })] }) }), _jsxs("div", { className: "bg-white rounded-lg shadow", children: [_jsx("div", { className: "flex flex-wrap border-b border-gray-200", children: ['global', 'regional', 'weekly'].map(s => (_jsxs("button", { onClick: () => changeScopeAction(s), className: `px-6 py-4 font-medium border-b-2 transition-colors capitalize ${scope === s
                                ? 'border-blue-600 text-blue-600'
                                : 'border-transparent text-gray-600 hover:text-gray-900'}`, children: [s === 'regional' ? 'Regional' : s === 'weekly' ? 'Weekly' : 'Global', " Rankings"] }, s))) }), scope === 'regional' && (_jsxs("div", { className: "px-6 py-4 bg-gray-50 border-b border-gray-200", children: [_jsx("label", { className: "block text-sm font-medium text-gray-700 mb-2", children: "Select Region" }), _jsxs("select", { value: region, onChange: (e) => changeRegionAction(e.target.value), className: "w-full md:w-48 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500", children: [_jsx("option", { value: "", children: "All Regions" }), REGIONS.map(r => (_jsx("option", { value: r, children: r }, r)))] })] }))] }), lastUpdated && (_jsxs("p", { className: "text-sm text-gray-600 text-center", children: ["Last updated: ", new Date(lastUpdated).toLocaleTimeString()] })), _jsx(RankingTable, { rankings: currentLeaderboard })] }));
};
export default LeaderboardTabs;
