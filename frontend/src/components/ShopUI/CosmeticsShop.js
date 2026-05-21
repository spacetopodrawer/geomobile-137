import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { cosmeticsService } from '../../services/cosmetics';
import { setCosmeticsList, setSelectedCategory, setLoading, setError } from '../../redux/slices/cosmeticSlice';
import { COSMETIC_CATEGORIES } from '../../utils/constants';
import { formatCurrency } from '../../utils/formatters';
import Loading from '../Common/Loading';
import CosmeticCard from './CosmeticCard';
const CosmeticsShop = () => {
    const dispatch = useDispatch();
    const { items, selectedCategory, loading, error } = useSelector((state) => state.cosmetic);
    const { tier_level } = useSelector((state) => state.user);
    useEffect(() => {
        const fetchCosmetics = async () => {
            dispatch(setLoading(true));
            try {
                const cosmetics = await cosmeticsService.getCosmeticsList();
                dispatch(setCosmeticsList(cosmetics));
            }
            catch (err) {
                dispatch(setError(err instanceof Error ? err.message : 'Failed to fetch cosmetics'));
            }
            finally {
                dispatch(setLoading(false));
            }
        };
        fetchCosmetics();
    }, [dispatch]);
    const filteredItems = selectedCategory === 'All Items'
        ? items
        : items.filter(item => item.category === selectedCategory);
    const totalPrice = filteredItems.reduce((sum, item) => sum + item.final_price, 0);
    if (loading)
        return _jsx(Loading, { message: "Loading cosmetics..." });
    return (_jsxs("div", { className: "space-y-6 py-8", children: [_jsxs("div", { className: "bg-gradient-to-r from-purple-600 to-pink-600 text-white rounded-lg shadow p-6", children: [_jsx("h1", { className: "text-3xl font-bold mb-2", children: "Cosmetics Shop" }), _jsx("p", { className: "text-purple-100", children: "Express yourself with exclusive cosmetics" })] }), error && (_jsx("div", { className: "bg-red-50 border border-red-200 rounded-lg p-4 text-red-700", children: error })), _jsx("div", { className: "bg-white rounded-lg shadow", children: _jsx("div", { className: "flex flex-wrap border-b border-gray-200", children: COSMETIC_CATEGORIES.map(category => (_jsx("button", { onClick: () => dispatch(setSelectedCategory(category)), className: `px-6 py-4 font-medium border-b-2 transition-colors ${selectedCategory === category
                            ? 'border-purple-600 text-purple-600'
                            : 'border-transparent text-gray-600 hover:text-gray-900'}`, children: category }, category))) }) }), tier_level > 0 && (_jsxs("div", { className: "bg-blue-50 border border-blue-200 rounded-lg p-4", children: [_jsx("p", { className: "text-blue-900 font-semibold", children: "\u2728 You have tier-based cosmetic discounts active!" }), _jsx("p", { className: "text-blue-700 text-sm mt-1", children: "Cosmetic prices are reduced based on your subscription tier." })] })), filteredItems.length === 0 ? (_jsx("div", { className: "bg-white rounded-lg shadow p-12 text-center", children: _jsx("p", { className: "text-gray-500", children: "No cosmetics in this category yet." }) })) : (_jsx("div", { className: "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6", children: filteredItems.map(cosmetic => (_jsx(CosmeticCard, { cosmetic: cosmetic }, cosmetic.cosmetic_id))) })), _jsx("div", { className: "bg-white rounded-lg shadow p-6", children: _jsxs("div", { className: "flex justify-between items-center", children: [_jsxs("div", { children: [_jsx("p", { className: "text-gray-600", children: "Total Selected Items" }), _jsx("p", { className: "text-2xl font-bold text-gray-900", children: filteredItems.length })] }), _jsxs("div", { className: "text-right", children: [_jsx("p", { className: "text-gray-600", children: "Subtotal" }), _jsx("p", { className: "text-2xl font-bold text-green-600", children: formatCurrency(totalPrice) })] })] }) })] }));
};
export default CosmeticsShop;
