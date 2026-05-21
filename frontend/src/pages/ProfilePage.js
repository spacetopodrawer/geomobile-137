import { jsx as _jsx } from "react/jsx-runtime";
import UserDashboard from '../components/UserDashboard/UserDashboard';
const ProfilePage = () => {
    return (_jsx("div", { className: "py-8", children: _jsx(UserDashboard, {}) }));
};
export default ProfilePage;
