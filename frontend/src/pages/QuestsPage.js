import { jsx as _jsx, Fragment as _Fragment } from "react/jsx-runtime";
import { useSelector } from 'react-redux';
import QuestList from '../components/QuestUI/QuestList';
import QuestSession from '../components/QuestUI/QuestSession';
const QuestsPage = () => {
    const { activeSession } = useSelector((state) => state.quest);
    return (_jsx("div", { className: "space-y-8 py-8", children: activeSession ? (_jsx(_Fragment, { children: _jsx("div", { className: "flex gap-6", children: _jsx("div", { className: "flex-1", children: _jsx(QuestSession, {}) }) }) })) : (_jsx(QuestList, {})) }));
};
export default QuestsPage;
