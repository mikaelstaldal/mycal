import type { VNode } from 'preact';
import { useRef, useEffect } from 'preact/hooks';
import { Icon } from './Icon.js';

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
    // The sentence MyMail and MyNotes show, with MyCal's noun. A bare "Demo"
    // said that this is a demo without saying the thing that actually matters,
    // which is where the user's data is going.
    //
    // Deliberately *not* role="status", which both of theirs carry. Theirs are
    // an inert <span> and <p>, where a live region is the right way to expose a
    // standing message. This one is a button — the only route back to
    // DemoDialog once the first-visit notice has been dismissed, since
    // demoNoticeSeen() persists that dismissal. role="status" would replace the
    // button role, so assistive tech would stop announcing it as operable; and
    // the text is not a live update anyway, it is there from first paint and
    // never changes.
    //
    // No aria-label either. It used to read "About this demo" over the visible
    // word "Demo" — a different string, but a short one nobody would try to
    // speak. Over a full sentence that same override breaks WCAG 2.5.3 (Label
    // in Name): a speech user says what they see and hits nothing. So the
    // visible sentence is the accessible name, and title carries what clicking
    // does. For a button, contents beat title in the name computation, so title
    // becomes the description rather than competing with the name.
    //
    // The clause is a span because MyCal's top bar cannot always show it —
    // .demo-badge-detail in app.css hides it below a breakpoint, and carries
    // the measurements that set where that breakpoint is. Hidden *visually
    // only*: the span stays in the accessibility tree, so the name is the full
    // sentence at every width and "Demo" is still contained in it, which is all
    // 2.5.3 asks.
    return (
        <button class="demo-badge" onClick={onClick} title="About this demo">
            Demo<span class="demo-badge-detail">&nbsp;&mdash; events are stored in this browser only</span>
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
                <button class="close-btn" onClick={handleClose} title="Close" aria-label="Close"><Icon name="x" /></button>
            </div>
            <p>
                There is no server behind this calendar. A service worker in your browser
                answers the whole API, and everything you create, edit or delete is stored
                in this browser only &mdash; nothing is uploaded, and nobody else can see it.
            </p>
            <p>
                It opens on a week of made-up sample events, so there is something to look
                at. Your changes survive a reload, and clearing this site's data resets the
                demo to that starting content.
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
