/**
 * Custom confirm/choice dialog replacing browser's native confirm().
 * Returns a Promise that resolves based on user's choice.
 */

interface ConfirmOptions {
    title?: string;
    okText?: string;
    cancelText?: string;
    danger?: boolean;
}

interface ChoiceOption {
    label: string;
    value: string;
    primary?: boolean;
}

interface ChoiceOptions {
    title?: string;
    choices?: ChoiceOption[];
}

let dialogIdCounter = 0;

function buildDialogShell(titleText: string, messageText: string): {
    dialog: HTMLDialogElement;
    actionsDiv: HTMLDivElement;
} {
    const id = ++dialogIdCounter;
    const titleId = `confirm-dialog-title-${id}`;
    const msgId = `confirm-dialog-msg-${id}`;

    const dialog = document.createElement('dialog');
    dialog.className = 'confirm-dialog';
    dialog.setAttribute('aria-labelledby', titleId);
    dialog.setAttribute('aria-describedby', msgId);

    const body = document.createElement('div');
    body.className = 'confirm-dialog-body';

    const h3 = document.createElement('h3');
    h3.id = titleId;
    h3.className = 'confirm-dialog-title';
    h3.textContent = titleText;

    const p = document.createElement('p');
    p.id = msgId;
    p.className = 'confirm-dialog-message';
    p.textContent = messageText;

    const actionsDiv = document.createElement('div');
    actionsDiv.className = 'confirm-dialog-actions';

    body.appendChild(h3);
    body.appendChild(p);
    body.appendChild(actionsDiv);
    dialog.appendChild(body);

    return { dialog, actionsDiv };
}

/**
 * Show a confirm dialog with OK/Cancel buttons.
 * @param message - The message to display
 * @param options
 * @returns true if confirmed, false if cancelled
 */
export function showConfirm(message: string, options: ConfirmOptions = {}): Promise<boolean> {
    const { title = 'Confirm', okText = 'OK', cancelText = 'Cancel', danger = false } = options;

    return new Promise(resolve => {
        const { dialog, actionsDiv } = buildDialogShell(title, message);

        const cancelBtn = document.createElement('button');
        cancelBtn.className = 'confirm-btn';
        cancelBtn.textContent = cancelText;

        const okBtn = document.createElement('button');
        okBtn.className = danger ? 'confirm-btn confirm-btn-danger' : 'confirm-btn confirm-btn-primary';
        okBtn.textContent = okText;

        actionsDiv.appendChild(cancelBtn);
        actionsDiv.appendChild(okBtn);

        function cleanup(result: boolean) {
            dialog.close();
            dialog.remove();
            resolve(result);
        }

        okBtn.onclick = () => cleanup(true);
        cancelBtn.onclick = () => cleanup(false);
        dialog.addEventListener('cancel', () => cleanup(false));

        document.body.appendChild(dialog);
        dialog.showModal();
        okBtn.focus();
    });
}

/**
 * Show a choice dialog with multiple buttons.
 * @param message - The message to display
 * @param options
 * @returns The chosen value, or null if cancelled
 */
export function showChoice(message: string, options: ChoiceOptions = {}): Promise<string | null> {
    const { title = 'Choose', choices = [] } = options;

    return new Promise(resolve => {
        const { dialog, actionsDiv } = buildDialogShell(title, message);

        function cleanup(result: string | null) {
            dialog.close();
            dialog.remove();
            resolve(result);
        }

        choices.forEach(c => {
            const btn = document.createElement('button');
            btn.className = c.primary ? 'confirm-btn confirm-btn-primary' : 'confirm-btn';
            btn.textContent = c.label;
            btn.onclick = () => cleanup(c.value);
            actionsDiv.appendChild(btn);
        });

        dialog.addEventListener('cancel', () => cleanup(null));

        document.body.appendChild(dialog);
        dialog.showModal();
    });
}
