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

/**
 * Show a confirm dialog with OK/Cancel buttons.
 * @param message - The message to display
 * @param options
 * @returns true if confirmed, false if cancelled
 */
export function showConfirm(message: string, options: ConfirmOptions = {}): Promise<boolean> {
    const { title = 'Confirm', okText = 'OK', cancelText = 'Cancel', danger = false } = options;

    return new Promise(resolve => {
        const dialog = document.createElement('dialog');
        dialog.className = 'confirm-dialog';

        const okBtnClass = danger ? 'confirm-btn confirm-btn-danger' : 'confirm-btn confirm-btn-primary';

        dialog.innerHTML = `
            <div class="confirm-dialog-body">
                <h3 class="confirm-dialog-title">${escapeHtml(title)}</h3>
                <p class="confirm-dialog-message">${escapeHtml(message)}</p>
                <div class="confirm-dialog-actions">
                    <button class="confirm-btn" data-action="cancel">${escapeHtml(cancelText)}</button>
                    <button class="${okBtnClass}" data-action="ok">${escapeHtml(okText)}</button>
                </div>
            </div>
        `;

        function cleanup(result: boolean) {
            dialog.close();
            dialog.remove();
            resolve(result);
        }

        (dialog.querySelector('[data-action="ok"]') as HTMLElement).onclick = () => cleanup(true);
        (dialog.querySelector('[data-action="cancel"]') as HTMLElement).onclick = () => cleanup(false);
        dialog.addEventListener('cancel', () => cleanup(false));

        document.body.appendChild(dialog);
        dialog.showModal();
        (dialog.querySelector('[data-action="ok"]') as HTMLElement).focus();
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
        const dialog = document.createElement('dialog');
        dialog.className = 'confirm-dialog';

        const buttonsHtml = choices.map(c => {
            const cls = c.primary ? 'confirm-btn confirm-btn-primary' : 'confirm-btn';
            return `<button class="${cls}" data-value="${escapeHtml(c.value)}">${escapeHtml(c.label)}</button>`;
        }).join('');

        dialog.innerHTML = `
            <div class="confirm-dialog-body">
                <h3 class="confirm-dialog-title">${escapeHtml(title)}</h3>
                <p class="confirm-dialog-message">${escapeHtml(message)}</p>
                <div class="confirm-dialog-actions">
                    ${buttonsHtml}
                </div>
            </div>
        `;

        function cleanup(result: string | null) {
            dialog.close();
            dialog.remove();
            resolve(result);
        }

        choices.forEach(c => {
            (dialog.querySelector(`[data-value="${c.value}"]`) as HTMLElement).onclick = () => cleanup(c.value);
        });
        dialog.addEventListener('cancel', () => cleanup(null));

        document.body.appendChild(dialog);
        dialog.showModal();
    });
}

function escapeHtml(str: string): string {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}
