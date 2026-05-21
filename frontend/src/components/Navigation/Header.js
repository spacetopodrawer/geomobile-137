import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useDispatch, useSelector } from 'react-redux';
import { logout } from '../../redux/slices/authSlice';
import { TIERS } from '../../utils/constants';
const Header = () => {
    const dispatch = useDispatch();
    const { userId } = useSelector((state) => state.auth);
    const { tier_level, level, total_xp } = useSelector((state) => state.user);
    const handleLogout = () => {
        dispatch(logout());
        window.location.href = '/';
    };
    return (_jsx("header", { className: "bg-white shadow", children: _jsx("div", { className: "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4", children: _jsxs("div", { className: "flex justify-between items-center", children: [_jsxs("div", { className: "flex items-center gap-8", children: [_jsx("h1", { className: "text-2xl font-bold text-blue-600", children: "Geo-Mobile137" }), _jsxs("nav", { className: "hidden md:flex gap-6", children: [_jsx("a", { href: "/quests", className: "hover:text-blue-600", children: "Quests" }), _jsx("a", { href: "/map", className: "hover:text-blue-600", children: "Map" }), _jsx("a", { href: "/shop", className: "hover:text-blue-600", children: "Shop" }), _jsx("a", { href: "/leaderboard", className: "hover:text-blue-600", children: "Leaderboard" })] })] }), _jsxs("div", { className: "flex items-center gap-4", children: [_jsxs("div", { className: "text-right hidden sm:block", children: [_jsxs("p", { className: "text-sm font-semibold", children: ["Level ", level] }), _jsx("p", { className: "text-xs text-gray-500", children: TIERS[tier_level] })] }), _jsx("button", { onClick: handleLogout, className: "px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700", children: "Logout" })] })] }) }) }));
};
export default Header;
