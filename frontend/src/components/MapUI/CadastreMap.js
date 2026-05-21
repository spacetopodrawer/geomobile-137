import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { MapContainer, TileLayer, Popup, Marker } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
// Fix Leaflet default icon
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
    iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-icon-2x.png',
    iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-icon.png',
    shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-shadow.png',
});
const CadastreMap = ({ entities = [], questLocations = [], center = [3.8667, 11.5167], // Yaoundé, Cameroon
zoom = 10, }) => {
    const allMarkers = [...entities, ...questLocations];
    const getMarkerColor = (type) => {
        switch (type) {
            case 'quest':
                return '#FFAA00';
            case 'poi':
                return '#0084FF';
            default:
                return '#EF4444';
        }
    };
    const getMarkerIcon = (type) => {
        return L.divIcon({
            html: `<div style="background-color: ${getMarkerColor(type)}; border-radius: 50%; width: 24px; height: 24px; border: 3px solid white; box-shadow: 0 2px 4px rgba(0,0,0,0.3);" />`,
            iconSize: [30, 30],
            className: 'custom-marker',
        });
    };
    return (_jsxs("div", { className: "h-screen w-full bg-gray-100", children: [_jsxs(MapContainer, { center: center, zoom: zoom, style: { height: '100%', width: '100%' }, className: "rounded-lg", children: [_jsx(TileLayer, { url: "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", attribution: '\u00A9 <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors', maxZoom: 19 }), entities.map(entity => (_jsx(Marker, { position: [entity.lat, entity.lng], icon: getMarkerIcon('entity'), children: _jsx(Popup, { children: _jsxs("div", { className: "text-sm", children: [_jsx("p", { className: "font-bold", children: entity.name }), _jsx("p", { className: "text-gray-600", children: "Cadastral Entity" }), _jsxs("p", { className: "text-xs text-gray-500 mt-1", children: [entity.lat.toFixed(4), ", ", entity.lng.toFixed(4)] })] }) }) }, entity.id))), questLocations.map(location => (_jsx(Marker, { position: [location.lat, location.lng], icon: getMarkerIcon('quest'), children: _jsx(Popup, { children: _jsxs("div", { className: "text-sm", children: [_jsx("p", { className: "font-bold text-orange-600", children: location.name }), _jsx("p", { className: "text-gray-600", children: "Quest Location" }), _jsxs("p", { className: "text-xs text-gray-500 mt-1", children: [location.lat.toFixed(4), ", ", location.lng.toFixed(4)] })] }) }) }, location.id)))] }), _jsxs("div", { className: "absolute bottom-4 left-4 bg-white rounded-lg shadow p-4 z-40 max-w-xs", children: [_jsx("h3", { className: "font-bold text-gray-900 mb-3", children: "Map Legend" }), _jsxs("div", { className: "space-y-2 text-sm", children: [_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("div", { className: "w-4 h-4 rounded-full bg-red-500 border-2 border-white" }), _jsx("span", { className: "text-gray-700", children: "Cadastral Entities" })] }), _jsxs("div", { className: "flex items-center gap-2", children: [_jsx("div", { className: "w-4 h-4 rounded-full bg-orange-400 border-2 border-white" }), _jsx("span", { className: "text-gray-700", children: "Quest Locations" })] }), _jsxs("div", { className: "flex items-center gap-2", children: [_jsx("div", { className: "w-4 h-4 rounded-full bg-blue-600 border-2 border-white" }), _jsx("span", { className: "text-gray-700", children: "Points of Interest" })] })] })] }), _jsxs("div", { className: "absolute top-4 right-4 bg-white rounded-lg shadow p-4 z-40 max-w-xs", children: [_jsx("h3", { className: "font-bold text-gray-900 mb-2", children: "Map Information" }), _jsxs("div", { className: "text-sm text-gray-600 space-y-1", children: [_jsxs("p", { children: ["Entities: ", entities.length] }), _jsxs("p", { children: ["Quests: ", questLocations.length] }), _jsxs("p", { children: ["Total markers: ", allMarkers.length] })] })] })] }));
};
export default CadastreMap;
