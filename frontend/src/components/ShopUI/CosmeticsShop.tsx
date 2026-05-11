import React, { useEffect, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { cosmeticsService } from '../../services/cosmetics';
import { setCosmeticsList, setSelectedCategory, setLoading, setError } from '../../redux/slices/cosmeticSlice';
import { COSMETIC_CATEGORIES } from '../../utils/constants';
import { formatCurrency } from '../../utils/formatters';
import Loading from '../Common/Loading';
import CosmeticCard from './CosmeticCard';

const CosmeticsShop: React.FC = () => {
  const dispatch = useDispatch();
  const { items, selectedCategory, loading, error } = useSelector((state: RootState) => state.cosmetic);
  const { tier_level } = useSelector((state: RootState) => state.user);

  useEffect(() => {
    const fetchCosmetics = async () => {
      dispatch(setLoading(true));
      try {
        const cosmetics = await cosmeticsService.getCosmeticsList();
        dispatch(setCosmeticsList(cosmetics));
      } catch (err) {
        dispatch(setError(err instanceof Error ? err.message : 'Failed to fetch cosmetics'));
      } finally {
        dispatch(setLoading(false));
      }
    };

    fetchCosmetics();
  }, [dispatch]);

  const filteredItems = selectedCategory === 'All Items'
    ? items
    : items.filter(item => item.category === selectedCategory);

  const totalPrice = filteredItems.reduce((sum, item) => sum + item.final_price, 0);

  if (loading) return <Loading message="Loading cosmetics..." />;

  return (
    <div className="space-y-6 py-8">
      {/* Header */}
      <div className="bg-gradient-to-r from-purple-600 to-pink-600 text-white rounded-lg shadow p-6">
        <h1 className="text-3xl font-bold mb-2">Cosmetics Shop</h1>
        <p className="text-purple-100">Express yourself with exclusive cosmetics</p>
      </div>

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {error}
        </div>
      )}

      {/* Category Tabs */}
      <div className="bg-white rounded-lg shadow">
        <div className="flex flex-wrap border-b border-gray-200">
          {COSMETIC_CATEGORIES.map(category => (
            <button
              key={category}
              onClick={() => dispatch(setSelectedCategory(category))}
              className={`px-6 py-4 font-medium border-b-2 transition-colors ${
                selectedCategory === category
                  ? 'border-purple-600 text-purple-600'
                  : 'border-transparent text-gray-600 hover:text-gray-900'
              }`}
            >
              {category}
            </button>
          ))}
        </div>
      </div>

      {/* Tier Discount Info */}
      {tier_level > 0 && (
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <p className="text-blue-900 font-semibold">
            ✨ You have tier-based cosmetic discounts active!
          </p>
          <p className="text-blue-700 text-sm mt-1">
            Cosmetic prices are reduced based on your subscription tier.
          </p>
        </div>
      )}

      {/* Items Grid */}
      {filteredItems.length === 0 ? (
        <div className="bg-white rounded-lg shadow p-12 text-center">
          <p className="text-gray-500">No cosmetics in this category yet.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredItems.map(cosmetic => (
            <CosmeticCard key={cosmetic.cosmetic_id} cosmetic={cosmetic} />
          ))}
        </div>
      )}

      {/* Cart Summary */}
      <div className="bg-white rounded-lg shadow p-6">
        <div className="flex justify-between items-center">
          <div>
            <p className="text-gray-600">Total Selected Items</p>
            <p className="text-2xl font-bold text-gray-900">{filteredItems.length}</p>
          </div>
          <div className="text-right">
            <p className="text-gray-600">Subtotal</p>
            <p className="text-2xl font-bold text-green-600">{formatCurrency(totalPrice)}</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default CosmeticsShop;
