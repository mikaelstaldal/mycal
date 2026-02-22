import { html } from 'htm/preact';
import { useState, useRef, useEffect } from 'preact/hooks';
import { saveConfig } from '../lib/config.js';

// Google Maps API keys are 39 chars starting with "AIza"
function isValidGoogleMapsApiKey(key) {
    return /^AIza[A-Za-z0-9_-]{35}$/.test(key);
}

export function Settings({ config, onConfigChange }) {
    const [open, setOpen] = useState(false);
    const [apiKeyError, setApiKeyError] = useState('');
    const [pendingApiKey, setPendingApiKey] = useState('');
    // Track dropdown selection locally so we can show the API key field
    // before actually saving 'google' as the provider
    const [selectedProvider, setSelectedProvider] = useState(config.mapProvider || 'none');
    const dialogRef = useRef(null);

    useEffect(() => {
        if (open && dialogRef.current && !dialogRef.current.open) {
            dialogRef.current.showModal();
        }
    }, [open]);

    useEffect(() => {
        if (open) {
            setPendingApiKey(config.googleMapsApiKey || '');
            setSelectedProvider(config.mapProvider || 'none');
            setApiKeyError('');
        }
    }, [open]);

    function handleClose() {
        setOpen(false);
    }

    function handleChange(key, value) {
        const numericKeys = ['weekStartDay', 'dayStartHour'];
        const updated = { ...config, [key]: numericKeys.includes(key) ? Number(value) : value };
        saveConfig(updated);
        onConfigChange(updated);
    }

    function handleMapProviderChange(value) {
        setSelectedProvider(value);
        if (value === 'google') {
            setPendingApiKey(config.googleMapsApiKey || '');
            if (isValidGoogleMapsApiKey(config.googleMapsApiKey)) {
                setApiKeyError('');
                handleChange('mapProvider', 'google');
            } else {
                setApiKeyError('Enter a valid API key to enable Google Maps');
                // Don't save 'google' as provider yet — keep current provider
            }
        } else {
            setApiKeyError('');
            handleChange('mapProvider', value);
        }
    }

    function handleApiKeyInput(value) {
        setPendingApiKey(value);
        if (value === '') {
            setApiKeyError('API key is required for Google Maps');
        } else if (!isValidGoogleMapsApiKey(value)) {
            setApiKeyError('Invalid API key format (expected 39 characters starting with AIza)');
        } else {
            setApiKeyError('');
            // Key is valid — now save both the key and switch the provider
            const updated = { ...config, googleMapsApiKey: value, mapProvider: 'google' };
            saveConfig(updated);
            onConfigChange(updated);
            return;
        }
        // Save the key value for persistence, but don't switch provider
        handleChange('googleMapsApiKey', value);
    }

    return html`
        <button class="settings-btn" onClick=${() => setOpen(true)} title="Settings">
            \u2699
        </button>
        ${open && html`
            <dialog ref=${dialogRef} class="settings-dialog" onClose=${handleClose}>
                <div class="dialog-header">
                    <h2>Settings</h2>
                    <button class="close-btn" onClick=${handleClose}>\u00d7</button>
                </div>
                <label>
                    Week starts on
                    <select value=${config.weekStartDay}
                            onChange=${e => handleChange('weekStartDay', e.target.value)}>
                        <option value="1">Monday</option>
                        <option value="0">Sunday</option>
                    </select>
                </label>
                <label>
                    Clock format
                    <select value=${config.clockFormat}
                            onChange=${e => handleChange('clockFormat', e.target.value)}>
                        <option value="24h">24-hour</option>
                        <option value="12h">12-hour</option>
                    </select>
                </label>
                <label>
                    Date format
                    <select value=${config.dateFormat}
                            onChange=${e => handleChange('dateFormat', e.target.value)}>
                        <option value="yyyy-MM-dd">yyyy-MM-dd</option>
                        <option value="MM/dd/yyyy">MM/dd/yyyy</option>
                        <option value="dd/MM/yyyy">dd/MM/yyyy</option>
                    </select>
                </label>
                <label>
                    Default view
                    <select value=${config.defaultView}
                            onChange=${e => handleChange('defaultView', e.target.value)}>
                        <option value="month">Month</option>
                        <option value="week">Week</option>
                    </select>
                </label>
                <label>
                    Week view starts at
                    <select value=${config.dayStartHour}
                            onChange=${e => handleChange('dayStartHour', e.target.value)}>
                        ${Array.from({ length: 24 }, (_, i) => html`
                            <option value=${i}>${config.clockFormat === '12h'
                                ? (i === 0 ? '12 AM' : i < 12 ? i + ' AM' : i === 12 ? '12 PM' : (i - 12) + ' PM')
                                : String(i).padStart(2, '0') + ':00'}</option>
                        `)}
                    </select>
                </label>
                <label>
                    Map provider
                    <select value=${selectedProvider}
                            onChange=${e => handleMapProviderChange(e.target.value)}>
                        <option value="none">None</option>
                        <option value="openstreetmap">OpenStreetMap</option>
                        <option value="google">Google Maps</option>
                    </select>
                </label>
                ${selectedProvider === 'google' && html`
                    <label>
                        Google Maps API key
                        <input type="text" value=${pendingApiKey}
                               onInput=${e => handleApiKeyInput(e.target.value)}
                               placeholder="AIza..." />
                    </label>
                    ${apiKeyError && html`
                        <div style="color: #c62828; font-size: 0.8rem; margin: -8px 0 8px;">
                            ${apiKeyError}
                        </div>
                    `}
                `}
                <div class="dialog-actions">
                    <button onClick=${handleClose}>Close</button>
                </div>
            </dialog>
        `}
    `;
}
