import { useEffect, useState } from 'preact/hooks';
import { api, NotFoundError } from '../api/client.js';
import type { components } from '../api/types.js';
import { showConfirm } from '../util/confirm.js';
import { DeviceApproval } from './DeviceApproval.js';

type ApiToken = components['schemas']['APIToken'];

export function ApiTokens() {
    const [tokens, setTokens] = useState<ApiToken[]>([]);
    const [calendars, setCalendars] = useState<{ id: number; name: string }[]>([]);
    const [tokenError, setTokenError] = useState('');
    const [calendarError, setCalendarError] = useState('');
    const [loading, setLoading] = useState(true);
    const [calendarsLoading, setCalendarsLoading] = useState(true);
    const [busy, setBusy] = useState(false);

    async function reload() {
        setTokenError(''); setLoading(true);
        try { setTokens((await api.tokens.list()).items); }
        catch (e) { setTokenError(e instanceof Error ? e.message : 'Could not load API tokens'); }
        finally { setLoading(false); }
    }
    useEffect(() => {
        void reload();
        void api.calendars.list().then(setCalendars).catch(() => setCalendarError('Could not load calendars.')).finally(() => setCalendarsLoading(false));
    }, []);

    async function revoke(token: ApiToken) {
        if (busy) return;
        setBusy(true);
        try {
            if (!await showConfirm(`Revoke "${token.name}"? Any client using it will immediately lose access.`, {
                title: 'Revoke API token', okText: 'Revoke', cancelText: 'Keep', danger: true,
            })) return;
            setTokenError('');
            await api.tokens.revoke(token.slug);
            setTokens(current => current.filter(item => item.slug !== token.slug));
            await reload();
        } catch (e) {
            if (e instanceof NotFoundError) {
                setTokens(current => current.filter(item => item.slug !== token.slug));
                await reload();
            } else setTokenError(e instanceof Error ? e.message : 'Could not revoke token');
        }
        finally { setBusy(false); }
    }

    return <section class="api-tokens-settings">
        <h3>API Tokens</h3>
        <p>Tokens can read their granted calendars until they expire. Use mycal-token login or create to obtain a token; existing token secrets cannot be shown again.</p>
        {calendarError && <p role="alert">{calendarError}</p>}
        <DeviceApproval resources={calendars} scopePrefix="calendar" resourcesLoading={calendarsLoading} onDecision={() => void reload()} />
        <h3>Existing tokens</h3>
        {tokenError && <p role="alert">{tokenError}</p>}
        {loading ? <p role="status">Loading API tokens…</p> : !tokenError && tokens.length === 0 && <p>No API tokens.</p>}
        <ul class="api-token-list">{tokens.map(token => <li key={token.slug}>
            <strong>{token.name}</strong> (<code>{token.slug}</code>) {new Date(token.expires_at).getTime() <= Date.now() ? '(expired)' : ''}
            <p>Expires {new Date(token.expires_at).toLocaleString()} — {token.calendar_ids.map(id => calendars.find(calendar => calendar.id === id)?.name ?? `Calendar ${id}`).join(', ')}</p>
            <button type="button" class="small-btn danger" disabled={busy} onClick={() => void revoke(token)}>Revoke</button>
        </li>)}</ul>
    </section>;
}
