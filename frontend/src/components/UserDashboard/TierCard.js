import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { formatDate } from '../../utils/formatters';
const TierCard = ({ tier, tierName, expiration }) => {
    const tierColors = [
        'bg-gray-100 border-gray-300',
        'bg-blue-100 border-blue-300',
        'bg-purple-100 border-purple-300',
        'bg-pink-100 border-pink-300',
        'bg-yellow-100 border-yellow-300',
        'bg-orange-100 border-orange-300',
    ];
    const tierIcons = ['🌟', '🎯', '⚡', '🔥', '💎', '👑'];
    return (_jsxs("div", { className: `${tierColors[tier]} rounded-lg shadow p-6 border-2`, children: [_jsx("h3", { className: "text-lg font-bold text-gray-900 mb-4", children: "Subscription Tier" }), _jsxs("div", { className: "flex items-center justify-between mb-4", children: [_jsx("div", { children: _jsx("p", { className: "text-5xl", children: tierIcons[tier] }) }), _jsxs("div", { className: "text-right", children: [_jsx("p", { className: "text-sm text-gray-600", children: "Current" }), _jsx("p", { className: "text-3xl font-bold text-gray-900", children: tierName })] })] }), expiration ? (_jsxs("div", { className: "bg-white bg-opacity-70 rounded p-3", children: [_jsx("p", { className: "text-xs text-gray-600", children: "Expires" }), _jsx("p", { className: "text-sm font-semibold text-gray-900", children: formatDate(expiration) })] })) : (_jsxs("div", { className: "bg-white bg-opacity-70 rounded p-3", children: [_jsx("p", { className: "text-xs text-gray-600", children: "Status" }), _jsx("p", { className: "text-sm font-semibold text-gray-900", children: "Free Tier" })] })), _jsx("a", { href: "/profile?upgrade=true", className: "w-full mt-4 px-4 py-2 bg-gradient-to-r from-blue-600 to-purple-600 text-white rounded font-semibold hover:shadow-lg transition-all", children: "Upgrade Tier" })] }));
};
export default TierCard;
