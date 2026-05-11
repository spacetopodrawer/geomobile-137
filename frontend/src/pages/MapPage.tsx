import React from 'react';
import CadastreMap from '../components/MapUI/CadastreMap';

const MapPage: React.FC = () => {
  // Mock data - in production, this would come from the backend
  const mockEntities = [
    {
      id: 'ent1',
      name: 'Administrative Building',
      lat: 3.8667,
      lng: 11.5167,
      type: 'entity' as const,
    },
    {
      id: 'ent2',
      name: 'School Campus',
      lat: 3.8700,
      lng: 11.5200,
      type: 'entity' as const,
    },
    {
      id: 'ent3',
      name: 'Market District',
      lat: 3.8650,
      lng: 11.5100,
      type: 'entity' as const,
    },
  ];

  const mockQuests = [
    {
      id: 'q1',
      name: 'Building Detective - City Hall',
      lat: 3.8667,
      lng: 11.5167,
      type: 'quest' as const,
    },
    {
      id: 'q2',
      name: 'POI Hunt - Educational Zone',
      lat: 3.8700,
      lng: 11.5200,
      type: 'quest' as const,
    },
  ];

  return (
    <div className="h-screen w-full">
      <CadastreMap
        entities={mockEntities}
        questLocations={mockQuests}
        center={[3.8667, 11.5167]}
        zoom={12}
      />
    </div>
  );
};

export default MapPage;
