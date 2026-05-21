import { jsxs as _jsxs, jsx as _jsx } from "react/jsx-runtime";
import { usePresenceStore } from '../stores/presenceStore';
export const PresenceList = () => {
    const users = usePresenceStore((state) => state.getUsers());
    return (_jsxs("div", { className: "fixed left-4 top-4 w-56 bg-white rounded-lg shadow-lg p-4", children: [_jsxs("h3", { className: "font-bold text-lg mb-3", children: ["Online Users: ", users.length] }), _jsx("div", { className: "space-y-2 max-h-64 overflow-y-auto", children: users.map((user) => (_jsxs("div", { className: "flex items-center gap-2 p-2 bg-gray-50 rounded text-sm", children: [_jsx("div", { className: "w-2 h-2 bg-green-500 rounded-full" }), _jsx("span", { className: "font-mono text-xs", children: user.deviceId.slice(0, 12) }), _jsx("span", { className: "text-xs text-gray-500", children: user.status === 'active' ? '👁️' : '😴' })] }, user.deviceId))) }), users.length === 0 && (_jsx("div", { className: "text-gray-400 text-sm", children: "Waiting for other users..." }))] }));
};
