import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useState } from 'react';
import { useQuest } from '../../hooks/useQuest';
import { QUEST_DIFFICULTIES, QUEST_TYPES } from '../../utils/constants';
import Loading from '../Common/Loading';
import QuestCard from './QuestCard';
const QuestList = () => {
    const { availableQuests, loading, error, fetchAvailableQuests } = useQuest();
    const [filteredQuests, setFilteredQuests] = useState(availableQuests);
    const [selectedDifficulty, setSelectedDifficulty] = useState('');
    const [selectedType, setSelectedType] = useState('');
    const [selectedRegion, setSelectedRegion] = useState('');
    useEffect(() => {
        fetchAvailableQuests(50);
    }, []);
    useEffect(() => {
        let filtered = availableQuests;
        if (selectedDifficulty) {
            filtered = filtered.filter(q => q.difficulty === selectedDifficulty);
        }
        if (selectedType) {
            filtered = filtered.filter(q => q.type === selectedType);
        }
        if (selectedRegion) {
            filtered = filtered.filter(q => q.region === selectedRegion);
        }
        setFilteredQuests(filtered);
    }, [availableQuests, selectedDifficulty, selectedType, selectedRegion]);
    if (loading)
        return _jsx(Loading, { message: "Loading quests..." });
    return (_jsxs("div", { className: "space-y-6", children: [_jsxs("div", { className: "bg-white rounded-lg shadow p-6", children: [_jsx("h1", { className: "text-3xl font-bold text-gray-900 mb-2", children: "Available Quests" }), _jsx("p", { className: "text-gray-600", children: "Choose a quest to begin your adventure" })] }), error && (_jsx("div", { className: "bg-red-50 border border-red-200 rounded-lg p-4 text-red-700", children: error })), _jsxs("div", { className: "bg-white rounded-lg shadow p-6", children: [_jsx("h2", { className: "text-lg font-semibold mb-4", children: "Filters" }), _jsxs("div", { className: "grid grid-cols-1 md:grid-cols-3 gap-4", children: [_jsxs("div", { children: [_jsx("label", { className: "block text-sm font-medium text-gray-700 mb-2", children: "Difficulty" }), _jsxs("select", { value: selectedDifficulty, onChange: (e) => setSelectedDifficulty(e.target.value), className: "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500", children: [_jsx("option", { value: "", children: "All Difficulties" }), QUEST_DIFFICULTIES.map(diff => (_jsx("option", { value: diff, children: diff }, diff)))] })] }), _jsxs("div", { children: [_jsx("label", { className: "block text-sm font-medium text-gray-700 mb-2", children: "Type" }), _jsxs("select", { value: selectedType, onChange: (e) => setSelectedType(e.target.value), className: "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500", children: [_jsx("option", { value: "", children: "All Types" }), QUEST_TYPES.map(type => (_jsx("option", { value: type, children: type }, type)))] })] }), _jsxs("div", { children: [_jsx("label", { className: "block text-sm font-medium text-gray-700 mb-2", children: "Region" }), _jsxs("select", { value: selectedRegion, onChange: (e) => setSelectedRegion(e.target.value), className: "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-blue-500", children: [_jsx("option", { value: "", children: "All Regions" }), _jsx("option", { value: "L\u00E9ki\u00E9", children: "L\u00E9ki\u00E9" }), _jsx("option", { value: "Douala", children: "Douala" }), _jsx("option", { value: "Yaound\u00E9", children: "Yaound\u00E9" })] })] })] }), (selectedDifficulty || selectedType || selectedRegion) && (_jsx("button", { onClick: () => {
                            setSelectedDifficulty('');
                            setSelectedType('');
                            setSelectedRegion('');
                        }, className: "mt-4 px-4 py-2 text-sm text-blue-600 hover:text-blue-700 font-medium", children: "Clear Filters" }))] }), _jsxs("div", { className: "space-y-2", children: [_jsxs("p", { className: "text-sm text-gray-600", children: ["Showing ", filteredQuests.length, " of ", availableQuests.length, " quests"] }), filteredQuests.length === 0 ? (_jsx("div", { className: "bg-white rounded-lg shadow p-12 text-center", children: _jsx("p", { className: "text-gray-500", children: "No quests match your filters. Try adjusting them!" }) })) : (_jsx("div", { className: "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6", children: filteredQuests.map(quest => (_jsx(QuestCard, { quest: quest }, quest.quest_id))) }))] })] }));
};
export default QuestList;
