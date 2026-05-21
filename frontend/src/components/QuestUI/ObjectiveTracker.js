import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useQuest } from '../../hooks/useQuest';
import { formatXP } from '../../utils/formatters';
const ObjectiveTracker = ({ objective, sessionId }) => {
    const { completeObjective } = useQuest();
    const handleComplete = async () => {
        if (!objective.completed) {
            await completeObjective(sessionId, objective.id);
        }
    };
    return (_jsxs("div", { className: "flex items-start gap-4 p-4 border border-gray-200 rounded-lg hover:border-blue-300 transition-colors", children: [_jsx("button", { onClick: handleComplete, className: `flex-shrink-0 mt-1 w-6 h-6 rounded border-2 transition-all ${objective.completed
                    ? 'bg-green-600 border-green-600'
                    : 'border-gray-300 hover:border-blue-400'}`, children: objective.completed && (_jsx("svg", { className: "w-4 h-4 text-white", fill: "currentColor", viewBox: "0 0 20 20", children: _jsx("path", { fillRule: "evenodd", d: "M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z", clipRule: "evenodd" }) })) }), _jsx("div", { className: "flex-1", children: _jsx("p", { className: `font-medium transition-all ${objective.completed ? 'text-gray-400 line-through' : 'text-gray-900'}`, children: objective.description }) }), _jsx("div", { className: "flex-shrink-0 text-right", children: _jsxs("p", { className: `text-sm font-semibold ${objective.completed ? 'text-green-600' : 'text-gray-600'}`, children: ["+", formatXP(objective.xp_reward), " XP"] }) })] }));
};
export default ObjectiveTracker;
