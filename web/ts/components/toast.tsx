import { h } from 'preact';
import type { VNode } from 'preact';
import { useEffect } from 'preact/hooks';

interface ToastProps {
    message: string;
    isError?: boolean;
    onDone: () => void;
}

export function Toast({ message, isError, onDone }: ToastProps): VNode | null {
    useEffect(() => {
        const id = setTimeout(onDone, isError ? 5000 : 3000);
        return () => clearTimeout(id);
    }, []);

    return (
        <div class={`toast${isError ? ' toast-error' : ''}`} onClick={onDone}>{message}</div>
    );
}
