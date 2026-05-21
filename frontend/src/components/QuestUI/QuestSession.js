import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useEffect } from 'react';
import { useQuest } from '../../hooks/useQuest';
import { formatXP } from '../../utils/formatters';
import ObjectiveTracker from './ObjectiveTracker';
const QuestSession = () => {
    const { activeSession, finishQuest, abandonCurrentQuest } = useQuest();
    const [elapsedTime, setElapsedTime] = useState(0);
    const [showAbandonment, setShowAbandonment] = useState(false);
    useEffect(() => {
        if (!activeSession)
            return;
        const timer = setInterval(() => {
            setElapsedTime(prev => prev + 1);
        }, 1000);
        return () => clearInterval(timer);
    }, [activeSession]);
    if (!activeSession) {
        return (_jsx("div", { className: "bg-white rounded-lg shadow p-8 text-center", children: _jsx("p", { className: "text-gray-500", children: "No active quest. Start one from the quest list!" }) }));
    }
    const completedObjectives = activeSession.objectives.filter(obj => obj.completed).length;
    const totalObjectives = activeSession.objectives.length;
    const progressPercent = (completedObjectives / totalObjectives) * 100;
    const handleComplete = async () => {
        if (completedObjectives === totalObjectives) {
            await finishQuest(activeSession.session_id);
        }
    };
    const handleAbandon = () => {
        abandonCurrentQuest(activeSession.session_id);
        setShowAbandonment(false);
    };
    return (_jsxs("div", { className: "space-y-6", children: [_jsx("div", { className: "bg-gradient-to-r from-green-600 to-green-700 text-white rounded-lg shadow p-6", children: _jsxs("div", { className: "flex justify-between items-start mb-4", children: [_jsxs("div", { children: [_jsx("h1", { className: "text-3xl font-bold", children: "Active Quest" }), _jsxs("p", { className: "text-green-100 mt-1", children: ["Session ID: ", activeSession.session_id.substring(0, 8), "..."] })] }), _jsxs("div", { className: "text-right", children: [_jsx("p", { className: "text-sm text-green-100", children: "Elapsed Time" }), _jsxs("p", { className: "text-2xl font-bold", children: [Math.floor(elapsedTime / 60), ":", (elapsedTime % 60).toString().padStart(2, '0')] })] })] }) }), _jsxs("div", { className: "bg-white rounded-lg shadow p-6", children: [_jsxs("div", { className: "flex justify-between items-center mb-2", children: [_jsx("h3", { className: "text-lg font-semibold", children: "Progress" }), _jsxs("span", { className: "text-sm text-gray-600", children: [completedObjectives, " / ", totalObjectives, " objectives complete"] })] }), _jsx("div", { className: "w-full bg-gray-200 rounded-full h-3", children: _jsx("div", { className: "bg-green-600 h-3 rounded-full transition-all duration-500", style: { width: `${progressPercent}%` } }) })] }), _jsxs("div", { className: "bg-white rounded-lg shadow p-6", children: [_jsx("h3", { className: "text-lg font-semibold mb-4", children: "Objectives" }), _jsx("div", { className: "space-y-3", children: activeSession.objectives.map(objective => (_jsx(ObjectiveTracker, { objective: objective, sessionId: activeSession.session_id }, objective.id))) })] }), _jsx("div", { className: "bg-white rounded-lg shadow p-6", children: _jsxs("div", { className: "flex justify-between items-center", children: [_jsxs("div", { children: [_jsx("p", { className: "text-sm text-gray-600", children: "Current XP Earned" }), _jsx("p", { className: "text-3xl font-bold text-green-600", children: formatXP(activeSession.xp_earned) })] }), _jsxs("div", { className: "text-right", children: [_jsx("p", { className: "text-xs text-gray-500 mb-2", children: "Progress towards completion" }), _jsxs("div", { className: "text-xl font-semibold text-gray-700", children: [Math.round(progressPercent), "%"] })] })] }) }), _jsxs("div", { className: "bg-white rounded-lg shadow p-6 flex gap-3", children: [_jsx("button", { onClick: handleComplete, disabled: completedObjectives !== totalObjectives, className: `flex-1 px-6 py-3 rounded-lg font-semibold transition-colors ${completedObjectives === totalObjectives
                            ? 'bg-green-600 text-white hover:bg-green-700'
                            : 'bg-gray-200 text-gray-500 cursor-not-allowed'}`, children: "Complete Quest" }), _jsx("button", { onClick: () => setShowAbandonment(true), className: "flex-1 px-6 py-3 bg-red-600 text-white rounded-lg font-semibold hover:bg-red-700 transition-colors", children: "Abandon" })] }), showAbandonment && (_jsx("div", { className: "fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50", children: _jsxs("div", { className: "bg-white rounded-lg shadow-lg p-6 max-w-sm", children: [_jsx("h3", { className: "text-lg font-bold text-gray-900 mb-3", children: "Abandon Quest?" }), _jsx("p", { className: "text-gray-600 mb-6", children: "You will lose all progress and earned XP for this quest. This action cannot be undone." }), _jsxs("div", { className: "flex gap-3", children: [_jsx("button", { onClick: () => setShowAbandonment(false), className: "flex-1 px-4 py-2 bg-gray-200 text-gray-900 rounded hover:bg-gray-300 font-semibold", children: "Cancel" }), _jsx("button", { onClick: handleAbandon, className: "flex-1 px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 font-semibold", children: "Confirm Abandon" })] })] }) }))] }));
};
export default QuestSession;
