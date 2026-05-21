import { jsx as _jsx } from "react/jsx-runtime";
import CadastreMap from '../components/MapUI/CadastreMap';
const MapPage = () => {
    // Mock data - in production, this would come from the backend
    const mockEntities = [
        {
            id: 'ent1',
            name: 'Administrative Building',
            lat: 3.8667,
            lng: 11.5167,
            type: 'entity',
        },
        {
            id: 'ent2',
            name: 'School Campus',
            lat: 3.8700,
            lng: 11.5200,
            type: 'entity',
        },
        {
            id: 'ent3',
            name: 'Market District',
            lat: 3.8650,
            lng: 11.5100,
            type: 'entity',
        },
    ];
    const mockQuests = [
        {
            id: 'q1',
            name: 'Building Detective - City Hall',
            lat: 3.8667,
            lng: 11.5167,
            type: 'quest',
        },
        {
            id: 'q2',
            name: 'POI Hunt - Educational Zone',
            lat: 3.8700,
            lng: 11.5200,
            type: 'quest',
        },
    ];
    return (_jsx("div", { className: "h-screen w-full", children: _jsx(CadastreMap, { entities: mockEntities, questLocations: mockQuests, center: [3.8667, 11.5167], zoom: 12 }) }));
};
export default MapPage;
