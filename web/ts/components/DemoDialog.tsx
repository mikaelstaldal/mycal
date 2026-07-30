import type { VNode } from 'preact';
import { useRef, useEffect } from 'preact/hooks';

// The demo's one piece of demo-only UI: a first-visit notice explaining that
// there is no server behind this calendar, and a badge that keeps saying so and
// brings the notice back. Everything else in the app is the real thing (see
// web/ts/demo/ for the backend that stands in for the Go server).

/** Where the dismissal is remembered. Per browser profile, like the data. */
const NOTICE_SEEN_KEY = 'mycal-demo:notice-seen';

/** Whether the first-visit notice has already been dismissed. */
export function demoNoticeSeen(): boolean {
    try {
        return localStorage.getItem(NOTICE_SEEN_KEY) !== null;
    } catch {
        // Storage denied (private mode in some browsers): show the notice every
        // time rather than fail to start.
        return false;
    }
}

function rememberDemoNoticeSeen(): void {
    try {
        localStorage.setItem(NOTICE_SEEN_KEY, '1');
    } catch {
        // Nothing to do: the notice simply shows again next time.
    }
}

interface DemoBadgeProps {
    onClick: () => void;
}

/** The persistent marker in the top bar; clicking it reopens the notice. */
export function DemoBadge({ onClick }: DemoBadgeProps): VNode {
    return (
        <button class="demo-badge" onClick={onClick}
                title="About this demo" aria-label="About this demo">
            Demo
        </button>
    );
}

interface DemoDialogProps {
    onClose: () => void;
}

export function DemoDialog({ onClose }: DemoDialogProps): VNode {
    const dialogRef = useRef<HTMLDialogElement | null>(null);

    useEffect(() => {
        if (dialogRef.current && !dialogRef.current.open) {
            dialogRef.current.showModal();
        }
    }, []);

    function handleClose() {
        rememberDemoNoticeSeen();
        onClose();
    }

    return (
        <dialog ref={dialogRef} class="demo-dialog" onClose={handleClose}>
            <div class="dialog-header">
                <h2>This is a demo</h2>
                <button class="close-btn" onClick={handleClose}>&#xd7;</button>
            </div>
            <p>
                There is no server behind this calendar. A service worker in your browser
                answers the whole API, and everything you create, edit or delete is stored
                in this browser only &mdash; nothing is uploaded, and nobody else can see it.
            </p>
            <p>
                Your changes survive a reload, and clearing this site's data resets the
                demo to its starting content.
            </p>
            <p>
                A few features need a real server and are therefore not available here:
                importing and exporting iCalendar files, subscribing to calendar feeds,
                sharing an event by e-mail, and linking notes from MyNotes.
            </p>
            <p>
                Repeating events are missing too &mdash; the in-browser backend cannot
                expand a recurrence rule into its occurrences &mdash; so events here are
                created one at a time and the demo shows no repeat options.
            </p>
            <div class="dialog-actions">
                <button onClick={handleClose}>Explore the demo</button>
            </div>
        </dialog>
    );
}
