/**
 * Custom confirm/choice dialog replacing browser's native confirm().
 * Returns a Promise that resolves based on user's choice.
 */

/**
 * Show a confirm dialog with OK/Cancel buttons.
 * @param {string} message - The message to display
 * @param {Object} [options]
 * @param {string} [options.title] - Dialog title (default: "Confirm")
 * @param {string} [options.okText] - OK button text (default: "OK")
 * @param {string} [options.cancelText] - Cancel button text (default: "Cancel")
 * @param {boolean} [options.danger] - Style OK button as danger
 * @returns {Promise<boolean>} true if confirmed, false if cancelled
 */
export function showConfirm(message, options = {}) {
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

        function cleanup(result) {
            dialog.close();
            dialog.remove();
            resolve(result);
        }

        dialog.querySelector('[data-action="ok"]').onclick = () => cleanup(true);
        dialog.querySelector('[data-action="cancel"]').onclick = () => cleanup(false);
        dialog.addEventListener('cancel', () => cleanup(false));

        document.body.appendChild(dialog);
        dialog.showModal();
        dialog.querySelector('[data-action="ok"]').focus();
    });
}

/**
 * Show a choice dialog with multiple buttons.
 * @param {string} message - The message to display
 * @param {Object} options
 * @param {string} [options.title] - Dialog title
 * @param {Array<{label: string, value: string, primary?: boolean}>} options.choices
 * @returns {Promise<string|null>} The chosen value, or null if cancelled
 */
export function showChoice(message, options = {}) {
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

        function cleanup(result) {
            dialog.close();
            dialog.remove();
            resolve(result);
        }

        choices.forEach(c => {
            dialog.querySelector(`[data-value="${c.value}"]`).onclick = () => cleanup(c.value);
        });
        dialog.addEventListener('cancel', () => cleanup(null));

        document.body.appendChild(dialog);
        dialog.showModal();
    });
}

function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}
