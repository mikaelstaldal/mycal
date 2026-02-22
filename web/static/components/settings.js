import { html } from 'htm/preact';
import { useState, useRef, useEffect } from 'preact/hooks';
import { saveConfig } from '../lib/config.js';

export function Settings({ config, onConfigChange }) {
    const [open, setOpen] = useState(false);
    const dialogRef = useRef(null);

    useEffect(() => {
        if (open && dialogRef.current && !dialogRef.current.open) {
            dialogRef.current.showModal();
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
                    Google Maps API key
                    <input type="text" value=${config.googleMapsApiKey || ''}
                           onInput=${e => handleChange('googleMapsApiKey', e.target.value)}
                           placeholder="Leave empty to disable maps" />
                </label>
                <div class="dialog-actions">
                    <button onClick=${handleClose}>Close</button>
                </div>
            </dialog>
        `}
    `;
}
