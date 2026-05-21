import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
export const ActivityStream = () => {
    const { messageHistory } = useWebSocket();
    const [isOpen, setIsOpen] = useState(true);
    return (_jsxs("div", { className: "fixed bottom-4 right-4 w-96 bg-white rounded-lg shadow-lg", children: [_jsxs("div", { className: "p-3 border-b flex justify-between items-center cursor-pointer hover:bg-gray-50", onClick: () => setIsOpen(!isOpen), children: [_jsx("h3", { className: "font-bold", children: "Activity Stream" }), _jsxs("span", { className: "text-xs text-gray-500", children: ["(", messageHistory.length, ")"] })] }), isOpen && (_jsx("div", { className: "max-h-72 overflow-y-auto p-3 space-y-2 bg-gray-50", children: messageHistory.length === 0 ? (_jsx("div", { className: "text-gray-400 text-sm text-center py-8", children: "Waiting for messages..." })) : (messageHistory
                    .slice()
                    .reverse()
                    .map((msg, idx) => (_jsxs("div", { className: "text-sm border-l-2 border-blue-400 pl-2 bg-white p-2 rounded", children: [_jsx("div", { className: "font-semibold text-xs text-blue-600", children: msg.type }), _jsx("div", { className: "text-gray-700 text-xs", children: msg.device_id || 'server' }), _jsx("div", { className: "text-xs text-gray-400", children: new Date(msg.timestamp || 0).toLocaleTimeString() })] }, idx)))) }))] }));
};
