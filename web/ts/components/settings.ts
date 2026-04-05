import type { VNode } from 'preact';
import { html } from 'htm/preact';
import { useState, useRef, useEffect } from 'preact/hooks';
import { saveConfig } from '../lib/config.js';
import { formatHour } from '../lib/date-utils.js';
import type { AppConfig } from '../types/models.js';

// Google Maps API keys are 39 chars starting with "AIza"
function isValidGoogleMapsApiKey(key: string) {
    return /^AIza[A-Za-z0-9_-]{35}$/.test(key);
}

interface SettingsProps {
    config: AppConfig;
    onConfigChange: (config: AppConfig) => void;
}

export function Settings({ config, onConfigChange }: SettingsProps): VNode | null {
    const [open, setOpen] = useState(false);
    const [apiKeyError, setApiKeyError] = useState('');
    const [pendingApiKey, setPendingApiKey] = useState('');
    // Track dropdown selection locally so we can show the API key field
    // before actually saving 'google' as the provider
    const [selectedProvider, setSelectedProvider] = useState(config.mapProvider || 'none');
    const dialogRef = useRef<HTMLDialogElement | null>(null);

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

    function handleChange(key: string, value: any) {
        const numericKeys = ['dayStartHour', 'weekStartDay'];
        const updated = { ...config, [key]: numericKeys.includes(key) ? Number(value) : value };
        saveConfig(updated);
        onConfigChange(updated);
    }

    function handleMapProviderChange(value: string) {
        setSelectedProvider(value as any);
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

    function handleApiKeyInput(value: string) {
        setPendingApiKey(value);
        if (value === '') {
            setApiKeyError('API key is required for Google Maps');
        } else if (!isValidGoogleMapsApiKey(value)) {
            setApiKeyError('Invalid API key format (expected 39 characters starting with AIza)');
        } else {
            setApiKeyError('');
            // Key is valid — now save both the key and switch the provider
            const updated = { ...config, googleMapsApiKey: value, mapProvider: 'google' as const };
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
                    Default view
                    <select value=${config.defaultView}
                            onChange=${(e: Event) => handleChange('defaultView', (e.target as HTMLSelectElement).value)}>
                        <option value="year">Year</option>
                        <option value="month">Month</option>
                        <option value="week">Week</option>
                        <option value="day">Day</option>
                        <option value="schedule">Schedule</option>
                    </select>
                </label>
                <label>
                    Week starts on
                    <select value=${config.weekStartDay}
                            onChange=${(e: Event) => handleChange('weekStartDay', (e.target as HTMLSelectElement).value)}>
                        <option value="0">Sunday</option>
                        <option value="1">Monday</option>
                        <option value="2">Tuesday</option>
                        <option value="3">Wednesday</option>
                        <option value="4">Thursday</option>
                        <option value="5">Friday</option>
                        <option value="6">Saturday</option>
                    </select>
                </label>
                <label>
                    Week view starts at
                    <select value=${config.dayStartHour}
                            onChange=${(e: Event) => handleChange('dayStartHour', (e.target as HTMLSelectElement).value)}>
                        ${Array.from({ length: 24 }, (_, i) => html`
                            <option value=${i}>${formatHour(i)}</option>
                        `)}
                    </select>
                </label>
                <label>
                    Map provider
                    <select value=${selectedProvider}
                            onChange=${(e: Event) => handleMapProviderChange((e.target as HTMLSelectElement).value)}>
                        <option value="none">None</option>
                        <option value="openstreetmap">OpenStreetMap</option>
                        <option value="google">Google Maps</option>
                    </select>
                </label>
                ${selectedProvider === 'google' && html`
                    <label>
                        Google Maps API key
                        <input type="text" value=${pendingApiKey}
                               onInput=${(e: Event) => handleApiKeyInput((e.target as HTMLInputElement).value)}
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
    ` as VNode;
}
