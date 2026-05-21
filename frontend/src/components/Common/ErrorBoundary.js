import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import React from 'react';
class ErrorBoundary extends React.Component {
    constructor(props) {
        super(props);
        this.state = { hasError: false, error: null };
    }
    static getDerivedStateFromError(error) {
        return { hasError: true, error };
    }
    componentDidCatch(error, errorInfo) {
        console.error('ErrorBoundary caught error:', error, errorInfo);
    }
    render() {
        if (this.state.hasError) {
            return (_jsxs("div", { className: "error p-6 rounded-lg", children: [_jsx("h2", { className: "text-xl font-bold mb-2", children: "Something went wrong" }), _jsx("p", { className: "text-sm mb-4", children: this.state.error?.message }), _jsx("button", { onClick: () => window.location.reload(), className: "px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700", children: "Reload Page" })] }));
        }
        return this.props.children;
    }
}
export default ErrorBoundary;
