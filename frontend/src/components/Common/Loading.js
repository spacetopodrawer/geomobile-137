import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
const Loading = ({ message = 'Loading...', fullscreen = false }) => {
    const className = fullscreen ? 'loading' : 'flex justify-center items-center py-12';
    return (_jsx("div", { className: className, children: _jsxs("div", { className: "text-center", children: [_jsx("div", { className: "animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4" }), _jsx("p", { className: "text-gray-600", children: message })] }) }));
};
export default Loading;
