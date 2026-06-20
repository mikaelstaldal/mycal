import type { VNode } from 'preact';
import { useEffect, useRef } from 'preact/hooks';

let quillLoadPromise: Promise<void> | null = null;
let quillLoaded = false;

function loadQuill(): Promise<void> {
    if (quillLoaded) return Promise.resolve();
    if (quillLoadPromise) return quillLoadPromise;
    quillLoadPromise = new Promise((resolve, reject) => {
        const link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = 'vendor/quill.snow.css';
        document.head.appendChild(link);

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
    const wrapperRef = useRef<HTMLDivElement | null>(null);
    const quillRef = useRef<Quill | null>(null);
    const lastValueRef = useRef(value);

    useEffect(() => {
        let cancelled = false;
        const wrapper = wrapperRef.current!;

        // Loading indicator managed imperatively so Preact never reconciles
        // children it doesn't own (Quill inserts a sibling toolbar into the DOM).
        const loadingEl = document.createElement('div');
        loadingEl.className = 'rich-editor-loading';
        loadingEl.textContent = 'Loading editor...';
        wrapper.appendChild(loadingEl);

        loadQuill().then(() => {
            if (cancelled) return;

            loadingEl.remove();

            const container = document.createElement('div');
            wrapper.appendChild(container);

            const quill = new Quill(container, {
                theme: 'snow',
                modules: { toolbar: TOOLBAR_OPTIONS },
                placeholder: 'Event description...',
            });
            quillRef.current = quill;

            if (value) {
                quill.setContents(quill.clipboard.convert({ html: value }));
            }

            wrapper.querySelector('.ql-toolbar')
                ?.querySelectorAll<HTMLButtonElement>('button')
                .forEach(b => b.type = 'button');

            quill.root.addEventListener('paste', (e: ClipboardEvent) => {
                const sel = quill.getSelection();
                if (!sel || sel.length === 0) return;
                const text = ((e.clipboardData || (window as any).clipboardData) as DataTransfer).getData('text/plain').trim();
                if (!URL_RE.test(text)) return;
                e.preventDefault();
                e.stopImmediatePropagation();
                quill.formatText(sel.index, sel.length, 'link', text);
            }, true);

            quill.on('text-change', (_delta: any, _old: any, source: string) => {
                if (source !== 'user') return;
                const content = quill.root.innerHTML;
                const normalized = content === '<p><br></p>' ? '' : content;
                lastValueRef.current = normalized;
                if (onChange) onChange(normalized);
            });
        });

        return () => {
            cancelled = true;
            quillRef.current = null;
            while (wrapper.firstChild) wrapper.removeChild(wrapper.firstChild);
        };
    }, []);

    useEffect(() => {
        if (!quillRef.current) return;
        if (value !== lastValueRef.current) {
            lastValueRef.current = value;
            quillRef.current.setContents(quillRef.current.clipboard.convert({ html: value || '' }));
        }
    }, [value]);

    // No children in the virtual DOM — all children are managed imperatively
    // to prevent Preact from reconciling against Quill's DOM modifications.
    return <div ref={wrapperRef} class="rich-editor-wrapper" />;
}
