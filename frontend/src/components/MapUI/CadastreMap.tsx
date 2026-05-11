import React, { useEffect } from 'react';
import { MapContainer, TileLayer, Popup, Marker } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

// Fix Leaflet default icon
delete (L.Icon.Default.prototype as any)._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-icon-2x.png',
  iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-icon.png',
  shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-shadow.png',
});

interface MapEntity {
  id: string;
  name: string;
  lat: number;
  lng: number;
  type: 'entity' | 'quest' | 'poi';
}

interface CadastreMapProps {
  entities?: MapEntity[];
  questLocations?: MapEntity[];
  center?: [number, number];
  zoom?: number;
}

const CadastreMap: React.FC<CadastreMapProps> = ({
  entities = [],
  questLocations = [],
  center = [3.8667, 11.5167], // Yaoundé, Cameroon
  zoom = 10,
}) => {
  const allMarkers = [...entities, ...questLocations];

  const getMarkerColor = (type: string) => {
    switch (type) {
      case 'quest':
        return '#FFAA00';
      case 'poi':
        return '#0084FF';
      default:
        return '#EF4444';
    }
  };

  const getMarkerIcon = (type: string) => {
    return L.divIcon({
      html: `<div style="background-color: ${getMarkerColor(type)}; border-radius: 50%; width: 24px; height: 24px; border: 3px solid white; box-shadow: 0 2px 4px rgba(0,0,0,0.3);" />`,
      iconSize: [30, 30],
      className: 'custom-marker',
    });
  };

  return (
    <div className="h-screen w-full bg-gray-100">
      <MapContainer
        center={center}
        zoom={zoom}
        style={{ height: '100%', width: '100%' }}
        className="rounded-lg"
      >
        {/* Base Map Tiles */}
        <TileLayer
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          maxZoom={19}
        />

        {/* Entities */}
        {entities.map(entity => (
          <Marker
            key={entity.id}
            position={[entity.lat, entity.lng]}
            icon={getMarkerIcon('entity')}
          >
            <Popup>
              <div className="text-sm">
                <p className="font-bold">{entity.name}</p>
                <p className="text-gray-600">Cadastral Entity</p>
                <p className="text-xs text-gray-500 mt-1">
                  {entity.lat.toFixed(4)}, {entity.lng.toFixed(4)}
                </p>
              </div>
            </Popup>
          </Marker>
        ))}

        {/* Quest Locations */}
        {questLocations.map(location => (
          <Marker
            key={location.id}
            position={[location.lat, location.lng]}
            icon={getMarkerIcon('quest')}
          >
            <Popup>
              <div className="text-sm">
                <p className="font-bold text-orange-600">{location.name}</p>
                <p className="text-gray-600">Quest Location</p>
                <p className="text-xs text-gray-500 mt-1">
                  {location.lat.toFixed(4)}, {location.lng.toFixed(4)}
                </p>
              </div>
            </Popup>
          </Marker>
        ))}
      </MapContainer>

      {/* Legend */}
      <div className="absolute bottom-4 left-4 bg-white rounded-lg shadow p-4 z-40 max-w-xs">
        <h3 className="font-bold text-gray-900 mb-3">Map Legend</h3>
        <div className="space-y-2 text-sm">
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded-full bg-red-500 border-2 border-white" />
            <span className="text-gray-700">Cadastral Entities</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded-full bg-orange-400 border-2 border-white" />
            <span className="text-gray-700">Quest Locations</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded-full bg-blue-600 border-2 border-white" />
            <span className="text-gray-700">Points of Interest</span>
          </div>
        </div>
      </div>

      {/* Info Panel */}
      <div className="absolute top-4 right-4 bg-white rounded-lg shadow p-4 z-40 max-w-xs">
        <h3 className="font-bold text-gray-900 mb-2">Map Information</h3>
        <div className="text-sm text-gray-600 space-y-1">
          <p>Entities: {entities.length}</p>
          <p>Quests: {questLocations.length}</p>
          <p>Total markers: {allMarkers.length}</p>
        </div>
      </div>
    </div>
  );
};

export default CadastreMap;
