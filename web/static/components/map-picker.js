import { html } from 'htm/preact';
import { useEffect, useRef, useState } from 'preact/hooks';

// Google Maps loader
let mapsLoadPromise = null;
let mapsLoaded = false;

function loadGoogleMaps(apiKey) {
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

// Leaflet loader
let leafletLoadPromise = null;
let leafletLoaded = false;

function loadLeaflet() {
    if (leafletLoaded) return Promise.resolve();
    if (leafletLoadPromise) return leafletLoadPromise;
    leafletLoadPromise = new Promise((resolve, reject) => {
        // Load CSS
        const link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css';
        link.integrity = 'sha384-sHL9NAb7lN7rfvG5lfHpm643Xkcjzp4jFvuavGOndn6pjVqS6ny56CAt3nsEVT4H';
        link.crossOrigin = 'anonymous';
        document.head.appendChild(link);

        // Load JS
        const script = document.createElement('script');
        script.src = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js';
        script.integrity = 'sha384-cxOPjt7s7Iz04uaHJceBmS+qpjv2JkIHNVcuOrM+YHwZOmJGBXI00mdUXEq65HTH';
        script.crossOrigin = 'anonymous';
        script.async = true;
        script.onload = () => { leafletLoaded = true; resolve(); };
        script.onerror = () => { leafletLoadPromise = null; reject(new Error('Failed to load Leaflet')); };
        document.head.appendChild(script);
    });
    return leafletLoadPromise;
}

const DEFAULT_CENTER = { lat: 59.3293, lng: 18.0686 }; // Stockholm

export function MapPicker({ mapProvider, apiKey, latitude, longitude, editing, onCoordinateChange }) {
    const mapRef = useRef(null);
    const mapInstanceRef = useRef(null);
    const markerRef = useRef(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(false);

    if (!mapProvider || mapProvider === 'none') return null;
    if (mapProvider === 'google' && !/^AIza[A-Za-z0-9_-]{35}$/.test(apiKey)) return null;

    const hasCoords = latitude !== '' && longitude !== '' && latitude != null && longitude != null;

    // OpenStreetMap via Leaflet
    useEffect(() => {
        if (mapProvider !== 'openstreetmap') return;
        let cancelled = false;
        setLoading(true);
        setError(false);

        loadLeaflet().then(() => {
            if (cancelled || !mapRef.current) return;
            setLoading(false);

            const center = hasCoords
                ? [parseFloat(latitude), parseFloat(longitude)]
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

            // Leaflet needs a size recalc after the dialog finishes layout
            requestAnimationFrame(() => map.invalidateSize());

            // Prevent Leaflet's buttons from submitting the parent form
            mapRef.current.querySelectorAll('button').forEach(b => b.type = 'button');

            L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '&copy; OpenStreetMap contributors',
                maxZoom: 19,
            }).addTo(map);

            if (hasCoords) {
                markerRef.current = L.marker(center).addTo(map);
            }

            if (editing) {
                map.on('click', (e) => {
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

    // Google Maps
    useEffect(() => {
        if (mapProvider !== 'google') return;
        let cancelled = false;
        setLoading(true);
        setError(false);

        loadGoogleMaps(apiKey).then(() => {
            if (cancelled || !mapRef.current) return;
            setLoading(false);

            const center = hasCoords
                ? { lat: parseFloat(latitude), lng: parseFloat(longitude) }
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
                map.addListener('click', (e) => {
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

    // Update marker when coordinates change externally (Leaflet)
    useEffect(() => {
        if (mapProvider !== 'openstreetmap' || !mapInstanceRef.current || !leafletLoaded) return;
        if (hasCoords) {
            const pos = [parseFloat(latitude), parseFloat(longitude)];
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

    // Update marker when coordinates change externally (Google)
    useEffect(() => {
        if (mapProvider !== 'google' || !mapInstanceRef.current || !mapsLoaded) return;
        if (hasCoords) {
            const pos = { lat: parseFloat(latitude), lng: parseFloat(longitude) };
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

    return html`
        <div class="map-picker" style="margin: 8px 0; position: relative; overflow: hidden; border-radius: 6px;">
            ${loading && html`<div style="padding: 12px; color: #666;">Loading map...</div>`}
            <div ref=${mapRef} style="width: 100%; height: ${editing ? '250px' : '200px'}; ${loading ? 'display:none' : ''}" />
        </div>
    `;
}
