const SHORTCUT_KEYS = ['n', 't', 'y', 'm', 'w', 'd', 's', 'e', 'c', 'x'];

/**
 * Activate the same visible, enabled buttons used by pointer users, like MyMail.
 * A button opts in with `data-shortcut="<letter>"`. While a modal dialog is open,
 * only buttons inside it are candidates, so the same letter can mean one thing on
 * the calendar (D for Day) and another in the event dialog (D for Delete).
 */
export function handleActionShortcut(event: KeyboardEvent): void {
    if (event.defaultPrevented || event.repeat || event.isComposing
        || event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return;
    const key = event.key.toLowerCase();
    if (!SHORTCUT_KEYS.includes(key)) return;
    const target = event.target;
    if (target instanceof HTMLElement && (target.isContentEditable
        || target.closest('input, textarea, select, [contenteditable], [role="textbox"]'))) return;
    const scope = shortcutScope(target);
    if (!scope) return;
    const button = Array.from(scope.querySelectorAll<HTMLButtonElement>(`button[data-shortcut="${key}"]`))
        .find(button => !button.disabled && button.getClientRects().length > 0);
    if (!button) return;
    event.preventDefault();
    button.click();
}

// The open modal dialog that owns the keyboard, the whole document when there is
// none, or null when it is ambiguous which of several dialogs is in front.
function shortcutScope(target: EventTarget | null): ParentNode | null {
    if (document.querySelector('[aria-modal="true"]')) return null;
    const modals = Array.from(document.querySelectorAll<HTMLDialogElement>('dialog[open]'))
        .filter(dialog => dialog.matches(':modal'));
    if (modals.length === 0) return document;
    // Focus is normally inside the front dialog; when it has fallen back to the
    // body (its focused button was just removed), a lone dialog is still in front.
    const owner = target instanceof Element ? target.closest('dialog') : null;
    if (owner && modals.includes(owner as HTMLDialogElement)) return owner;
    return modals.length === 1 ? modals[0] : null;
}
