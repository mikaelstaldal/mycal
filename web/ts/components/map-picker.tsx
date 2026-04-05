import { h } from 'preact';
import type { VNode } from 'preact';
import { useEffect, useRef, useState } from 'preact/hooks';

declare const google: any;

let mapsLoadPromise: Promise<void> | null = null;
let mapsLoaded = false;

function loadGoogleMaps(apiKey: string): Promise<void> {
    if (mapsLoaded) return Promise.resolve();
    if (mapsLoadPromise) return mapsLoadPromise;
    mapsLoadPromise = new Promise((resolve, reject) => {
        const script = document.createElement('script');
        script.src = `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(apiKey)}`;
        script.async = true;
        script.onload = () => { mapsLoaded = true; resolve(); };
        script.onerror = () => { mapsLoadPromise = null; reject(new Error('Failed to load Google Maps')); };
        document.head.appendChild(script);
    });
    return mapsLoadPromise;
}

let leafletLoadPromise: Promise<void> | null = null;
let leafletLoaded = false;

function loadLeaflet(): Promise<void> {
    if (leafletLoaded) return Promise.resolve();
    if (leafletLoadPromise) return leafletLoadPromise;
    leafletLoadPromise = new Promise((resolve, reject) => {
        const link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = 'vendor/leaflet.css';
        document.head.appendChild(link);

        const script = document.createElement('script');
        script.src = 'vendor/leaflet.js';
        script.async = true;
        script.onload = () => { leafletLoaded = true; resolve(); };
        script.onerror = () => { leafletLoadPromise = null; reject(new Error('Failed to load Leaflet')); };
        document.head.appendChild(script);
    });
    return leafletLoadPromise;
}

const DEFAULT_CENTER = { lat: 59.3293, lng: 18.0686 };

interface MapPickerProps {
    mapProvider: string;
    apiKey?: string;
    latitude?: string | number | null;
    longitude?: string | number | null;
    editing: boolean;
    onCoordinateChange?: (lat: string, lng: string) => void;
}

export function MapPicker({ mapProvider, apiKey, latitude, longitude, editing, onCoordinateChange }: MapPickerProps): VNode | null {
    const mapRef = useRef<HTMLDivElement | null>(null);
    const mapInstanceRef = useRef<any>(null);
    const markerRef = useRef<any>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(false);

    if (!mapProvider || mapProvider === 'none') return null;
    if (mapProvider === 'google' && !/^AIza[A-Za-z0-9_-]{35}$/.test(apiKey || '')) return null;

    const hasCoords = latitude !== '' && longitude !== '' && latitude != null && longitude != null;

    useEffect(() => {
        if (mapProvider !== 'openstreetmap') return;
        let cancelled = false;
        setLoading(true);
        setError(false);

        loadLeaflet().then(() => {
            if (cancelled || !mapRef.current) return;
            setLoading(false);

            const center: [number, number] = hasCoords
                ? [parseFloat(String(latitude)), parseFloat(String(longitude))]
                : [DEFAULT_CENTER.lat, DEFAULT_CENTER.lng];

            const map = L.map(mapRef.current, {
                center,
                zoom: hasCoords ? 15 : 4,
                dragging: editing,
                scrollWheelZoom: editing,
                doubleClickZoom: editing,
                boxZoom: editing,
                keyboard: editing,
                zoomControl: editing,
                touchZoom: editing,
            });
            mapInstanceRef.current = map;

            requestAnimationFrame(() => map.invalidateSize());

            mapRef.current.querySelectorAll<HTMLButtonElement>('button').forEach(b => b.type = 'button');

            L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '&copy; OpenStreetMap contributors',
                maxZoom: 19,
            }).addTo(map);

            if (hasCoords) {
                markerRef.current = L.marker(center).addTo(map);
            }

            if (editing) {
                map.on('click', (e: any) => {
                    const { lat, lng } = e.latlng;
                    if (markerRef.current) {
                        markerRef.current.setLatLng(e.latlng);
                    } else {
                        markerRef.current = L.marker(e.latlng).addTo(map);
                    }
                    if (onCoordinateChange) {
                        onCoordinateChange(lat.toFixed(6), lng.toFixed(6));
                    }
                });
            }
        }).catch(() => {
            if (!cancelled) {
                setLoading(false);
                setError(true);
            }
        });

        return () => {
            cancelled = true;
            if (mapInstanceRef.current) {
                mapInstanceRef.current.remove();
                mapInstanceRef.current = null;
                markerRef.current = null;
            }
        };
    }, [mapProvider, editing]);

    useEffect(() => {
        if (mapProvider !== 'google') return;
        let cancelled = false;
        setLoading(true);
        setError(false);

        loadGoogleMaps(apiKey || '').then(() => {
            if (cancelled || !mapRef.current) return;
            setLoading(false);

            const center = hasCoords
                ? { lat: parseFloat(String(latitude)), lng: parseFloat(String(longitude)) }
                : DEFAULT_CENTER;

            const map = new google.maps.Map(mapRef.current, {
                center,
                zoom: hasCoords ? 15 : 4,
                disableDefaultUI: !editing,
                gestureHandling: editing ? 'auto' : 'cooperative',
            });
            mapInstanceRef.current = map;

            if (hasCoords) {
                markerRef.current = new google.maps.Marker({
                    position: center,
                    map,
                });
            }

            if (editing) {
                map.addListener('click', (e: any) => {
                    const lat = e.latLng.lat();
                    const lng = e.latLng.lng();
                    if (markerRef.current) {
                        markerRef.current.setPosition(e.latLng);
                    } else {
                        markerRef.current = new google.maps.Marker({
                            position: e.latLng,
                            map,
                        });
                    }
                    if (onCoordinateChange) {
                        onCoordinateChange(lat.toFixed(6), lng.toFixed(6));
                    }
                });
            }
        }).catch(() => {
            if (!cancelled) {
                setLoading(false);
                setError(true);
            }
        });

        return () => { cancelled = true; };
    }, [mapProvider, apiKey, editing]);

    useEffect(() => {
        if (mapProvider !== 'openstreetmap' || !mapInstanceRef.current || !leafletLoaded) return;
        if (hasCoords) {
            const pos: [number, number] = [parseFloat(String(latitude)), parseFloat(String(longitude))];
            if (!isNaN(pos[0]) && !isNaN(pos[1])) {
                if (markerRef.current) {
                    markerRef.current.setLatLng(pos);
                } else {
                    markerRef.current = L.marker(pos).addTo(mapInstanceRef.current);
                }
                mapInstanceRef.current.panTo(pos);
            }
        }
    }, [latitude, longitude]);

    useEffect(() => {
        if (mapProvider !== 'google' || !mapInstanceRef.current || !mapsLoaded) return;
        if (hasCoords) {
            const pos = { lat: parseFloat(String(latitude)), lng: parseFloat(String(longitude)) };
            if (!isNaN(pos.lat) && !isNaN(pos.lng)) {
                if (markerRef.current) {
                    markerRef.current.setPosition(pos);
                } else {
                    markerRef.current = new google.maps.Marker({
                        position: pos,
                        map: mapInstanceRef.current,
                    });
                }
                mapInstanceRef.current.panTo(pos);
            }
        }
    }, [latitude, longitude]);

    if (error) return null;

    return (
        <div class="map-picker" style="margin: 8px 0; position: relative; overflow: hidden; border-radius: 6px;">
            {loading && <div style="padding: 12px; color: #666;">Loading map...</div>}
            <div ref={mapRef} style={`width: 100%; height: ${editing ? '250px' : '200px'}; ${loading ? 'display:none' : ''}`} />
        </div>
    );
}
