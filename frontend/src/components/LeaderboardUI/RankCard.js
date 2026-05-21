import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { TIERS, COSMETIC_DISCOUNTS } from '../../utils/constants';
import { formatXP } from '../../utils/formatters';
const RankCard = ({ rank, username, xp, tier, quests, isHighlighted = false, medal, }) => {
    const tierName = TIERS[tier] || 'Free';
    const discount = COSMETIC_DISCOUNTS[tier] || 0;
    return (_jsxs("tr", { className: isHighlighted ? 'bg-yellow-50' : 'hover:bg-gray-50', children: [_jsx("td", { className: "px-6 py-4 whitespace-nowrap", children: _jsxs("div", { className: "flex items-center gap-2", children: [_jsx("span", { className: "text-2xl", children: medal || '' }), _jsxs("span", { className: "font-bold text-gray-900", children: ["#", rank] })] }) }), _jsx("td", { className: "px-6 py-4 whitespace-nowrap", children: _jsxs("div", { className: "flex items-center gap-3", children: [_jsx("div", { className: "w-10 h-10 rounded-full bg-gradient-to-br from-blue-400 to-purple-500 flex items-center justify-center text-white font-bold", children: username.charAt(0).toUpperCase() }), _jsxs("div", { children: [_jsx("p", { className: "font-semibold text-gray-900", children: username }), _jsx("p", { className: "text-xs text-gray-500", children: tierName })] })] }) }), _jsx("td", { className: "px-6 py-4 whitespace-nowrap", children: _jsx("p", { className: "font-bold text-green-600", children: formatXP(xp) }) }), _jsx("td", { className: "px-6 py-4 whitespace-nowrap", children: _jsx("span", { className: `px-3 py-1 rounded-full text-sm font-semibold ${tier === 0 ? 'bg-gray-100 text-gray-800' :
                        tier <= 2 ? 'bg-blue-100 text-blue-800' :
                            tier <= 4 ? 'bg-purple-100 text-purple-800' :
                                'bg-gold-100 text-gold-800'}`, children: tierName }) }), _jsx("td", { className: "px-6 py-4 whitespace-nowrap", children: _jsx("p", { className: "text-gray-900 font-medium", children: quests }) })] }));
};
export default RankCard;
