import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useWebSocket } from '../hooks/useWebSocket';
export const WebSocketStatus = () => {
    const store = useWebSocket();
    const { connected, connecting, error, deviceId } = store;
    return (_jsxs("div", { className: "fixed top-4 right-4 p-4 rounded-lg shadow-lg max-w-xs bg-white", children: [connecting && (_jsx("div", { className: "bg-yellow-100 text-yellow-900 p-2 rounded", children: "\u27F3 Connecting..." })), connected && (_jsxs("div", { className: "bg-green-100 text-green-900 p-2 rounded", children: ["\u2705 Connected (", deviceId.slice(0, 8), ")"] })), error && (_jsxs("div", { className: "bg-red-100 text-red-900 p-2 rounded", children: ["\u274C Error: ", error] })), !connected && !connecting && !error && (_jsx("div", { className: "bg-gray-100 text-gray-900 p-2 rounded", children: "\u26AB Disconnected" }))] }));
};
