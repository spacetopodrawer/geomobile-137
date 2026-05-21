import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { purchaseCosmetic, equipCosmetic } from '../../redux/slices/cosmeticSlice';
import { formatCurrency } from '../../utils/formatters';
import { paymentService } from '../../services/payment';
const CosmeticCard = ({ cosmetic }) => {
    const dispatch = useDispatch();
    const { ownedItems } = useSelector((state) => state.cosmetic);
    const [showPurchaseConfirm, setShowPurchaseConfirm] = useState(false);
    const isOwned = ownedItems.includes(cosmetic.cosmetic_id);
    const handlePurchase = async () => {
        try {
            const payment = await paymentService.initiateCosmeticPurchase(cosmetic.cosmetic_id, cosmetic.name, cosmetic.final_price, 'user@example.com', '+237600000000', window.location.origin + '/shop');
            // In production, redirect to Flutterwave/Paytech
            // window.location.href = payment.payment_link;
            // For demo, just mark as purchased
            dispatch(purchaseCosmetic(cosmetic.cosmetic_id));
            setShowPurchaseConfirm(false);
        }
        catch (error) {
            console.error('Purchase failed:', error);
        }
    };
    const handleEquip = () => {
        dispatch(equipCosmetic({ cosmeticId: cosmetic.cosmetic_id, category: cosmetic.category }));
    };
    return (_jsxs("div", { className: "bg-white rounded-lg shadow hover:shadow-lg transition-shadow overflow-hidden", children: [_jsx("div", { className: "bg-gradient-to-br from-purple-200 to-pink-200 h-48 flex items-center justify-center", children: _jsx("div", { className: "text-4xl", children: "\u2728" }) }), _jsxs("div", { className: "p-4 space-y-3", children: [_jsx("h3", { className: "text-lg font-bold text-gray-900", children: cosmetic.name }), _jsx("p", { className: "text-sm text-gray-600", children: cosmetic.description }), _jsx("div", { className: "inline-block", children: _jsx("span", { className: "px-2 py-1 bg-purple-100 text-purple-800 text-xs font-semibold rounded", children: cosmetic.category }) }), _jsx("div", { className: "border-t border-gray-200 pt-3", children: cosmetic.discount_percentage > 0 ? (_jsxs("div", { children: [_jsx("p", { className: "text-sm text-gray-500 line-through", children: formatCurrency(cosmetic.original_price) }), _jsxs("div", { className: "flex justify-between items-center", children: [_jsx("p", { className: "text-xl font-bold text-green-600", children: formatCurrency(cosmetic.final_price) }), _jsxs("span", { className: "px-2 py-1 bg-red-100 text-red-700 text-xs font-bold rounded", children: ["-", Math.round(cosmetic.discount_percentage * 100), "%"] })] })] })) : (_jsx("p", { className: "text-xl font-bold text-gray-900", children: formatCurrency(cosmetic.final_price) })) }), cosmetic.tier_requirement > 0 && (_jsxs("p", { className: "text-xs text-gray-500", children: ["Requires: Tier ", cosmetic.tier_requirement] }))] }), _jsx("div", { className: "px-4 py-3 bg-gray-50 border-t border-gray-100 space-y-2", children: !isOwned ? (_jsx("button", { onClick: () => setShowPurchaseConfirm(true), className: "w-full px-4 py-2 bg-purple-600 text-white rounded font-semibold hover:bg-purple-700 transition-colors", children: "Buy Now" })) : (_jsxs(_Fragment, { children: [_jsx("button", { onClick: handleEquip, className: "w-full px-4 py-2 bg-blue-600 text-white rounded font-semibold hover:bg-blue-700 transition-colors", children: "Equip" }), _jsx("p", { className: "text-center text-xs text-green-600 font-semibold", children: "\u2713 Owned" })] })) }), showPurchaseConfirm && (_jsx("div", { className: "fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50", children: _jsxs("div", { className: "bg-white rounded-lg shadow-lg p-6 max-w-sm", children: [_jsx("h3", { className: "text-lg font-bold text-gray-900 mb-3", children: "Purchase Cosmetic?" }), _jsx("p", { className: "text-gray-600 mb-2", children: cosmetic.name }), _jsx("p", { className: "text-2xl font-bold text-purple-600 mb-6", children: formatCurrency(cosmetic.final_price) }), _jsxs("div", { className: "flex gap-3", children: [_jsx("button", { onClick: () => setShowPurchaseConfirm(false), className: "flex-1 px-4 py-2 bg-gray-200 text-gray-900 rounded hover:bg-gray-300 font-semibold", children: "Cancel" }), _jsx("button", { onClick: handlePurchase, className: "flex-1 px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700 font-semibold", children: "Purchase" })] })] }) }))] }));
};
export default CosmeticCard;
