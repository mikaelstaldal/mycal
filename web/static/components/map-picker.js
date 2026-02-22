import { html } from 'htm/preact';
import { useEffect, useRef, useState } from 'preact/hooks';

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

const DEFAULT_CENTER = { lat: 59.3293, lng: 18.0686 }; // Stockholm

export function MapPicker({ apiKey, latitude, longitude, editing, onCoordinateChange }) {
    const mapRef = useRef(null);
    const mapInstanceRef = useRef(null);
    const markerRef = useRef(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(false);

    if (!apiKey) return null;

    const hasCoords = latitude !== '' && longitude !== '' && latitude != null && longitude != null;

    useEffect(() => {
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
    }, [apiKey, editing]);

    // Update marker when coordinates change externally
    useEffect(() => {
        if (!mapInstanceRef.current || !mapsLoaded) return;
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
        <div class="map-picker" style="margin: 8px 0;">
            ${loading && html`<div style="padding: 12px; color: #666;">Loading map...</div>`}
            <div ref=${mapRef} style="width: 100%; height: ${editing ? '250px' : '200px'}; border-radius: 6px; ${loading ? 'display:none' : ''}" />
        </div>
    `;
}
