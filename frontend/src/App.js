import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { setUserId } from './redux/slices/authSlice';
import ErrorBoundary from './components/Common/ErrorBoundary';
import Header from './components/Navigation/Header';
import { WebSocketStatus } from './components/WebSocketStatus';
import { PresenceList } from './components/PresenceList';
import { ActivityStream } from './components/ActivityStream';
import { useWebSocket } from './hooks/useWebSocket';
import { usePresenceStore } from './stores/presenceStore';
import QuestsPage from './pages/QuestsPage';
import MapPage from './pages/MapPage';
import ShopPage from './pages/ShopPage';
import LeaderboardPage from './pages/LeaderboardPage';
import ProfilePage from './pages/ProfilePage';
const LoginPage = () => _jsx("div", { className: "p-8", children: _jsx("h2", { className: "text-2xl font-bold", children: "Login" }) });
const App = () => {
    const dispatch = useDispatch();
    const { isAuthenticated, userId } = useSelector((state) => state.auth);
    const wsStore = useWebSocket();
    const { connect } = wsStore;
    const addUser = usePresenceStore((state) => state.addUser);
    useEffect(() => {
        const storedUserId = localStorage.getItem('userId');
        if (storedUserId && !userId) {
            dispatch(setUserId(storedUserId));
        }
    }, [dispatch, userId]);
    // Auto-connect WebSocket on mount
    useEffect(() => {
        connect();
    }, []);
    // Add self to presence list when connected
    useEffect(() => {
        if (wsStore.connected) {
            addUser({
                deviceId: wsStore.deviceId,
                userId: wsStore.userId,
                status: 'active',
                lastSeen: Date.now(),
            });
        }
    }, [wsStore.connected, wsStore.deviceId, wsStore.userId]);
    if (!isAuthenticated) {
        return (_jsx(ErrorBoundary, { children: _jsxs(Router, { children: [_jsxs(Routes, { children: [_jsx(Route, { path: "/login", element: _jsx(LoginPage, {}) }), _jsx(Route, { path: "*", element: _jsx(Navigate, { to: "/login", replace: true }) })] }), _jsx(WebSocketStatus, {}), _jsx(ActivityStream, {})] }) }));
    }
    return (_jsx(ErrorBoundary, { children: _jsxs(Router, { children: [_jsxs("div", { className: "min-h-screen bg-gray-50", children: [_jsx(Header, {}), _jsx("main", { className: "max-w-7xl mx-auto", children: _jsxs(Routes, { children: [_jsx(Route, { path: "/quests", element: _jsx(QuestsPage, {}) }), _jsx(Route, { path: "/map", element: _jsx(MapPage, {}) }), _jsx(Route, { path: "/shop", element: _jsx(ShopPage, {}) }), _jsx(Route, { path: "/leaderboard", element: _jsx(LeaderboardPage, {}) }), _jsx(Route, { path: "/profile", element: _jsx(ProfilePage, {}) }), _jsx(Route, { path: "/", element: _jsx(Navigate, { to: "/quests", replace: true }) })] }) })] }), _jsx(WebSocketStatus, {}), _jsx(PresenceList, {}), _jsx(ActivityStream, {})] }) }));
};
export default App;
