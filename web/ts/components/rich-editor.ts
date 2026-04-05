import type { VNode } from 'preact';
import { html } from 'htm/preact';
import { useEffect, useRef, useState } from 'preact/hooks';

let quillLoadPromise: Promise<void> | null = null;
let quillLoaded = false;

function loadQuill(): Promise<void> {
    if (quillLoaded) return Promise.resolve();
    if (quillLoadPromise) return quillLoadPromise;
    quillLoadPromise = new Promise((resolve, reject) => {
        // Load CSS
        const link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = 'vendor/quill.snow.css';
        document.head.appendChild(link);

        // Load JS
        const script = document.createElement('script');
        script.src = 'vendor/quill.js';
        script.async = true;
        script.onload = () => { quillLoaded = true; resolve(); };
        script.onerror = () => { quillLoadPromise = null; reject(new Error('Failed to load Quill')); };
        document.head.appendChild(script);
    });
    return quillLoadPromise;
}

const URL_RE = /^https?:\/\/\S+$/i;

const TOOLBAR_OPTIONS = [
    [{ header: [1, 2, 3, false] }],
    ['bold', 'italic', 'underline'],
    [{ list: 'ordered' }, { list: 'bullet' }],
    ['link'],
    ['clean'],
];

interface RichEditorProps {
    value: string;
    onChange?: (value: string) => void;
}

export function RichEditor({ value, onChange }: RichEditorProps): VNode | null {
    const containerRef = useRef<HTMLDivElement | null>(null);
    const quillRef = useRef<Quill | null>(null);
    const [loading, setLoading] = useState(true);
    const lastValueRef = useRef(value);

    useEffect(() => {
        let cancelled = false;
        setLoading(true);

        loadQuill().then(() => {
            if (cancelled || !containerRef.current) return;
            setLoading(false);

            const quill = new Quill(containerRef.current, {
                theme: 'snow',
                modules: { toolbar: TOOLBAR_OPTIONS },
                placeholder: 'Event description...',
            });
            quillRef.current = quill;

            // Set initial content
            if (value) {
                quill.root.innerHTML = value;
            }

            // Prevent toolbar buttons from submitting the parent form
            containerRef.current.closest('.ql-container')
                ?.previousElementSibling
                ?.querySelectorAll<HTMLButtonElement>('button')
                .forEach(b => b.type = 'button');

            // Paste URL over selected text to create a link.
            // Use capture phase + stopImmediatePropagation to intercept before
            // Quill's own clipboard handler replaces the selected text.
            quill.root.addEventListener('paste', (e: ClipboardEvent) => {
                const sel = quill.getSelection();
                if (!sel || sel.length === 0) return;
                const text = ((e.clipboardData || (window as any).clipboardData) as DataTransfer).getData('text/plain').trim();
                if (!URL_RE.test(text)) return;
                e.preventDefault();
                e.stopImmediatePropagation();
                quill.formatText(sel.index, sel.length, 'link', text);
            }, true);

            quill.on('text-change', () => {
                const content = quill.root.innerHTML;
                // Normalize Quill's empty state to empty string
                const normalized = content === '<p><br></p>' ? '' : content;
                lastValueRef.current = normalized;
                if (onChange) onChange(normalized);
            });
        });

        return () => {
            cancelled = true;
            quillRef.current = null;
        };
    }, []);

    // Sync external value changes into Quill (e.g. switching between events)
    useEffect(() => {
        if (!quillRef.current) return;
        if (value !== lastValueRef.current) {
            lastValueRef.current = value;
            quillRef.current.root.innerHTML = value || '';
        }
    }, [value]);

    return html`
        <div class="rich-editor-wrapper">
            ${loading && html`<div style="padding: 8px; color: #666; font-size: 0.85rem;">Loading editor...</div>`}
            <div ref=${containerRef} style="${loading ? 'display:none' : ''}" />
        </div>
    ` as VNode;
}
