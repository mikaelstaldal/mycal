import { render } from 'preact';
import { useEffect, useState } from 'preact/hooks';
import { api } from './api/client.js';
import { DeviceApproval } from './components/DeviceApproval.js';
import { Toast } from './components/Toast.js';
document.documentElement.setAttribute('data-theme', localStorage.getItem('darkMode') === 'true' ? 'dark' : 'light');

// A separate entry point loads only device consent, without the normal app shell.
function DeviceApprovalPage() {
  const [resources, setResources] = useState<{ id: number; name: string }[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    void api.calendars.list().then(setResources).catch(() => setError('Could not load calendars. Reload this page to try again.')).finally(() => setLoading(false));
  }, []);
  return <main class="device-page-content">
    <h1>MyCal — device login</h1>
    {error && <p role="alert">{error}</p>}
    <DeviceApproval resources={resources} scopePrefix="calendar" closeWhenDone resourcesLoading={loading} />
    <Toast />
  </main>;
}

const root = document.getElementById('app');
if (root) render(<DeviceApprovalPage />, root);
