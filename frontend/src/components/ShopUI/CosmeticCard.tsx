import React, { useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { purchaseCosmetic, equipCosmetic } from '../../redux/slices/cosmeticSlice';
import { formatCurrency } from '../../utils/formatters';
import { paymentService } from '../../services/payment';
import type { Cosmetic } from '../../services/cosmetics';

interface CosmeticCardProps {
  cosmetic: Cosmetic;
}

const CosmeticCard: React.FC<CosmeticCardProps> = ({ cosmetic }) => {
  const dispatch = useDispatch();
  const { ownedItems } = useSelector((state: RootState) => state.cosmetic);
  const [showPurchaseConfirm, setShowPurchaseConfirm] = useState(false);

  const isOwned = ownedItems.includes(cosmetic.cosmetic_id);

  const handlePurchase = async () => {
    try {
      const payment = await paymentService.initiateCosmeticPurchase(
        cosmetic.cosmetic_id,
        cosmetic.name,
        cosmetic.final_price,
        'user@example.com',
        '+237600000000',
        window.location.origin + '/shop'
      );

      // In production, redirect to Flutterwave/Paytech
      // window.location.href = payment.payment_link;

      // For demo, just mark as purchased
      dispatch(purchaseCosmetic(cosmetic.cosmetic_id));
      setShowPurchaseConfirm(false);
    } catch (error) {
      console.error('Purchase failed:', error);
    }
  };

  const handleEquip = () => {
    dispatch(equipCosmetic({ cosmeticId: cosmetic.cosmetic_id, category: cosmetic.category }));
  };

  return (
    <div className="bg-white rounded-lg shadow hover:shadow-lg transition-shadow overflow-hidden">
      {/* Image */}
      <div className="bg-gradient-to-br from-purple-200 to-pink-200 h-48 flex items-center justify-center">
        <div className="text-4xl">✨</div>
      </div>

      {/* Content */}
      <div className="p-4 space-y-3">
        {/* Name */}
        <h3 className="text-lg font-bold text-gray-900">{cosmetic.name}</h3>
        <p className="text-sm text-gray-600">{cosmetic.description}</p>

        {/* Category Badge */}
        <div className="inline-block">
          <span className="px-2 py-1 bg-purple-100 text-purple-800 text-xs font-semibold rounded">
            {cosmetic.category}
          </span>
        </div>

        {/* Pricing */}
        <div className="border-t border-gray-200 pt-3">
          {cosmetic.discount_percentage > 0 ? (
            <div>
              <p className="text-sm text-gray-500 line-through">
                {formatCurrency(cosmetic.original_price)}
              </p>
              <div className="flex justify-between items-center">
                <p className="text-xl font-bold text-green-600">
                  {formatCurrency(cosmetic.final_price)}
                </p>
                <span className="px-2 py-1 bg-red-100 text-red-700 text-xs font-bold rounded">
                  -{Math.round(cosmetic.discount_percentage * 100)}%
                </span>
              </div>
            </div>
          ) : (
            <p className="text-xl font-bold text-gray-900">
              {formatCurrency(cosmetic.final_price)}
            </p>
          )}
        </div>

        {/* Tier Requirement */}
        {cosmetic.tier_requirement > 0 && (
          <p className="text-xs text-gray-500">
            Requires: Tier {cosmetic.tier_requirement}
          </p>
        )}
      </div>

      {/* Actions */}
      <div className="px-4 py-3 bg-gray-50 border-t border-gray-100 space-y-2">
        {!isOwned ? (
          <button
            onClick={() => setShowPurchaseConfirm(true)}
            className="w-full px-4 py-2 bg-purple-600 text-white rounded font-semibold hover:bg-purple-700 transition-colors"
          >
            Buy Now
          </button>
        ) : (
          <>
            <button
              onClick={handleEquip}
              className="w-full px-4 py-2 bg-blue-600 text-white rounded font-semibold hover:bg-blue-700 transition-colors"
            >
              Equip
            </button>
            <p className="text-center text-xs text-green-600 font-semibold">✓ Owned</p>
          </>
        )}
      </div>

      {/* Purchase Confirmation */}
      {showPurchaseConfirm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-lg p-6 max-w-sm">
            <h3 className="text-lg font-bold text-gray-900 mb-3">Purchase Cosmetic?</h3>
            <p className="text-gray-600 mb-2">{cosmetic.name}</p>
            <p className="text-2xl font-bold text-purple-600 mb-6">
              {formatCurrency(cosmetic.final_price)}
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setShowPurchaseConfirm(false)}
                className="flex-1 px-4 py-2 bg-gray-200 text-gray-900 rounded hover:bg-gray-300 font-semibold"
              >
                Cancel
              </button>
              <button
                onClick={handlePurchase}
                className="flex-1 px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700 font-semibold"
              >
                Purchase
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default CosmeticCard;
