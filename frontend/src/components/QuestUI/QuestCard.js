import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useDispatch } from 'react-redux';
import { useQuest } from '../../hooks/useQuest';
import { formatXP, formatDuration } from '../../utils/formatters';
const QuestCard = ({ quest }) => {
    const dispatch = useDispatch();
    const { startNewQuest } = useQuest();
    const difficultyColor = {
        Easy: 'bg-green-100 text-green-800',
        Normal: 'bg-blue-100 text-blue-800',
        Hard: 'bg-orange-100 text-orange-800',
        Master: 'bg-purple-100 text-purple-800',
    };
    const handleStartQuest = () => {
        startNewQuest(quest.quest_id);
    };
    return (_jsxs("div", { className: "bg-white rounded-lg shadow hover:shadow-lg transition-shadow", children: [_jsxs("div", { className: "bg-gradient-to-r from-blue-600 to-blue-700 text-white p-4 rounded-t-lg", children: [_jsx("h3", { className: "text-lg font-bold", children: quest.title }), _jsx("p", { className: "text-sm text-blue-100 mt-1", children: quest.region })] }), _jsxs("div", { className: "p-4 space-y-3", children: [_jsx("p", { className: "text-gray-700 text-sm line-clamp-2", children: quest.description }), _jsxs("div", { className: "flex flex-wrap gap-2", children: [_jsx("span", { className: `px-2 py-1 rounded text-xs font-semibold ${difficultyColor[quest.difficulty] || 'bg-gray-100 text-gray-800'}`, children: quest.difficulty }), _jsx("span", { className: "px-2 py-1 bg-purple-100 text-purple-800 rounded text-xs font-semibold", children: quest.type })] }), _jsxs("div", { className: "grid grid-cols-2 gap-3 pt-2 border-t border-gray-200", children: [_jsxs("div", { children: [_jsx("p", { className: "text-xs text-gray-500", children: "XP Reward" }), _jsx("p", { className: "text-sm font-bold text-green-600", children: formatXP(quest.xp_reward) })] }), _jsxs("div", { children: [_jsx("p", { className: "text-xs text-gray-500", children: "Duration" }), _jsx("p", { className: "text-sm font-bold text-gray-700", children: formatDuration(quest.estimated_duration * 60) })] }), _jsxs("div", { children: [_jsx("p", { className: "text-xs text-gray-500", children: "Min Tier" }), _jsxs("p", { className: "text-sm font-bold text-gray-700", children: ["Tier ", quest.min_tier] })] }), _jsxs("div", { children: [_jsx("p", { className: "text-xs text-gray-500", children: "Min XP" }), _jsx("p", { className: "text-sm font-bold text-gray-700", children: formatXP(quest.min_xp) })] })] })] }), _jsx("div", { className: "px-4 py-3 bg-gray-50 rounded-b-lg border-t border-gray-100", children: _jsx("button", { onClick: handleStartQuest, className: "w-full px-4 py-2 bg-blue-600 text-white font-semibold rounded hover:bg-blue-700 transition-colors", children: "Start Quest" }) })] }));
};
export default QuestCard;
